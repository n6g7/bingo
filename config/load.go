package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetDefault("Proxy.Type", Fabio)
	viper.SetDefault("Proxy.Fabio.AdminPort", "9998")
	viper.SetDefault("Proxy.Fabio.Scheme", "http")
	viper.SetDefault("Nameserver.Type", Pihole)
	viper.SetDefault("LogLevel", "INFO")

	viper.BindEnv("Proxy.Type", "PROXY_TYPE")
	viper.BindEnv("Proxy.Fabio.Hosts", "FABIO_HOSTS")
	viper.BindEnv("Proxy.Fabio.AdminPort", "FABIO_ADMIN_PORT")
	viper.BindEnv("Proxy.Fabio.Scheme", "FABIO_SCHEME")
	viper.BindEnv("Nameserver.Type", "NAMESERVER_TYPE")
	viper.BindEnv("Nameserver.Pihole.URL", "PIHOLE_URL")
	viper.BindEnv("Nameserver.Pihole.Password", "PIHOLE_PASSWORD")
	viper.BindEnv("ServiceDomain", "SERVICE_DOMAIN")
	viper.BindEnv("Targets", "TARGETS")
	viper.BindEnv("LogLevel", "LOG_LEVEL")

	config := &Config{}
	err := viper.Unmarshal(config)
	if err != nil {
		return nil, fmt.Errorf("Couldn't parse config: %w", err)
	}

	// Manual fixes
	config.Targets = splitAndFilter(config.Targets[0])
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
