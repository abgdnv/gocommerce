package config

import (
	"fmt"
	"strings"
)

type GrpcServerConfig struct {
	Port              string `koanf:"port"`
	ReflectionEnabled bool   `koanf:"reflection"`
}

// String returns a string representation of the gRPC server configuration.
func (c *GrpcServerConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- gRPC Server ---\n")
	b.WriteString(fmt.Sprintf("  port: %s\n", c.Port))
	b.WriteString(fmt.Sprintf("  reflection_enabled: %t\n", c.ReflectionEnabled))
	return b.String()
}

func (c *GrpcServerConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("gRPC port is not configured")
	}
	return nil
}
