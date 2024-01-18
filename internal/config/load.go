package config

import (
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetDefault("Proxy.Type", Fabio)
	viper.SetDefault("Proxy.PollInterval", 5*time.Second)
	viper.SetDefault("Proxy.Fabio.AdminPort", "9998")
	viper.SetDefault("Proxy.Fabio.Scheme", "http")
	viper.SetDefault("Proxy.Traefik.AdminPort", "8080")
	viper.SetDefault("Proxy.Traefik.Scheme", "http")
	viper.SetDefault("Nameserver.Type", Pihole)
	viper.SetDefault("Nameserver.PollInterval", 30*time.Second)
	viper.SetDefault("Nameserver.Route53.TTL", 3600)
	viper.SetDefault("Nameserver.Route53.AWSRegion", "us-west-1")
	viper.SetDefault("LogLevel", slog.LevelInfo)
	viper.SetDefault("MainLoopTimeout", 1*time.Second)
	viper.SetDefault("ReconciliationTimeout", 30*time.Second)
	viper.SetDefault("ReconcilerLoopTimeout", 1*time.Second)
	viper.SetDefault("Prometheus.ListenAddr", ":9100")
	viper.SetDefault("Prometheus.MetricsPath", "/metrics")

	viper.BindEnv("Proxy.Type", "PROXY_TYPE")
	viper.BindEnv("Proxy.PollInterval", "PROXY_POLL_INTERVAL")
	viper.BindEnv("Proxy.Fabio.Hosts", "FABIO_HOSTS")
	viper.BindEnv("Proxy.Fabio.AdminPort", "FABIO_ADMIN_PORT")
	viper.BindEnv("Proxy.Fabio.Scheme", "FABIO_SCHEME")
	viper.BindEnv("Proxy.Traefik.Hosts", "TRAEFIK_HOSTS")
	viper.BindEnv("Proxy.Traefik.AdminPort", "TRAEFIK_ADMIN_PORT")
	viper.BindEnv("Proxy.Traefik.Scheme", "TRAEFIK_SCHEME")
	viper.BindEnv("Proxy.Traefik.EntryPoints", "TRAEFIK_ENTRYPOINTS")
	viper.BindEnv("Nameserver.Type", "NAMESERVER_TYPE")
	viper.BindEnv("Nameserver.PollInterval", "NAMESERVER_POLL_INTERVAL")
	viper.BindEnv("Nameserver.Pihole.URL", "PIHOLE_URL")
	viper.BindEnv("Nameserver.Pihole.Password", "PIHOLE_PASSWORD")
	viper.BindEnv("Nameserver.Route53.HostedZone", "ROUTE53_HOSTED_ZONE")
	viper.BindEnv("Nameserver.Route53.TTL", "ROUTE53_TTL")
	viper.BindEnv("Nameserver.Route53.AWSRegion", "AWS_REGION")
	viper.BindEnv("ServiceDomain", "SERVICE_DOMAIN")
	viper.BindEnv("LogLevel", "LOG_LEVEL")
	viper.BindEnv("MainLoopTimeout", "MAIN_LOOP_TIMEOUT")
	viper.BindEnv("ReconciliationTimeout", "RECONCILIATION_TIMEOUT")
	viper.BindEnv("ReconcilerLoopTimeout", "RECONCILER_LOOP_TIMEOUT")
	viper.BindEnv("Prometheus.ListenAddr", "PROMETHEUS_LISTEN_ADDR")
	viper.BindEnv("Prometheus.MetricsPath", "PROMETHEUS_METRICS_PATH")

	config := &Config{}
	err := viper.Unmarshal(config, viper.DecodeHook(mapstructure.TextUnmarshallerHookFunc()))
	if err != nil {
		return nil, fmt.Errorf("couldn't parse config: %w", err)
	}

	// Manual fixes
	if len(config.Proxy.Fabio.Hosts) > 0 {
		config.Proxy.Fabio.Hosts = splitAndFilter(config.Proxy.Fabio.Hosts[0])
	}
	if len(config.Proxy.Traefik.Hosts) > 0 {
		config.Proxy.Traefik.Hosts = splitAndFilter(config.Proxy.Traefik.Hosts[0])
	}
	if len(config.Proxy.Traefik.EntryPoints) > 0 {
		config.Proxy.Traefik.EntryPoints = splitAndFilter(config.Proxy.Traefik.EntryPoints[0])
	}

	return config, config.Validate()
}

func splitAndFilter(data string) (ret []string) {
	for _, val := range strings.Split(data, " ") {
		if val == "" {
			continue
		}
		ret = append(ret, val)
	}
	return
}
