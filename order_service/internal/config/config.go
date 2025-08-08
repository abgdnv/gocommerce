package config

import (
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	HTTPServer config.HTTPConfig     `koanf:"server"`
	Database   config.DatabaseConfig `koanf:"db"`
	Log        config.LogConfig      `koanf:"log"`
	PProf      config.PProfConfig    `koanf:"pprof"`
	Nats       config.NATSConfig     `koanf:"nats"`
	Shutdown   config.ShutdownConfig `koanf:"shutdown"`
	Services   struct {
		Product struct {
			Grpc config.GrpcClientConfig `koanf:"grpc"`
		} `koanf:"product"`
	} `koanf:"services"`
}

func (c *Config) String() string {

	var b strings.Builder
	b.WriteString(c.HTTPServer.String())
	b.WriteString(c.Database.String())
	b.WriteString(c.Services.Product.Grpc.String())
	b.WriteString(c.Nats.String())
	b.WriteString(c.Log.String())
	b.WriteString(c.PProf.String())
	b.WriteString(c.Shutdown.String())

	return b.String()
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if err := c.HTTPServer.Validate(); err != nil {
		return err
	}
	if err := c.Database.Validate(); err != nil {
		return err
	}
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.PProf.Validate(); err != nil {
		return err
	}
	if err := c.Nats.Validate(); err != nil {
		return err
	}
	if err := c.Shutdown.Validate(); err != nil {
		return err
	}
	if err := c.Services.Product.Grpc.Validate(); err != nil {
		return err
	}

	return nil
}
