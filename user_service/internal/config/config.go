package config

import (
	"fmt"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	Log      config.LogConfig        `koanf:"log"`
	PProf    config.PProfConfig      `koanf:"pprof"`
	GRPC     config.GrpcServerConfig `koanf:"grpc"`
	IdP      IdP                     `koanf:"idp"`
	Shutdown config.ShutdownConfig   `koanf:"shutdown"`
}

type IdP struct {
	URL      string `koanf:"url"`
	Realm    string `koanf:"realm"`
	ClientID string `koanf:"clientid"`
	Secret   string `koanf:"secret"`
}

func (c *IdP) Validate() error {
	if c.URL == "" {
		return fmt.Errorf("IdP URL cannot be empty")
	}
	if c.Realm == "" {
		return fmt.Errorf("IdP realm cannot be empty")
	}
	if c.ClientID == "" {
		return fmt.Errorf("IdP client ID cannot be empty")
	}
	if c.Secret == "" {
		return fmt.Errorf("IdP secret cannot be empty")
	}
	return nil
}

func (c *Config) String() string {
	var b strings.Builder
	b.WriteString("\n--- Identity Provider ---\n")
	b.WriteString(fmt.Sprintf("  idp.clientid: %s\n", c.IdP.ClientID))
	b.WriteString(c.GRPC.String())
	b.WriteString(c.Log.String())
	b.WriteString(c.PProf.String())
	b.WriteString(c.Shutdown.String())
	return b.String()
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.PProf.Validate(); err != nil {
		return err
	}
	if err := c.GRPC.Validate(); err != nil {
		return err
	}
	if err := c.IdP.Validate(); err != nil {
		return err
	}
	if err := c.Shutdown.Validate(); err != nil {
		return err
	}
	return nil
}
