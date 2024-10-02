package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/openshift/pf-status-relay/pkg/log"
)

const (
	pfStatusRelayPollingInterval = "PF_STATUS_RELAY_POLLING_INTERVAL"
	pfStatusRelayInterfaces      = "PF_STATUS_RELAY_INTERFACES"
)

// Config contains the configuration of the application.
type Config struct {
	Interfaces      []string `yaml:"interfaces"`
	PollingInterval int      `yaml:"pollingInterval"`
}

// ReadConfig read yaml config file.
func ReadConfig() (Config, error) {
	c := Config{}

	c.PollingInterval = 1000
	raw, ok := os.LookupEnv(pfStatusRelayPollingInterval)
	if ok && raw != "" {
		pollingInterval, err := strconv.Atoi(raw)
		if err != nil {
			return c, fmt.Errorf("failed to convert polling interval to int: %w", err)
		}

		if pollingInterval < 100 {
			return c, fmt.Errorf("polling interval must be greater than 100 - current value: %d", pollingInterval)
		}

		c.PollingInterval = pollingInterval
	}

	raw, ok = os.LookupEnv(pfStatusRelayInterfaces)
	if !ok || raw == "" {
		return c, fmt.Errorf("interfaces must be set")
	}

	pfs := strings.Split(raw, ",")
	for i := range pfs {
		pf := strings.TrimSpace(pfs[i])
		if pf != "" {
			c.Interfaces = append(c.Interfaces, pf)
		}
	}

	log.Log.Info("interfaces to monitor", "interfaces", c.Interfaces)

	return c, nil
}
