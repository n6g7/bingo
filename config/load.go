package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetDefault("Registry.Type", Fabio)
	viper.SetDefault("Registry.Fabio.AdminPort", "9998")
	viper.SetDefault("Registry.Fabio.Scheme", "http")
	viper.SetDefault("Nameserver.Type", Pihole)
	viper.SetDefault("LogLevel", "INFO")

	viper.BindEnv("Registry.Type", "REGISTRY_TYPE")
	viper.BindEnv("Registry.Fabio.Hosts", "FABIO_HOSTS")
	viper.BindEnv("Registry.Fabio.AdminPort", "FABIO_ADMIN_PORT")
	viper.BindEnv("Registry.Fabio.Scheme", "FABIO_SCHEME")
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
	config.Targets = strings.Split(config.Targets[0], " ")
	config.Registry.Fabio.Hosts = strings.Split(config.Registry.Fabio.Hosts[0], " ")

	return config, nil
}
