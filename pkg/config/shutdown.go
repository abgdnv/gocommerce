package config

import (
	"fmt"
	"time"
)

type ShutdownConfig struct {
	Timeout time.Duration `koanf:"timeout"`
}

func (c *ShutdownConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("shutdown timeout is not configured")
	}
	return nil
}
