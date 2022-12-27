package registry

import (
	"fmt"
	"regexp"

	"github.com/hashicorp/consul/api"
	"github.com/n6g7/bingo/config"
)

const (
	tagFormat = "urlprefix-(([a-z.-]+).%s)(/[a-zA-Z0-9_/.-]*)?( |$)"
)

type ConsulRegistry struct {
	config        *api.Config
	client        *api.Client
	serviceDomain string
	tagRegex      *regexp.Regexp
}

func NewConsulRegistry(
	conf config.ConsulConf,
	serviceDomain string,
) (*ConsulRegistry, error) {
	config := api.DefaultConfig()
	config.Address = conf.Address
	config.Scheme = conf.Scheme
	config.Token = conf.Token
	if conf.Scheme == "https" {
		config.TLSConfig.KeyFile = conf.TLS.KeyFile
		config.TLSConfig.CertFile = conf.TLS.CertFile
		config.TLSConfig.CAFile = conf.TLS.CAFile
		config.TLSConfig.CAPath = conf.TLS.CAPath
		config.TLSConfig.InsecureSkipVerify = conf.TLS.InsecureSkipVerify
	}

	client, err := api.NewClient(config)

	if err != nil {
		return nil, fmt.Errorf("Consul client creation failed: %w", err)
	}

	return &ConsulRegistry{
		config,
		client,
		serviceDomain,
		regexp.MustCompile(fmt.Sprintf(tagFormat, serviceDomain)),
	}, nil
}

func (c *ConsulRegistry) Init() error {
	return nil
}

func (c *ConsulRegistry) ListFabioServices() ([]Service, error) {
	result, _, err := c.client.Catalog().Services(nil)
	if err != nil {
		return nil, fmt.Errorf("Error fetching services from Consul: %w", err)
	}
	services := []Service{}

	// Only return services with a tag matching the service domain
	for name, tags := range result {
		for _, tag := range tags {
			matches := c.tagRegex.FindStringSubmatch(tag)
			if len(matches) > 0 {
				services = append(services, Service{name, matches[1]})
			}
		}
	}
	return services, nil
}
