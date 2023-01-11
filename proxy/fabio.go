package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"

	"github.com/n6g7/bingo/config"
	"golang.org/x/exp/slices"
)

type FabioProxy struct {
	hosts     []string
	adminPort uint16
	scheme    string
}

func NewFabioProxy(conf config.FabioConf) *FabioProxy {
	return &FabioProxy{
		hosts:     conf.Hosts,
		adminPort: conf.AdminPort,
		scheme:    conf.Scheme,
	}
}

func (f *FabioProxy) Init() error {
	// Test connection
	_, err := f.ListServices()
	if err != nil {
		return err
	}
	return nil
}

func (f *FabioProxy) randomHost() string {
	return f.hosts[rand.Intn(len(f.hosts))]
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

func (f *FabioProxy) ListServices() ([]Service, error) {
	host := f.randomHost()
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

func (f *FabioProxy) GetTarget(sourceDomain string) string {
	return f.randomHost()
}

func (f *FabioProxy) IsValidTarget(target string) bool {
	return slices.Contains(f.hosts, target)
}
