package config

import (
	"fmt"
	"strings"
	"time"
)

type GrpcClientConfig struct {
	Addr    string        `koanf:"addr"`
	Timeout time.Duration `koanf:"timeout"`
}

// String returns a string representation of the gRPC client configuration.
func (c *GrpcClientConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- gRPC Client ---\n")
	b.WriteString(fmt.Sprintf("  addr: %s\n", c.Addr))
	b.WriteString(fmt.Sprintf("  timeout: %s\n", c.Timeout))
	return b.String()
}

func (c *GrpcClientConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("gRPC address is not configured")
	}
	if c.Timeout <= 0 {
		return fmt.Errorf("gRPC timeout is not configured")
	}
	return nil
}
