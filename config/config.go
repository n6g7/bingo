package config

type Config struct {
	Proxy         Proxy
	Nameserver    Nameserver
	ServiceDomain string
	LogLevel      string
}

// Proxy

type ProxyType = string

const (
	Fabio ProxyType = "fabio"
)

type Proxy struct {
	Type  ProxyType
	Fabio FabioConf
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
	Type   NameserverType
	Pihole PiholeConf
}

type PiholeConf struct {
	URL      string
	Password string
}
