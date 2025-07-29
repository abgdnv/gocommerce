package config

import (
	"fmt"
	"time"
)

type IdP struct {
	JwksURL     string        `koanf:"jwksurl"`
	Issuer      string        `koanf:"issuer"`
	ClientID    string        `koanf:"clientid"`
	MinInterval time.Duration `koanf:"mininterval"`
}

func (c *IdP) Validate() error {
	if c.JwksURL == "" {
		return fmt.Errorf("IdP JWKS URL cannot be empty")
	}
	if c.Issuer == "" {
		return fmt.Errorf("IdP issuer cannot be empty")
	}
	if c.ClientID == "" {
		return fmt.Errorf("IdP client ID cannot be empty")
	}
	if c.MinInterval <= 0 {
		return fmt.Errorf("IdP minimum interval must be greater than zero")
	}
	return nil
}
