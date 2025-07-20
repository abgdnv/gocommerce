package config

import (
	"fmt"
	"time"
)

type GrpcClientConfig struct {
	Addr    string        `koanf:"addr"`
	Timeout time.Duration `koanf:"timeout"`
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
