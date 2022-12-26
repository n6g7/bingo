package config

type Config struct {
	Registry      Registry
	Nameserver    Nameserver
	ServiceDomain string
	Targets       []string
	LogLevel      string
}

// Registry

type RegistryType = string

const (
	Consul RegistryType = "consul"
)

type Registry struct {
	Type   RegistryType
	Consul ConsulConf
}

type ConsulConf struct {
	Address string
	Scheme  string
	Token   string
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
