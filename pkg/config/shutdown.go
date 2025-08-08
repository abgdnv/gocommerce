package config

import (
	"fmt"
	"strings"
	"time"
)

type ShutdownConfig struct {
	Timeout time.Duration `koanf:"timeout"`
}

// String returns a string representation of the ShutdownConfig.
func (c *ShutdownConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Shutdown ---\n")
	b.WriteString(fmt.Sprintf("  timeout: %s\n", c.Timeout))
	return b.String()
}

func (c *ShutdownConfig) Validate() error {
	if c.Timeout <= 0 {
		return fmt.Errorf("shutdown timeout is not configured")
	}
	return nil
}
