package config

import (
	"fmt"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	HTTPServer config.HTTPConfig     `koanf:"server"`
	Log        config.LogConfig      `koanf:"log"`
	PProf      config.PProfConfig    `koanf:"pprof"`
	Shutdown   config.ShutdownConfig `koanf:"shutdown"`
	Services   Services              `koanf:"services"`
	IdP        config.IdP            `koanf:"idp"`
}

type Services struct {
	Product struct {
		Url  string `koanf:"url"`
		From string `koanf:"from"`
		To   string `koanf:"to"`
	} `koanf:"product"`
	Order struct {
		Url  string `koanf:"url"`
		From string `koanf:"from"`
		To   string `koanf:"to"`
	} `koanf:"order"`
}

func (c *Config) String() string {
	var b strings.Builder

	b.WriteString("\n--- Server Configuration ---\n")
	b.WriteString(fmt.Sprintf("  server.port: %d\n", c.HTTPServer.Port))
	b.WriteString(fmt.Sprintf("  server.maxHeaderBytes: %d\n", c.HTTPServer.MaxHeaderBytes))
	b.WriteString(fmt.Sprintf("  server.timeout.read: %v\n", c.HTTPServer.Timeout.Read))
	b.WriteString(fmt.Sprintf("  server.timeout.write: %v\n", c.HTTPServer.Timeout.Write))
	b.WriteString(fmt.Sprintf("  server.timeout.idle: %v\n", c.HTTPServer.Timeout.Idle))
	b.WriteString(fmt.Sprintf("  server.timeout.readHeader: %v\n", c.HTTPServer.Timeout.ReadHeader))

	b.WriteString("\n--- Services Configuration ---\n")
	b.WriteString(fmt.Sprintf("  product.service.url: %s\n", c.Services.Product.Url))
	b.WriteString(fmt.Sprintf("  product.service.from: %s\n", c.Services.Product.From))
	b.WriteString(fmt.Sprintf("  product.service.to: %s\n", c.Services.Product.To))
	b.WriteString(fmt.Sprintf("  order.service.url: %s\n", c.Services.Order.Url))
	b.WriteString(fmt.Sprintf("  order.service.from: %s\n", c.Services.Order.From))
	b.WriteString(fmt.Sprintf("  order.service.to: %s\n", c.Services.Order.To))

	b.WriteString("\n--- Identity Provider ---\n")
	b.WriteString(fmt.Sprintf("  idp.jwksurl: %s\n", c.IdP.JwksURL))
	b.WriteString(fmt.Sprintf("  idp.issuer: %s\n", c.IdP.Issuer))
	b.WriteString(fmt.Sprintf("  idp.clientid: %s\n", c.IdP.ClientID))
	b.WriteString(fmt.Sprintf("  idp.mininterval: %v\n", c.IdP.MinInterval))

	b.WriteString("\n--- Observability & Logging ---\n")
	b.WriteString(fmt.Sprintf("  log.level: %s\n", c.Log.Level))
	b.WriteString(fmt.Sprintf("  pprof.enabled: %t\n", c.PProf.Enabled))
	b.WriteString(fmt.Sprintf("  pprof.address: %s\n", c.PProf.Addr))

	b.WriteString("\n--- Application Behavior ---\n")
	b.WriteString(fmt.Sprintf("  shutdown.timeout: %s\n", c.Shutdown.Timeout))

	return b.String()
}

// Validate checks if the configuration values are valid
func (c *Config) Validate() error {
	if err := c.HTTPServer.Validate(); err != nil {
		return err
	}
	if err := c.Log.Validate(); err != nil {
		return err
	}
	if err := c.PProf.Validate(); err != nil {
		return err
	}
	if err := c.Shutdown.Validate(); err != nil {
		return err
	}
	if c.Services.Product.Url == "" {
		return fmt.Errorf("product service URL cannot be empty")
	}
	if c.Services.Product.From == "" {
		return fmt.Errorf("product service 'from' field cannot be empty")
	}
	if c.Services.Product.To == "" {
		return fmt.Errorf("product service 'to' field cannot be empty")
	}
	if c.Services.Order.Url == "" {
		return fmt.Errorf("order service URL cannot be empty")
	}
	if c.Services.Order.From == "" {
		return fmt.Errorf("order service 'from' field cannot be empty")
	}
	if c.Services.Order.To == "" {
		return fmt.Errorf("order service 'to' field cannot be empty")
	}
	if err := c.IdP.Validate(); err != nil {
		return err
	}
	return nil
}
