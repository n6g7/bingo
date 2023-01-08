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
	Fabio               = "fabio"
)

type Registry struct {
	Type   RegistryType
	Consul ConsulConf
	Fabio  FabioConf
}

type ConsulConf struct {
	Address string
	Scheme  string
	Token   string
	TLS     struct {
		CertFile           string
		KeyFile            string
		CAFile             string
		CAPath             string
		InsecureSkipVerify bool
	}
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
