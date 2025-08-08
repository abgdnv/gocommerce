package config

import (
	"fmt"
	"strings"
)

type LogConfig struct {
	Level string `koanf:"level"`
}

// String returns a string representation of the log configuration.
func (c *LogConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Log ---\n")
	b.WriteString(fmt.Sprintf("  level: %s\n", c.Level))
	return b.String()
}

func (c *LogConfig) Validate() error {
	return nil
}
