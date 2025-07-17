package config

import (
	"fmt"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	Log        config.LogConfig        `koanf:"log"`
	PProf      config.PProfConfig      `koanf:"pprof"`
	Nats       config.NATSConfig       `koanf:"nats"`
	Subscriber config.SubscriberConfig `koanf:"subscriber"`
	Shutdown   config.ShutdownConfig   `koanf:"shutdown"`
}

func (c *Config) String() string {
	var b strings.Builder

	b.WriteString("\n--- External Services ---\n")
	b.WriteString(fmt.Sprintf("  nats.url: %s\n", c.Nats.Url))
	b.WriteString(fmt.Sprintf("  nats.timeout: %s\n", c.Nats.Timeout))

	b.WriteString("\n--- Subscriber ---\n")
	b.WriteString(fmt.Sprintf("  subscriber.stream: %s\n", c.Subscriber.Stream))
	b.WriteString(fmt.Sprintf("  subscriber.subject: %s\n", c.Subscriber.Subject))
	b.WriteString(fmt.Sprintf("  subscriber.consumer: %s\n", c.Subscriber.Consumer))
	b.WriteString(fmt.Sprintf("  subscriber.timeout: %s\n", c.Subscriber.Timeout))
	b.WriteString(fmt.Sprintf("  subscriber.interval: %s\n", c.Subscriber.Interval))
	b.WriteString(fmt.Sprintf("  subscriber.workers: %d\n", c.Subscriber.Workers))

	b.WriteString("\n--- Observability & Logging ---\n")
	b.WriteString(fmt.Sprintf("  log.level: %s\n", c.Log.Level))
	b.WriteString(fmt.Sprintf("  pprof.enabled: %t\n", c.PProf.Enabled))
	b.WriteString(fmt.Sprintf("  pprof.address: %s\n", c.PProf.Addr))

	b.WriteString("\n--- Application Behavior ---\n")
	b.WriteString(fmt.Sprintf("  shutdown.timeout: %s\n", c.Shutdown.Timeout))

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
	if err := c.Shutdown.Validate(); err != nil {
		return err
	}

	return nil
}
