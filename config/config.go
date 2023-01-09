package config

import "time"

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
	Fabio ProxyType = "fabio"
)

type Proxy struct {
	Type         ProxyType
	PollInterval time.Duration
	Fabio        FabioConf
}

type FabioConf struct {
	Hosts     []string
	AdminPort uint16
	Scheme    string
}

// Nameserver

type NameserverType = string

const (
	Pihole NameserverType = "pihole"
)

type Nameserver struct {
	Type         NameserverType
	PollInterval time.Duration
	Pihole       PiholeConf
}

type PiholeConf struct {
	URL      string
	Password string
}

// Metrics

type Prometheus struct {
	ListenAddr  string
	MetricsPath string
}
