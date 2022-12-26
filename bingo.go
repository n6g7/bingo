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
	"github.com/n6g7/bingo/reconcile"
	"github.com/n6g7/bingo/registry"
)

var version = "0.0.1"

func main() {
	logOutput := logger.NewLevelWriter(os.Stderr, "INFO", "2017/01/01 00:00:00 ")
	log.SetOutput(logOutput)

	log.Printf("[INFO] Bingo v%s starting", version)
	log.Printf("[INFO] Go runtime is %s", runtime.Version())
	conf, err := config.Load()
	if err != nil {
		log.Fatalf("[FATAL] Failed to load config: %s", err)
	}
	log.Printf("[INFO] Setting log level to %s", conf.LogLevel)
	if !logOutput.SetLevel(conf.LogLevel) {
		log.Printf("[WARN] Cannot set log level to %s: %s", conf.LogLevel, err)
	}
	log.Printf("[DEBUG] Loaded config: %v", conf)

	// Load registry
	var reg registry.Registry

	switch conf.Registry.Type {
	case config.Consul:
		reg, err = registry.NewConsulRegistry(
			conf.Registry.Consul,
			conf.ServiceDomain,
		)
		if err != nil {
			log.Fatalf("[FATAL] Consul backend creation failed: %s", err)
		}
	default:
		log.Fatalf("[FATAL] Unknown registry type '%s'", conf.Registry.Type)
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

	err = bingo(ns, reg, conf)
	if err != nil {
		log.Fatal(err)
	}
	return
}

func bingo(ns nameserver.Nameserver, reg registry.Registry, conf *config.Config) error {
	reconciler := reconcile.NewReconciler(ns, 30*time.Second, conf.Targets)

	nsTick := time.Tick(1 * time.Minute)
	regTick := time.Tick(5 * time.Second)

	err := ns.Init()
	if err != nil {
		return fmt.Errorf("Nameserver backend initialization failed: %w", err)
	}
	err = reg.Init()
	if err != nil {
		return fmt.Errorf("Registry backend initialization failed: %w", err)
	}

	go reconciler.Run()

	for {
		select {
		case <-nsTick:
			records, err := ns.ListRecords()
			if err != nil {
				log.Printf("[ERROR] Error loading records from nameserver: %s", err)
			}
			newNSDomains := reconcile.NewDomainSet()
			for _, record := range records {
				if strings.HasSuffix(record.Name, conf.ServiceDomain) {
					newNSDomains.Add(record.Name)
				}
			}
			reconciler.SetNameserverDomains(newNSDomains)
		case <-regTick:
			services, err := reg.ListFabioServices()
			if err != nil {
				log.Printf("[ERROR] Error loading services from registry: %s", err)
			}
			newRegDomains := reconcile.NewDomainSet()
			for _, service := range services {
				newRegDomains.Add(service.Domain)
			}
			reconciler.SetRegistryDomains(newRegDomains)
		default:
			time.Sleep(100 * time.Millisecond)
		}
	}
}
