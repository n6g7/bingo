package proxy

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"regexp"
	"strings"

	mapset "github.com/deckarep/golang-set/v2"
	"github.com/n6g7/bingo/internal/config"
	"golang.org/x/exp/slices"
)

type TraefikProxy struct {
	hosts       []string
	adminPort   uint16
	scheme      string
	entryPoints mapset.Set[string]
	regexp      *regexp.Regexp
}

func NewTraefikProxy(conf config.TraefikConf) *TraefikProxy {
	return &TraefikProxy{
		hosts:       conf.Hosts,
		adminPort:   conf.AdminPort,
		scheme:      conf.Scheme,
		entryPoints: mapset.NewSet[string](conf.EntryPoints...),
	}
}

func (t *TraefikProxy) Init() error {
	re, err := regexp.Compile("^Host\\((`[a-z0-9.-]+`(, `[a-z0-9.-]+`)*)\\)$")
	if err != nil {
		return err
	}
	t.regexp = re

	// Test connection
	_, err = t.ListServices()
	if err != nil {
		return err
	}

	return nil
}

func (t *TraefikProxy) randomHost() string {
	return t.hosts[rand.Intn(len(t.hosts))]
}

type TraefikRouter struct {
	Name        string   `json:"name"`
	Provider    string   `json:"provider"`
	Status      string   `json:"status"`
	Rule        string   `json:"rule"`
	Service     string   `json:"service"`
	EntryPoints []string `json:"entryPoints"`
}

func (t *TraefikProxy) ListServices() ([]Service, error) {
	host := t.randomHost()
	port := fmt.Sprintf("%d", t.adminPort)
	url := t.scheme + "://" + host + ":" + port + "/api/http/routers"
	resp, err := http.Get(url)
	if err != nil {
		return nil, fmt.Errorf("error querying Traefik services: %w", err)
	}
	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("traefik returned an unexpected status code: %d", resp.StatusCode)
	}

	// Parse response body
	output := []TraefikRouter{}
	decoder := json.NewDecoder(resp.Body)
	err = decoder.Decode(&output)
	if err != nil {
		return nil, fmt.Errorf("error parsing Traefik routers body: %w", err)
	}

	services := []Service{}
	for _, router := range output {
		// Don't track disabled services
		if router.Status != "enabled" {
			continue
		}
		// Only track services on specified entrypoints
		inter := mapset.NewSet[string](router.EntryPoints...).Intersect(t.entryPoints)
		if inter.Cardinality() == 0 {
			continue
		}

		match := t.regexp.FindStringSubmatch(router.Rule)

		if len(match) < 2 {
			continue
		}

		items := strings.Split(match[1], ",")
		for _, item := range items {
			domain := strings.Trim(item, " `")
			services = append(services, Service{
				Name:   router.Service,
				Domain: domain,
			})
		}
	}

	return services, nil
}

func (t *TraefikProxy) GetTarget(sourceDomain string) string {
	return t.randomHost()
}

func (t *TraefikProxy) IsValidTarget(target string) bool {
	return slices.Contains(t.hosts, target)
}
