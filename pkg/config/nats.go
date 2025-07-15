package config

import (
	"fmt"
	"time"
)

type NATSConfig struct {
	Url     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

func (c *NATSConfig) Validate() error {
	if c.Url == "" {
		return fmt.Errorf("NATS URL is not configured")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("nats dial timeout is not configured")
	}
	return nil
}
