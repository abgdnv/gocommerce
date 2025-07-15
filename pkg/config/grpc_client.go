package config

import "fmt"

type GrpcClientConfig struct {
	Addr string `koanf:"addr"`
}

func (c *GrpcClientConfig) Validate() error {
	if c.Addr == "" {
		return fmt.Errorf("product service gRPC address is not configured")
	}
	return nil
}
