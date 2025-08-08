package config

import (
	"fmt"
	"strings"
)

type PProfConfig struct {
	Enabled bool   `koanf:"enabled"`
	Addr    string `koanf:"addr"`
}

// String returns a string representation of the pprof configuration.
func (c *PProfConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- PProf ---\n")
	b.WriteString(fmt.Sprintf("  enabled: %t\n", c.Enabled))
	b.WriteString(fmt.Sprintf("  address: %s\n", c.Addr))
	return b.String()
}

func (c *PProfConfig) Validate() error {
	if c.Enabled && c.Addr == "" {
		return fmt.Errorf("pprof is enabled but address is not configured")
	}
	return nil
}
