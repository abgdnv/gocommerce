package config

import (
	"fmt"
	"time"
)

type HTTPConfig struct {
	Port           int `koanf:"port"`
	MaxHeaderBytes int `koanf:"maxHeaderBytes"`
	Timeout        struct {
		Read       time.Duration `koanf:"read"`
		Write      time.Duration `koanf:"write"`
		Idle       time.Duration `koanf:"idle"`
		ReadHeader time.Duration `koanf:"readHeader"`
	} `koanf:"timeout"`
}

func (c *HTTPConfig) Validate() error {
	if c.Port <= 0 || c.Port > 65535 {
		return fmt.Errorf("invalid HTTP server port: %d", c.Port)
	}
	if c.Timeout.Read <= 0 {
		return fmt.Errorf("invalid HTTP server read timeout: %v", c.Timeout.Read)
	}
	if c.Timeout.Write <= 0 {
		return fmt.Errorf("invalid HTTP server write timeout: %v", c.Timeout.Write)
	}
	if c.Timeout.Idle <= 0 {
		return fmt.Errorf("invalid HTTP server idle timeout: %v", c.Timeout.Idle)
	}
	if c.Timeout.ReadHeader <= 0 {
		return fmt.Errorf("invalid HTTP server read header timeout: %v", c.Timeout.ReadHeader)
	}
	return nil
}
