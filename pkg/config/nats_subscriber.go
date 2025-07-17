package config

import (
	"fmt"
	"time"
)

type SubscriberConfig struct {
	Stream   string        `koanf:"stream"`
	Subject  string        `koanf:"subject"`
	Consumer string        `koanf:"consumer"`
	Timeout  time.Duration `koanf:"timeout"`
	Interval time.Duration `koanf:"interval"`
	Workers  int           `koanf:"workers"`
}

func (c *SubscriberConfig) Validate() error {
	if c.Stream == "" {
		return fmt.Errorf("SubscriberConfig: Stream is not configured")
	}
	if c.Subject == "" {
		return fmt.Errorf("SubscriberConfig: Subject is not configured")
	}
	if c.Consumer == "" {
		return fmt.Errorf("SubscriberConfig: consumer is not configured")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("SubscriberConfig: timeout must be greater than zero")
	}
	if c.Interval <= 0 {
		return fmt.Errorf("SubscriberConfig: interval must be greater than zero")
	}
	if c.Workers <= 0 {
		return fmt.Errorf("SubscriberConfig: workers must be greater than zero")
	}
	return nil
}
