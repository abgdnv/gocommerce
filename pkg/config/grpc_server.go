package config

import "fmt"

type GrpcServerConfig struct {
	Port              string `koanf:"port"`
	ReflectionEnabled bool   `koanf:"reflection"`
}

func (c *GrpcServerConfig) Validate() error {
	if c.Port == "" {
		return fmt.Errorf("gRPC port is not configured")
	}
	return nil
}
