package config

import (
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	Log          config.LogConfig        `koanf:"log"`
	PProf        config.PProfConfig      `koanf:"pprof"`
	Nats         config.NATSConfig       `koanf:"nats"`
	Subscriber   config.SubscriberConfig `koanf:"subscriber"`
	ProbesConfig config.ProbesConfig     `koanf:"probes"`
	Shutdown     config.ShutdownConfig   `koanf:"shutdown"`
}

func (c *Config) String() string {
	var b strings.Builder
	b.WriteString(c.Nats.String())
	b.WriteString(c.Subscriber.String())
	b.WriteString(c.Log.String())
	b.WriteString(c.PProf.String())
	b.WriteString(c.ProbesConfig.String())
	b.WriteString(c.Shutdown.String())
	return b.String()
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.PProf.Validate(); err != nil {
		return err
	}
	if err := c.Nats.Validate(); err != nil {
		return err
	}
	if err := c.Subscriber.Validate(); err != nil {
		return err
	}
	if err := c.ProbesConfig.Validate(); err != nil {
		return err
	}
	if err := c.Shutdown.Validate(); err != nil {
		return err
	}

	return nil
}
