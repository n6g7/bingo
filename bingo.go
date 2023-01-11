package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fabiolb/fabio/logger"
	"github.com/n6g7/bingo/config"
	"github.com/n6g7/bingo/nameserver"
	"github.com/n6g7/bingo/proxy"
	"github.com/n6g7/bingo/reconcile"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var version = "dev"

func main() {
	logOutput := logger.NewLevelWriter(os.Stderr, "INFO", "2017/01/01 00:00:00 ")
	log.SetOutput(logOutput)

	var displayVersion string
	if strings.Contains(version, ".") {
		displayVersion = version
	} else if len(version) > 7 {
		displayVersion = version[:7]
	} else {
		displayVersion = version
	}
	log.Printf("[INFO] Bingo %s starting", displayVersion)
	log.Printf("[INFO] Go runtime is %s", runtime.Version())
	conf, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %s", err)
	}
	log.Printf("[INFO] Setting log level to %s", conf.LogLevel)
	if !logOutput.SetLevel(conf.LogLevel) {
		log.Printf("[WARN] Cannot set log level to %s: %s", conf.LogLevel, err)
	}
	log.Printf("[DEBUG] Loaded config: %+v", conf)

	// Load proxy
	var prox proxy.Proxy

	switch conf.Proxy.Type {
	case config.Fabio:
		prox = proxy.NewFabioProxy(conf.Proxy.Fabio)
	default:
		log.Fatalf("[FATAL] Unknown proxy type '%s'", conf.Proxy.Type)
	}

	// Load nameserver
	var ns nameserver.Nameserver

	switch conf.Nameserver.Type {
	case config.Pihole:
		ns = nameserver.NewPiholeNS(conf.Nameserver.Pihole)
	default:
		log.Fatalf("[FATAL] Unknown nameserver type '%s'", conf.Nameserver.Type)
	}

	go metrics(conf)

	err = bingo(ns, prox, conf)
	if err != nil {
		log.Fatalf("[FATAL] %s", err)
	}
	return
}

func bingo(ns nameserver.Nameserver, prox proxy.Proxy, conf *config.Config) error {
	reconciler := reconcile.NewReconciler(ns, prox, conf)

	err := ns.Init()
	if err != nil {
		return fmt.Errorf("Nameserver backend initialization failed: %w", err)
	}
	log.Printf("[INFO] Initialized '%s' backend", conf.Nameserver.Type)
	err = prox.Init()
	if err != nil {
		return fmt.Errorf("Proxy backend initialization failed: %w", err)
	}
	log.Printf("[INFO] Initialized '%s' backend", conf.Proxy.Type)

	go reconciler.Run()

	onNameserverTick := func() {
		records, err := ns.ListRecords()
		if err != nil {
			log.Printf("[ERROR] Error loading records from nameserver: %s", err)
			return
		}
		newNSDomains := reconcile.NewDomainSet()
		for _, record := range records {
			// We only manage service domains
			if !conf.IsServiceDomain(record.Name) {
				continue
			}

			newNSDomains.Add(record.Name)
			if !prox.IsValidTarget(record.Cname) {
				log.Printf("[DEBUG] Domain \"%s\" points to invalid target \"%s\", marking it for deletion.", record.Name, record.Cname)
				reconciler.MarkForDeletion(record.Name)
			}
		}
		reconciler.SetNameserverDomains(newNSDomains)
	}

	onProxyTick := func() {
		services, err := prox.ListServices()
		if err != nil {
			log.Printf("[ERROR] Error loading services from proxy: %s", err)
			return
		}
		newProxyDomains := reconcile.NewDomainSet()
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

func metrics(conf *config.Config) {
	http.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "{\"healthy\": true}")
	})
	http.Handle(conf.Prometheus.MetricsPath, promhttp.Handler())

	log.Printf("[INFO] Starting prometheus exporter at %s%s", conf.Prometheus.ListenAddr, conf.Prometheus.MetricsPath)
	http.ListenAndServe(conf.Prometheus.ListenAddr, nil)
}
