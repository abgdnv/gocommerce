package config

import (
	"fmt"
	"strings"
	"time"
)

type DatabaseConfig struct {
	Host     string        `koanf:"host"`
	Port     int           `koanf:"port"`
	User     string        `koanf:"user"`
	Password string        `koanf:"password"`
	Name     string        `koanf:"name"`
	SSLMode  string        `koanf:"sslmode"`
	Timeout  time.Duration `koanf:"timeout"`
}

// URI constructs the PostgreSQL connection URI based on the configuration.
func (c *DatabaseConfig) URI() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%d/%s?sslmode=%s",
		c.User, c.Password, c.Host, c.Port, c.Name, c.SSLMode)
}

// String returns a string representation of the database configuration.
func (c *DatabaseConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Database ---\n")
	b.WriteString(fmt.Sprintf("  host: %s\n", c.Host))
	b.WriteString(fmt.Sprintf("  port: %d\n", c.Port))
	b.WriteString(fmt.Sprintf("  user: %s\n", c.User))
	b.WriteString(fmt.Sprintf("  name: %s\n", c.Name))
	b.WriteString(fmt.Sprintf("  sslmode: %s\n", c.SSLMode))
	b.WriteString(fmt.Sprintf("  timeout: %s\n", c.Timeout))
	return b.String()
}

func (c *DatabaseConfig) Validate() error {
	if c.Host == "" {
		return fmt.Errorf("database host is not configured")
	}
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("database port must be between 1 and 65535")
	}
	if c.User == "" {
		return fmt.Errorf("database user is not configured")
	}
	if c.Password == "" {
		return fmt.Errorf("database password is not configured")
	}
	if c.Name == "" {
		return fmt.Errorf("database name is not configured")
	}
	if c.SSLMode == "" {
		return fmt.Errorf("invalid SSL mode: %s", c.SSLMode)
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("database timeout must be greater than 0")
	}
	return nil
}
