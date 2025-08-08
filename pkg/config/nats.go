package config

import (
	"fmt"
	"strings"
	"time"
)

type NATSConfig struct {
	Url     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

// String returns a string representation of the NATS configuration.
func (c *NATSConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- NATS ---\n")
	b.WriteString(fmt.Sprintf("  url: %s\n", c.Url))
	b.WriteString(fmt.Sprintf("  timeout: %s\n", c.Timeout))
	return b.String()
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
