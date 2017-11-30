package main

import (
	"flag"
	"os"

	"fmt"
	"net/http"

	"github.com/nshttpd/mikrotik-exporter/collector"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/prometheus/common/version"
	log "github.com/sirupsen/logrus"
)

// single device can be defined via CLI flags, mutliple via config file.
var (
	device      = flag.String("device", "", "single device to monitor")
	address     = flag.String("address", "", "address of the device to monitor")
	user        = flag.String("user", "", "user for authentication with single device")
	password    = flag.String("password", "", "password for authentication for single device")
	cfgFile     = flag.String("config", "", "config file for multiple devices")
	logLevel    = flag.String("log-level", "info", "log level")
	logFormat   = flag.String("log-format", "json", "logformat text or json (default json)")
	port        = flag.String("port", ":9090", "port number to listen on")
	metricsPath = flag.String("path", "/metrics", "path to answer requests on")
	cfg         collector.Config
)

func init() {
	prometheus.MustRegister(version.NewCollector("mikrotik_exporter"))
}

func handler(w http.ResponseWriter, r *http.Request) {
	nc, err := collector.NewDeviceCollector(cfg)
	if err != nil {
		log.Warnln("Couldn't create", err)
		w.WriteHeader(http.StatusBadRequest)
		w.Write([]byte(fmt.Sprintf("Couldn't create %s", err)))
		return
	}

	registry := prometheus.NewRegistry()
	err = registry.Register(nc)
	if err != nil {
		log.Errorln("Couldn't register collector:", err)
		w.WriteHeader(http.StatusInternalServerError)
		w.Write([]byte(fmt.Sprintf("Couldn't register collector: %s", err)))
		return
	}

	gatherers := prometheus.Gatherers{
		prometheus.DefaultGatherer,
		registry,
	}
	// Delegate http serving to Prometheus client library, which will call collector.Collect.
	h := promhttp.HandlerFor(gatherers,
		promhttp.HandlerOpts{
			ErrorLog:      log.New(),
			ErrorHandling: promhttp.ContinueOnError,
		})
	h.ServeHTTP(w, r)
}

func main() {
	flag.Parse()

	// override default log level of info
	var ll log.Level
	var err error
	ll = log.InfoLevel
	if *logLevel != "info" {
		ll, err = log.ParseLevel(*logLevel)
		if err != nil {
			panic(err)
		}
	}
	log.SetLevel(ll)

	if *cfgFile == "" {
		if err := cfg.FromFlags(device, address, user, password); err != nil {
			log.WithFields(log.Fields{
				"error": err,
			}).Error("could not create configuration")
			os.Exit(1)
		}
	} else {
		log.Info("config file not supported yet")
		os.Exit(0)
	}

	http.HandleFunc(*metricsPath, prometheus.InstrumentHandlerFunc("prometheus", handler))
	http.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte(`<html>
			<head><title>Mikrotik Exporter</title></head>
			<body>
			<h1>Mikrotik Exporter</h1>
			<p><a href="` + *metricsPath + `">Metrics</a></p>
			</body>
			</html>`))
	})

	log.Info("Listening on", *port)
	err = http.ListenAndServe(*port, nil)
	if err != nil {
		log.Fatal(err)
	}

}
