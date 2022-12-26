package config

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

func Load() (*Config, error) {
	viper.SetDefault("Registry.Type", Consul)
	viper.SetDefault("Registry.Consul.Address", "localhost:8500")
	viper.SetDefault("Registry.Consul.Scheme", "http")
	viper.SetDefault("Nameserver.Type", Pihole)
	viper.SetDefault("LogLevel", "INFO")

	viper.BindEnv("Registry.Type", "REGISTRY_TYPE")
	viper.BindEnv("Registry.Consul.Address", "CONSUL_ADDRESS")
	viper.BindEnv("Registry.Consul.Scheme", "CONSUL_SCHEME")
	viper.BindEnv("Registry.Consul.Token", "CONSUL_TOKEN")
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

	return config, nil
}
