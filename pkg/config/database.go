package config

import (
	"fmt"
	"strings"
	"time"
)

type DatabaseConfig struct {
	URL     string        `koanf:"url"`
	Timeout time.Duration `koanf:"timeout"`
}

func (c *DatabaseConfig) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("database URL is not configured")
	}
	if !isValidPostgresURL(c.URL) {
		return fmt.Errorf("database URL must start with 'postgres://': %s", c.URL)
	}
	return nil
}

// isValidPostgresURL checks if the provided URL is a valid PostgreSQL URL
func isValidPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") ||
		strings.HasPrefix(url, "postgresql://")
}
