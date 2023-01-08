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
	Fabio RegistryType = "fabio"
)

type Registry struct {
	Type  RegistryType
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
