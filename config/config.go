package config

import (
	"strings"
	"time"
)

type Config struct {
	Proxy                 Proxy
	Nameserver            Nameserver
	ServiceDomain         string
	LogLevel              string
	MainLoopTimeout       time.Duration
	ReconciliationTimeout time.Duration
	ReconcilerLoopTimeout time.Duration
	Prometheus            Prometheus
}

// Proxy

type ProxyType = string

const (
	Fabio   ProxyType = "fabio"
	Traefik           = "traefik"
)

type Proxy struct {
	Type         ProxyType
	PollInterval time.Duration
	Fabio        FabioConf
	Traefik      TraefikConf
}

type FabioConf struct {
	Hosts     []string
	AdminPort uint16
	Scheme    string
}

type TraefikConf struct {
	Hosts       []string
	AdminPort   uint16
	Scheme      string
	EntryPoints []string
}

// Nameserver

type NameserverType = string

const (
	Pihole  NameserverType = "pihole"
	Route53                = "route53"
)

type Nameserver struct {
	Type         NameserverType
	PollInterval time.Duration
	Pihole       PiholeConf
	Route53      Route53Conf
}

type PiholeConf struct {
	URL      string
	Password string
}

type Route53Conf struct {
	HostedZone string
	TTL        int64
	AWSRegion  string
}

// Metrics

type Prometheus struct {
	ListenAddr  string
	MetricsPath string
}

func (c *Config) IsServiceDomain(domain string) bool {
	return strings.HasSuffix(domain, c.ServiceDomain)
}
