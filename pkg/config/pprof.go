package config

import "fmt"

type PProfConfig struct {
	Enabled bool   `koanf:"enabled"`
	Addr    string `koanf:"addr"`
}

func (c *PProfConfig) Validate() error {
	if c.Enabled && c.Addr == "" {
		return fmt.Errorf("pprof is enabled but address is not configured")
	}
	return nil
}
