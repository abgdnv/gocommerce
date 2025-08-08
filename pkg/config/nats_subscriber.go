package config

import (
	"fmt"
	"strings"
	"time"
)

type SubscriberConfig struct {
	Stream   string        `koanf:"stream"`
	Subject  string        `koanf:"subject"`
	Consumer string        `koanf:"consumer"`
	Batch    int           `koanf:"batch"`
	Timeout  time.Duration `koanf:"timeout"`
	Interval time.Duration `koanf:"interval"`
	Workers  int           `koanf:"workers"`
}

// String returns a string representation of the NATS Subscriber configuration.
func (c *SubscriberConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- NATS Subscriber ---\n")
	b.WriteString(fmt.Sprintf("  stream: %s\n", c.Stream))
	b.WriteString(fmt.Sprintf("  subject: %s\n", c.Subject))
	b.WriteString(fmt.Sprintf("  consumer: %s\n", c.Consumer))
	b.WriteString(fmt.Sprintf("  batch: %d\n", c.Batch))
	b.WriteString(fmt.Sprintf("  timeout: %s\n", c.Timeout))
	b.WriteString(fmt.Sprintf("  interval: %s\n", c.Interval))
	b.WriteString(fmt.Sprintf("  workers: %d\n", c.Workers))
	return b.String()
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
	if c.Batch <= 0 {
		return fmt.Errorf("SubscriberConfig: batch must be greater than zero")
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
