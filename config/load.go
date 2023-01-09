package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetDefault("Proxy.Type", Fabio)
	viper.SetDefault("Proxy.PollInterval", 5*time.Second)
	viper.SetDefault("Proxy.Fabio.AdminPort", "9998")
	viper.SetDefault("Proxy.Fabio.Scheme", "http")
	viper.SetDefault("Nameserver.Type", Pihole)
	viper.SetDefault("Nameserver.PollInterval", 30*time.Second)
	viper.SetDefault("LogLevel", "INFO")
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
	viper.BindEnv("Nameserver.Type", "NAMESERVER_TYPE")
	viper.BindEnv("Nameserver.PollInterval", "NAMESERVER_POLL_INTERVAL")
	viper.BindEnv("Nameserver.Pihole.URL", "PIHOLE_URL")
	viper.BindEnv("Nameserver.Pihole.Password", "PIHOLE_PASSWORD")
	viper.BindEnv("ServiceDomain", "SERVICE_DOMAIN")
	viper.BindEnv("LogLevel", "LOG_LEVEL")
	viper.BindEnv("MainLoopTimeout", "MAIN_LOOP_TIMEOUT")
	viper.BindEnv("ReconciliationTimeout", "RECONCILIATION_TIMEOUT")
	viper.BindEnv("ReconcilerLoopTimeout", "RECONCILER_LOOP_TIMEOUT")
	viper.BindEnv("Prometheus.ListenAddr", "PROMETHEUS_LISTEN_ADDR")
	viper.BindEnv("Prometheus.MetricsPath", "PROMETHEUS_METRICS_PATH")

	config := &Config{}
	err := viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse config: %w", err)
	}

	// Manual fixes
	config.Proxy.Fabio.Hosts = splitAndFilter(config.Proxy.Fabio.Hosts[0])

	return config, nil
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
