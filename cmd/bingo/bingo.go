package main

import (
	"fmt"
	"net/http"
	"os"
	"runtime"
	"time"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/n6g7/bingo/internal/config"
	"github.com/n6g7/bingo/internal/nameserver"
	"github.com/n6g7/bingo/internal/proxy"
	"github.com/n6g7/bingo/internal/reconcile"
	"github.com/n6g7/nomtail/pkg/log"
	"github.com/n6g7/nomtail/pkg/version"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	logger := log.SetupLogger()
	logger.Info("Bingo starting", "version", version.Display(), "go_runtime", runtime.Version())

	conf, err := config.Load()
	if err != nil {
		logger.Error("failed to load config", "err", err)
		os.Exit(1)
	}
	logger.Info("setting log level", "level", conf.LogLevel)
	log.SetLevel(conf.LogLevel)
	logger.Debug("loaded config", "config", conf)

	// Load proxy
	var prox proxy.Proxy

	switch conf.Proxy.Type {
	case config.Fabio:
		prox = proxy.NewFabioProxy(conf.Proxy.Fabio)
	case config.Traefik:
		prox = proxy.NewTraefikProxy(conf.Proxy.Traefik)
	default:
		logger.Error("unknown proxy type", "type", conf.Proxy.Type)
		os.Exit(1)
	}

	// Load nameserver
	var ns nameserver.Nameserver

	switch conf.Nameserver.Type {
	case config.Pihole:
		ns = nameserver.NewPiholeNS(logger, conf.Nameserver.Pihole)
	case config.Route53:
		ns = nameserver.NewRoute53NS(logger, conf.Nameserver.Route53)
	default:
		logger.Error("unknown nameserver type", "type", conf.Nameserver.Type)
		os.Exit(1)
	}

	go metrics(logger, conf)

	err = bingo(logger, ns, prox, conf)
	if err != nil {
		logger.Error("Bingo stopped with an error", "err", err)
		os.Exit(1)
	}
	return
}

func bingo(logger *log.Logger, ns nameserver.Nameserver, prox proxy.Proxy, conf *config.Config) error {
	reconciler := reconcile.NewReconciler(logger, ns, prox, conf)

	err := ns.Init()
	if err != nil {
		return fmt.Errorf("Nameserver backend initialization failed: %w", err)
	}
	logger.Info("initialized nameserver backend", "type", conf.Nameserver.Type)
	err = prox.Init()
	if err != nil {
		return fmt.Errorf("Proxy backend initialization failed: %w", err)
	}
	logger.Info("initialized proxy backend", "type", conf.Proxy.Type)

	go reconciler.Run()

	onNameserverTick := func() {
		records, err := ns.ListRecords()
		if err != nil {
			logger.Error("error loading records from nameserver", "err", err)
			return
		}
		newNSDomains := mapset.NewSet[string]()
		for _, record := range records {
			// We only manage service domains
			if !conf.IsServiceDomain(record.Name) {
				continue
			}

			newNSDomains.Add(record.Name)
			if !prox.IsValidTarget(record.Cname) {
				logger.Debug("domain points to invalid target, marking it for deletion.", "domain", record.Name, "target", record.Cname)
				reconciler.MarkForDeletion(record.Name)
			}
		}
		reconciler.SetNameserverDomains(newNSDomains)
	}

	onProxyTick := func() {
		services, err := prox.ListServices()
		if err != nil {
			logger.Error("error loading services from proxy", "err", err)
			return
		}
		newProxyDomains := mapset.NewSet[string]()
		for _, service := range services {
			// We only manage service domains
			if !conf.IsServiceDomain(service.Domain) {
				continue
			}

			newProxyDomains.Add(service.Domain)
		}
		reconciler.SetProxyDomains(newProxyDomains)
	}

	// Initial tick
	onNameserverTick()
	onProxyTick()

	// Main loop
	nameserverTick := time.Tick(conf.Nameserver.PollInterval)
	proxyTick := time.Tick(conf.Proxy.PollInterval)
	for {
		select {
		case <-nameserverTick:
			onNameserverTick()
		case <-proxyTick:
			onProxyTick()
		default:
			time.Sleep(conf.MainLoopTimeout)
		}
	}
}

func metrics(logger *log.Logger, conf *config.Config) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"healthy\": true}")
	})
	http.Handle(conf.Prometheus.MetricsPath, promhttp.Handler())

	logger.Info("starting prometheus exporter", "addr", conf.Prometheus.ListenAddr, "metrics_path", conf.Prometheus.MetricsPath)
	http.ListenAndServe(conf.Prometheus.ListenAddr, nil)
}
