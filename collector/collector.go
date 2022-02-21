package collector

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"mikrotik-exporter/config"

	"github.com/miekg/dns"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	routeros "gopkg.in/routeros.v2"
)

const (
	namespace  = "mikrotik"
	apiPort    = "8728"
	apiPortTLS = "8729"
	dnsPort    = 53

	// DefaultTimeout defines the default timeout when connecting to a router
	DefaultTimeout = 5 * time.Second
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"mikrotik_exporter: duration of a device collector scrape",
		[]string{"device"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"mikrotik_exporter: whether a device collector succeeded",
		[]string{"device"},
		nil,
	)
)

type collector struct {
	devices     []config.Device
	collectors  []routerOSCollector
	timeout     time.Duration
	enableTLS   bool
	insecureTLS bool
}

// WithBGP enables BGP routing metrics
func WithBGP() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newBGPCollector())
	}
}

// WithRoutes enables routing table metrics
func WithRoutes() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newRoutesCollector())
	}
}

// WithDHCP enables DHCP serrver metrics
func WithDHCP() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPCollector())
	}
}

// WithDHCPL enables DHCP server leases
func WithDHCPL() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPLCollector())
	}
}

// WithDHCPv6 enables DHCPv6 serrver metrics
func WithDHCPv6() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newDHCPv6Collector())
	}
}

// WithFirmware grab installed firmware and version
func WithFirmware() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newFirmwareCollector())
	}
}

// WithHealth enables board Health metrics
func WithHealth() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newhealthCollector())
	}
}

// WithPOE enables PoE metrics
func WithPOE() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newPOECollector())
	}
}

// WithPools enables IP(v6) pool metrics
func WithPools() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newPoolCollector())
	}
}

// WithOptics enables optical diagnstocs
func WithOptics() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newOpticsCollector())
	}
}

// WithW60G enables w60g metrics
func WithW60G() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, neww60gInterfaceCollector())
	}
}

// WithWlanSTA enables wlan STA metrics
func WithWlanSTA() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newWlanSTACollector())
	}
}

// WithWlanIF enables wireless interface metrics
func WithCapsman() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newCapsmanCollector())
	}
}

// WithWlanIF enables wireless interface metrics
func WithWlanIF() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newWlanIFCollector())
	}
}

// WithMonitor enables ethernet monitor collector metrics
func Monitor() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newMonitorCollector())
	}
}

// WithTimeout sets timeout for connecting to router
func WithTimeout(d time.Duration) Option {
	return func(c *collector) {
		c.timeout = d
	}
}

// WithTLS enables TLS
func WithTLS(insecure bool) Option {
	return func(c *collector) {
		c.enableTLS = true
		c.insecureTLS = insecure
	}
}

// WithIpsec enables ipsec metrics
func WithIpsec() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newIpsecCollector())
	}
}

// WithConntrack enables firewall/NAT connection tracking metrics
func WithConntrack() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newConntrackCollector())
	}
}

// WithLte enables lte metrics
func WithLte() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newLteCollector())
	}
}

// WithNetwatch enables netwatch metrics
func WithNetwatch() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newNetwatchCollector())
	}
}

// Option applies options to collector
type Option func(*collector)

// NewCollector creates a collector instance
func NewCollector(cfg *config.Config, opts ...Option) (prometheus.Collector, error) {
	log.WithFields(log.Fields{
		"numDevices": len(cfg.Devices),
	}).Info("setting up collector for devices")

	c := &collector{
		devices: cfg.Devices,
		timeout: DefaultTimeout,
		collectors: []routerOSCollector{
			newInterfaceCollector(),
			newResourceCollector(),
		},
	}

	for _, o := range opts {
		o(c)
	}

	return c, nil
}

// Describe implements the prometheus.Collector interface.
func (c *collector) Describe(ch chan<- *prometheus.Desc) {
	ch <- scrapeDurationDesc
	ch <- scrapeSuccessDesc

	for _, co := range c.collectors {
		co.describe(ch)
	}
}

// Collect implements the prometheus.Collector interface.
func (c *collector) Collect(ch chan<- prometheus.Metric) {
	wg := sync.WaitGroup{}

	var realDevices []config.Device

	for _, dev := range c.devices {
		if (config.SrvRecord{}) != dev.Srv {
			log.WithFields(log.Fields{
				"SRV": dev.Srv.Record,
			}).Info("SRV configuration detected")
			conf, _ := dns.ClientConfigFromFile("/etc/resolv.conf")
			dnsServer := net.JoinHostPort(conf.Servers[0], strconv.Itoa(dnsPort))
			if (config.DnsServer{}) != dev.Srv.Dns {
				dnsServer = net.JoinHostPort(dev.Srv.Dns.Address, strconv.Itoa(dev.Srv.Dns.Port))
				log.WithFields(log.Fields{
					"DnsServer": dnsServer,
				}).Info("Custom DNS config detected")
			}
			dnsMsg := new(dns.Msg)
			dnsCli := new(dns.Client)

			dnsMsg.RecursionDesired = true
			dnsMsg.SetQuestion(dns.Fqdn(dev.Srv.Record), dns.TypeSRV)
			r, _, err := dnsCli.Exchange(dnsMsg, dnsServer)

			if err != nil {
				os.Exit(1)
			}

			for _, k := range r.Answer {
				if s, ok := k.(*dns.SRV); ok {
					d := config.Device{}
					d.Name = strings.TrimRight(s.Target, ".")
					d.Address = strings.TrimRight(s.Target, ".")
					d.User = dev.User
					d.Password = dev.Password
					_ = c.getIdentity(&d)
					realDevices = append(realDevices, d)
				}
			}
		} else {
			realDevices = append(realDevices, dev)
		}
	}

	wg.Add(len(realDevices))

	for _, dev := range realDevices {
		go func(d config.Device) {
			c.collectForDevice(d, ch)
			wg.Done()
		}(dev)
	}

	wg.Wait()
}

func (c *collector) getIdentity(d *config.Device) error {
	cl, err := c.connect(d)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device fetching identity")
		return err
	}
	defer cl.Close()
	reply, err := cl.Run("/system/identity/print")
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error fetching ethernet interfaces")
		return err
	}
	for _, id := range reply.Re {
		d.Name = id.Map["name"]
	}
	return nil
}

func (c *collector) collectForDevice(d config.Device, ch chan<- prometheus.Metric) {
	begin := time.Now()

	err := c.connectAndCollect(&d, ch)

	duration := time.Since(begin)
	var success float64
	if err != nil {
		log.Errorf("ERROR: %s collector failed after %fs: %s", d.Name, duration.Seconds(), err)
		success = 0
	} else {
		log.Debugf("OK: %s collector succeeded after %fs.", d.Name, duration.Seconds())
		success = 1
	}

	ch <- prometheus.MustNewConstMetric(scrapeDurationDesc, prometheus.GaugeValue, duration.Seconds(), d.Name)
	ch <- prometheus.MustNewConstMetric(scrapeSuccessDesc, prometheus.GaugeValue, success, d.Name)
}

func (c *collector) connectAndCollect(d *config.Device, ch chan<- prometheus.Metric) error {
	cl, err := c.connect(d)
	if err != nil {
		log.WithFields(log.Fields{
			"device": d.Name,
			"error":  err,
		}).Error("error dialing device")
		return err
	}
	defer cl.Close()

	for _, co := range c.collectors {
		ctx := &collectorContext{ch, d, cl}
		err = co.collect(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *collector) connect(d *config.Device) (*routeros.Client, error) {
	var conn net.Conn
	var err error

	log.WithField("device", d.Name).Debug("trying to Dial")
	if !c.enableTLS {
		if (d.Port) == "" {
			d.Port = apiPort
		}
		conn, err = net.DialTimeout("tcp", d.Address+":"+d.Port, c.timeout)
		if err != nil {
			return nil, err
		}
		//		return routeros.DialTimeout(d.Address+apiPort, d.User, d.Password, c.timeout)
	} else {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.insecureTLS,
		}
		if (d.Port) == "" {
			d.Port = apiPortTLS
		}
		conn, err = tls.DialWithDialer(&net.Dialer{
			Timeout: c.timeout,
		},
			"tcp", d.Address+":"+d.Port, tlsCfg)
		if err != nil {
			return nil, err
		}
	}
	log.WithField("device", d.Name).Debug("done dialing")

	client, err := routeros.NewClient(conn)
	if err != nil {
		return nil, err
	}
	log.WithField("device", d.Name).Debug("got client")

	log.WithField("device", d.Name).Debug("trying to login")
	r, err := client.Run("/login", "=name="+d.User, "=password="+d.Password)
	if err != nil {
		return nil, err
	}
	ret, ok := r.Done.Map["ret"]
	if !ok {
		// Login method post-6.43 one stage, cleartext and no challenge
		if r.Done != nil {
			return client, nil
		}
		return nil, errors.New("RouterOS: /login: no ret (challenge) received")
	}

	// Login method pre-6.43 two stages, challenge
	b, err := hex.DecodeString(ret)
	if err != nil {
		return nil, fmt.Errorf("RouterOS: /login: invalid ret (challenge) hex string received: %s", err)
	}

	r, err = client.Run("/login", "=name="+d.User, "=response="+challengeResponse(b, d.Password))
	if err != nil {
		return nil, err
	}
	log.WithField("device", d.Name).Debug("done wth login")

	return client, nil

	//tlsCfg := &tls.Config{
	//	InsecureSkipVerify: c.insecureTLS,
	//}
	//	return routeros.DialTLSTimeout(d.Address+apiPortTLS, d.User, d.Password, tlsCfg, c.timeout)
}

func challengeResponse(cha []byte, password string) string {
	h := md5.New()
	h.Write([]byte{0})
	_, _ = io.WriteString(h, password)
	h.Write(cha)
	return fmt.Sprintf("00%x", h.Sum(nil))
}
