package exporter

import (
	"net"
	"net/http"
	"time"
)

type Server struct {
	l net.Listener
}

func runCollector(cfg Config) {
	cfg.Logger.Info("starting collector")

	for {
		for _, d := range cfg.Devices {
			d.CollectMetrics(cfg.Metrics, cfg.Logger)
		}
		time.Sleep(15 * time.Second)
	}

}

func (s *Server) Run(cfg Config, mh http.Handler, port *string) error {

	cfg.Logger.Infow("starting server",
		"port", *port,
	)

	var err error
	s.l, err = net.Listen("tcp", *port)
	if err != nil {
		cfg.Logger.Errorw("error creating listener",
			"port", *port,
			"error", err,
		)
		return err
	}

	go func() {
		runCollector(cfg)
	}()

	mux := http.NewServeMux()
	mux.Handle("/metrics", mh)
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("ok"))
	})

	go func() {
		if err := http.Serve(s.l, mux); err != nil {
			cfg.Logger.Errorw("unable to start service",
				"error", err,
			)
		}
	}()

	return nil
}

func (s *Server) Stop() error {
	return s.l.Close()
}
