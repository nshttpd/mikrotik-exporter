package main

import (
	"flag"
	"os"
	"os/signal"

	"github.com/nshttpd/mikrotik-exporter/exporter"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
)

// single device can be defined via CLI flags, mutliple via config file.
var (
	device          = flag.String("device", "", "single device to monitor")
	address         = flag.String("address", "", "address of the device to monitor")
	user            = flag.String("user", "", "user for authentication with single device")
	password        = flag.String("password", "", "password for authentication for single device")
	cfgFile         = flag.String("config", "", "config file for multiple devices")
	logLevel        = flag.String("log-level", "info", "log level")
	port            = flag.String("port", ":9090", "port number to listen on")
	currentLogLevel = zap.NewAtomicLevelAt(zap.InfoLevel)
)

// (nshttpd) TODO figure out if we need a caching option

func main() {
	flag.Parse()

	// override default log level of info
	if *logLevel != "info" {
		err := currentLogLevel.UnmarshalText([]byte(*logLevel))
		if err != nil {
			panic(err)
		}
	}

	// setup logger
	l, err := newLogger(currentLogLevel)
	if err != nil {
		panic(err)
	}
	defer l.Sync()

	var cfg exporter.Config
	if *cfgFile == "" {
		if err := cfg.FromFlags(device, address, user, password); err != nil {
			l.Sugar().Errorw("could not create configuration",
				"error", err,
			)
			return
		}
	} else {
		l.Sugar().Info("config file not supported yet")
		os.Exit(0)
	}

	cfg.Logger = l.Sugar()

	cfg.Metrics = exporter.PromMetrics{}
	mh, err := cfg.Metrics.SetupPrometheus(*cfg.Logger)
	if err != nil {
		log.Fatal(err)
	}

	s := &exporter.Server{}

	if err := s.Run(cfg, mh, port); err != nil {
		log.Fatal(err)
	}

	sigchan := make(chan os.Signal, 1)
	signal.Notify(sigchan, os.Interrupt, os.Kill)
	<-sigchan
	cfg.Logger.Info("stopping server")
	err = s.Stop()
	if err != nil {
		cfg.Logger.Errorw("error while stopping service",
			"error", err,
		)
		os.Exit(1)
	}

	os.Exit(0)

}

func newLogger(lvl zap.AtomicLevel) (*zap.Logger, error) {
	config := zap.NewProductionConfig()
	config.Level = lvl
	return config.Build()
}
