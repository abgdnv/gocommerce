package config

import (
	"fmt"
	"strings"
	"time"
)

type IdP struct {
	JwksURL     string        `koanf:"jwksurl"`
	Issuer      string        `koanf:"issuer"`
	ClientID    string        `koanf:"clientid"`
	MinInterval time.Duration `koanf:"mininterval"`
}

// String returns a string representation of the IdP configuration.
func (c *IdP) String() string {
	var b strings.Builder
	b.WriteString("\n--- Identity Provider ---\n")
	b.WriteString(fmt.Sprintf("  jwksurl: %s\n", c.JwksURL))
	b.WriteString(fmt.Sprintf("  issuer: %s\n", c.Issuer))
	b.WriteString(fmt.Sprintf("  clientid: %s\n", c.ClientID))
	b.WriteString(fmt.Sprintf("  mininterval: %v\n", c.MinInterval))
	return b.String()
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
