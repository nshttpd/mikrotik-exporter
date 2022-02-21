package main

import (
	"bytes"
	"flag"
	"io/ioutil"
	"os"

	"github.com/prometheus/common/version"

	"fmt"
	"net/http"

	"mikrotik-exporter/collector"
	"mikrotik-exporter/config"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	log "github.com/sirupsen/logrus"
)

// single device can be defined via CLI flags, multiple via config file.
var (
	address     = flag.String("address", "", "address of the device to monitor")
	configFile  = flag.String("config-file", "", "config file to load")
	device      = flag.String("device", "", "single device to monitor")
	insecure    = flag.Bool("insecure", false, "skips verification of server certificate when using TLS (not recommended)")
	logFormat   = flag.String("log-format", "json", "logformat text or json (default json)")
	logLevel    = flag.String("log-level", "info", "log level")
	metricsPath = flag.String("path", "/metrics", "path to answer requests on")
	password    = flag.String("password", "", "password for authentication for single device")
	deviceport  = flag.String("deviceport", "8728", "port for single device")
	port        = flag.String("port", ":9436", "port number to listen on")
	timeout     = flag.Duration("timeout", collector.DefaultTimeout, "timeout when connecting to devices")
	tls         = flag.Bool("tls", false, "use tls to connect to routers")
	user        = flag.String("user", "", "user for authentication with single device")
	ver         = flag.Bool("version", false, "find the version of binary")

	withBgp       = flag.Bool("with-bgp", false, "retrieves BGP routing infrormation")
	withConntrack = flag.Bool("with-conntrack", false, "retrieves connection tracking metrics")
	withRoutes    = flag.Bool("with-routes", false, "retrieves routing table information")
	withDHCP      = flag.Bool("with-dhcp", false, "retrieves DHCP server metrics")
	withDHCPL     = flag.Bool("with-dhcpl", false, "retrieves DHCP server lease metrics")
	withDHCPv6    = flag.Bool("with-dhcpv6", false, "retrieves DHCPv6 server metrics")
	withFirmware  = flag.Bool("with-firmware", false, "retrieves firmware versions")
	withHealth    = flag.Bool("with-health", false, "retrieves board Health metrics")
	withPOE       = flag.Bool("with-poe", false, "retrieves PoE metrics")
	withPools     = flag.Bool("with-pools", false, "retrieves IP(v6) pool metrics")
	withOptics    = flag.Bool("with-optics", false, "retrieves optical diagnostic metrics")
	withW60G      = flag.Bool("with-w60g", false, "retrieves w60g interface metrics")
	withWlanSTA   = flag.Bool("with-wlansta", false, "retrieves connected wlan station metrics")
	withWlanIF    = flag.Bool("with-wlanif", false, "retrieves wlan interface metrics")
	withCapsman   = flag.Bool("with-capsman", false, "retrieves capsman station metrics")
	withMonitor   = flag.Bool("with-monitor", false, "retrieves ethernet interface monitor info")
	withIpsec     = flag.Bool("with-ipsec", false, "retrieves ipsec metrics")
	withLte       = flag.Bool("with-lte", false, "retrieves lte metrics")
	withNetwatch  = flag.Bool("with-netwatch", false, "retrieves netwatch metrics")

	cfg *config.Config

	appVersion = "DEVELOPMENT"
	shortSha   = "0xDEADBEEF"
)

func init() {
	prometheus.MustRegister(version.NewCollector("mikrotik_exporter"))
}

func main() {
	flag.Parse()

	if *ver {
		fmt.Printf("\nVersion:   %s\nShort SHA: %s\n\n", appVersion, shortSha)
		os.Exit(0)
	}

	configureLog()

	c, err := loadConfig()
	if err != nil {
		log.Errorf("Could not load config: %v", err)
		os.Exit(3)
	}
	cfg = c

	startServer()
}

func configureLog() {
	ll, err := log.ParseLevel(*logLevel)
	if err != nil {
		panic(err)
	}

	log.SetLevel(ll)

	if *logFormat == "text" {
		log.SetFormatter(&log.TextFormatter{})
	} else {
		log.SetFormatter(&log.JSONFormatter{})
	}
}

func loadConfig() (*config.Config, error) {
	if *configFile != "" {
		return loadConfigFromFile()
	}

	return loadConfigFromFlags()
}

func loadConfigFromFile() (*config.Config, error) {
	b, err := ioutil.ReadFile(*configFile)
	if err != nil {
		return nil, err
	}

	return config.Load(bytes.NewReader(b))
}

func loadConfigFromFlags() (*config.Config, error) {
	// Attempt to read credentials from env if not already defined
	if *user == "" {
		*user = os.Getenv("MIKROTIK_USER")
	}
	if *password == "" {
		*password = os.Getenv("MIKROTIK_PASSWORD")
	}
	if *device == "" || *address == "" || *user == "" || *password == "" {
		return nil, fmt.Errorf("missing required param for single device configuration")
	}

	return &config.Config{
		Devices: []config.Device{
			config.Device{
				Name:     *device,
				Address:  *address,
				User:     *user,
				Password: *password,
				Port:     *deviceport,
			},
		},
	}, nil
}

func startServer() {
	h, err := createMetricsHandler()
	if err != nil {
		log.Fatal(err)
	}
	http.Handle(*metricsPath, h)

	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	})

	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`<html>
			<head><title>Mikrotik Exporter</title></head>
			<body>
			<h1>Mikrotik Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Info("Listening on ", *port)
	log.Fatal(http.ListenAndServe(*port, nil))
}

func createMetricsHandler() (http.Handler, error) {
	opts := collectorOptions()
	nc, err := collector.NewCollector(cfg, opts...)
	if err != nil {
		return nil, err
	}

	promhttp.Handler()

	registry := prometheus.NewRegistry()
	err = registry.Register(prometheus.NewGoCollector())
	if err != nil {
		return nil, err
	}
	err = registry.Register(nc)
	if err != nil {
		return nil, err
	}

	return promhttp.HandlerFor(registry,
		promhttp.HandlerOpts{
			ErrorLog:      log.New(),
			ErrorHandling: promhttp.ContinueOnError,
		}), nil
}

func collectorOptions() []collector.Option {
	opts := []collector.Option{}

	if *withBgp || cfg.Features.BGP {
		opts = append(opts, collector.WithBGP())
	}

	if *withRoutes || cfg.Features.Routes {
		opts = append(opts, collector.WithRoutes())
	}

	if *withDHCP || cfg.Features.DHCP {
		opts = append(opts, collector.WithDHCP())
	}

	if *withDHCPL || cfg.Features.DHCPL {
		opts = append(opts, collector.WithDHCPL())
	}

	if *withDHCPv6 || cfg.Features.DHCPv6 {
		opts = append(opts, collector.WithDHCPv6())
	}

	if *withFirmware || cfg.Features.Firmware {
		opts = append(opts, collector.WithFirmware())
	}

	if *withHealth || cfg.Features.Health {
		opts = append(opts, collector.WithHealth())
	}

	if *withPOE || cfg.Features.POE {
		opts = append(opts, collector.WithPOE())
	}

	if *withPools || cfg.Features.Pools {
		opts = append(opts, collector.WithPools())
	}

	if *withOptics || cfg.Features.Optics {
		opts = append(opts, collector.WithOptics())
	}

	if *withW60G || cfg.Features.W60G {
		opts = append(opts, collector.WithW60G())
	}

	if *withWlanSTA || cfg.Features.WlanSTA {
		opts = append(opts, collector.WithWlanSTA())
	}

	if *withCapsman || cfg.Features.Capsman {
		opts = append(opts, collector.WithCapsman())
	}

	if *withWlanIF || cfg.Features.WlanIF {
		opts = append(opts, collector.WithWlanIF())
	}

	if *withMonitor || cfg.Features.Monitor {
		opts = append(opts, collector.Monitor())

	}

	if *withIpsec || cfg.Features.Ipsec {
		opts = append(opts, collector.WithIpsec())
	}

	if *withConntrack || cfg.Features.Conntrack {
		opts = append(opts, collector.WithConntrack())
	}

	if *withLte || cfg.Features.Lte {
		opts = append(opts, collector.WithLte())
	}

	if *withNetwatch || cfg.Features.Netwatch {
		opts = append(opts, collector.WithNetwatch())
	}

	if *timeout != collector.DefaultTimeout {
		opts = append(opts, collector.WithTimeout(*timeout))
	}

	if *tls {
		opts = append(opts, collector.WithTLS(*insecure))
	}

	return opts
}
