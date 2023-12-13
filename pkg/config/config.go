package config

import (
	"os"

	"gopkg.in/yaml.v2"

	"github.com/mlguerrero12/pf-status-relay/pkg/log"
)

var path = "/etc/pf-status-relay/config.yaml"

// Config contains the configuration of the application.
type Config struct {
	Interfaces      []string `yaml:"interfaces"`
	PollingInterval int      `yaml:"pollingInterval"`
}

// ReadConfig read yaml config file.
func ReadConfig() Config {
	data, err := os.ReadFile(path)
	if err != nil {
		log.Log.Error("failed to read config file", "error", err)
		os.Exit(1)
	}

	// Default values.
	c := Config{
		PollingInterval: 1000,
	}

	err = yaml.Unmarshal(data, &c)
	if err != nil {
		log.Log.Error("failed to unmarshall config file", "error", err)
		os.Exit(1)
	}

	if c.Interfaces == nil {
		log.Log.Error("failed to parse config file", "error", "no interfaces provided")
		os.Exit(1)
	}

	if c.PollingInterval < 100 {
		log.Log.Error("failed to parse config file", "error", "polling interval must be greater or equal to 100")
		os.Exit(1)
	}

	return c
}
