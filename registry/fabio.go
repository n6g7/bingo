package registry

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/n6g7/bingo/config"
)

type FabioRegistry struct {
	hosts         []string
	adminPort     uint16
	scheme        string
	serviceDomain string
}

func NewFabioRegistry(
	conf config.FabioConf,
	serviceDomain string,
) (*FabioRegistry, error) {
	return &FabioRegistry{
		conf.Hosts,
		conf.AdminPort,
		conf.Scheme,
		serviceDomain,
	}, nil
}

func (f *FabioRegistry) Init() error {
	// Test connection
	_, err := f.ListServices()
	if err != nil {
		return err
	}
	return nil
}

type ResultService struct {
	Service string `json:"service"`
	Host    string `json:"host"`
	Path    string `json:"path"`
	Src     string `json:"src"`
	Dst     string `json:"dst"`
	Opts    string `json:"opts"`
	Weight  uint   `json:"weight"`
	Cmd     string `json:"cmd"`
	Rate1   uint   `json:"rate1"`
	Pct99   uint   `json:"pct99"`
}

func (f *FabioRegistry) ListServices() ([]Service, error) {
	host := f.hosts[rand.Intn(len(f.hosts))]
	port := fmt.Sprintf("%d", f.adminPort)
	url := f.scheme + "://" + host + ":" + port + "/api/routes"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("Error querying Fabio routes: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("Fabio returned an unexpected status code: %d", resp.StatusCode)
	}

	// Parse response body
	output := []ResultService{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("Error parsing Fabio routes body: %w", err)
	}

	services := []Service{}
	for _, service := range output {
		if service.Host == "" {
			continue
		}
		services = append(services, Service{
			Name:   service.Service,
			Domain: service.Host,
		})
	}

	return services, nil
}
