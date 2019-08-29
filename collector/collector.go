package collector

import (
	"crypto/md5"
	"crypto/tls"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/nshttpd/mikrotik-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	routeros "gopkg.in/routeros.v2"
)

const (
	namespace  = "mikrotik"
	apiPort    = ":8728"
	apiPortTLS = ":8729"

	// DefaultTimeout defines the default timeout when connecting to a router
	DefaultTimeout = 5 * time.Second
)

var (
	scrapeDurationDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_duration_seconds"),
		"mikrotik_exporter: duration of a collector scrape",
		[]string{"device"},
		nil,
	)
	scrapeSuccessDesc = prometheus.NewDesc(
		prometheus.BuildFQName(namespace, "scrape", "collector_success"),
		"mikrotik_exporter: whether a collector succeeded",
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

// WithWlanSTA enables wlan STA metrics
func WithWlanSTA() Option {
	return func(c *collector) {
		c.collectors = append(c.collectors, newWlanSTACollector())
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
		c.insecureTLS = true
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
	wg.Add(len(c.devices))

	for _, dev := range c.devices {
		go func(d config.Device) {
			c.collectForDevice(d, ch)
			wg.Done()
		}(dev)
	}

	wg.Wait()
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
		conn, err = net.Dial("tcp", d.Address+apiPort)
		if err != nil {
			return nil, err
		}
		//		return routeros.DialTimeout(d.Address+apiPort, d.User, d.Password, c.timeout)
	} else {
		tlsCfg := &tls.Config{
			InsecureSkipVerify: c.insecureTLS,
		}
		conn, err = tls.Dial("tcp", d.Address+apiPortTLS, tlsCfg)
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
