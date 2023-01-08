package main

import (
	"fmt"
	"log"
	"os"
	"runtime"
	"strings"
	"time"

	"github.com/fabiolb/fabio/logger"
	"github.com/n6g7/bingo/config"
	"github.com/n6g7/bingo/nameserver"
	"github.com/n6g7/bingo/proxy"
	"github.com/n6g7/bingo/reconcile"
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
		prox, err = proxy.NewFabioProxy(conf.Proxy.Fabio, conf.ServiceDomain)
		if err != nil {
			log.Fatalf("[FATAL] Fabio backend creation failed: %s", err)
		}
	default:
		log.Fatalf("[FATAL] Unknown proxy type '%s'", conf.Proxy.Type)
	}

	// Load nameserver
	var ns nameserver.Nameserver

	switch conf.Nameserver.Type {
	case config.Pihole:
		ns, err = nameserver.NewPiholeNS(
			conf.Nameserver.Pihole,
			conf.ServiceDomain,
		)
		if err != nil {
			log.Fatalf("[FATAL] Pihole backend creation failed: %s", err)
		}
	default:
		log.Fatalf("[FATAL] Unknown nameserver type '%s'", conf.Nameserver.Type)
	}

	err = bingo(ns, prox, conf)
	if err != nil {
		log.Fatalf("[FATAL] %s", err)
	}
	return
}

func bingo(ns nameserver.Nameserver, prox proxy.Proxy, conf *config.Config) error {
	reconciler := reconcile.NewReconciler(ns, 30*time.Second, conf.Targets)

	nsTick := time.Tick(1 * time.Minute)
	proxyTick := time.Tick(5 * time.Second)

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

	for {
		select {
		case <-nsTick:
			records, err := ns.ListRecords()
			if err != nil {
				log.Printf("[ERROR] Error loading records from nameserver: %s", err)
				continue
			}
			newNSDomains := reconcile.NewDomainSet()
			for _, record := range records {
				if strings.HasSuffix(record.Name, conf.ServiceDomain) {
					newNSDomains.Add(record.Name)
				}
			}
			reconciler.SetNameserverDomains(newNSDomains)
		case <-proxyTick:
			services, err := prox.ListServices()
			if err != nil {
				log.Printf("[ERROR] Error loading services from proxy: %s", err)
				continue
			}
			newProxyDomains := reconcile.NewDomainSet()
			for _, service := range services {
				newProxyDomains.Add(service.Domain)
			}
			reconciler.SetProxyDomains(newProxyDomains)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
