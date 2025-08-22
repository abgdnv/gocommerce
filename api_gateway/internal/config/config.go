package config

import (
	"fmt"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	HTTPServer config.HTTPConfig      `koanf:"server"`
	Log        config.LogConfig       `koanf:"log"`
	PProf      config.PProfConfig     `koanf:"pprof"`
	Telemetry  config.TelemetryConfig `koanf:"telemetry"`
	Shutdown   config.ShutdownConfig  `koanf:"shutdown"`
	Services   Services               `koanf:"services"`
	IdP        config.IdP             `koanf:"idp"`
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
	User struct {
		From string                  `koanf:"from"`
		Grpc config.GrpcClientConfig `koanf:"grpc"`
	} `koanf:"user"`
}

func (c *Config) String() string {
	var b strings.Builder
	b.WriteString(c.HTTPServer.String())

	b.WriteString("\n--- Services Configuration ---\n")
	b.WriteString(fmt.Sprintf("  product.url: %s\n", c.Services.Product.Url))
	b.WriteString(fmt.Sprintf("  product.from: %s\n", c.Services.Product.From))
	b.WriteString(fmt.Sprintf("  product.to: %s\n", c.Services.Product.To))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  order.url: %s\n", c.Services.Order.Url))
	b.WriteString(fmt.Sprintf("  order.from: %s\n", c.Services.Order.From))
	b.WriteString(fmt.Sprintf("  order.to: %s\n", c.Services.Order.To))
	b.WriteString("\n")
	b.WriteString(fmt.Sprintf("  user.grpc.addr: %s\n", c.Services.User.Grpc.Addr))
	b.WriteString(fmt.Sprintf("  user.grpc.timeout: %s\n", c.Services.User.Grpc.Timeout))

	b.WriteString(c.IdP.String())
	b.WriteString(c.Log.String())
	b.WriteString(c.PProf.String())
	b.WriteString(c.Telemetry.String())
	b.WriteString(c.Shutdown.String())
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
	if err := c.Telemetry.Validate(); err != nil {
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
	if c.Services.User.From == "" {
		return fmt.Errorf("user service 'from' field cannot be empty")
	}
	if err := c.Services.User.Grpc.Validate(); err != nil {
		return err
	}
	if err := c.IdP.Validate(); err != nil {
		return err
	}
	return nil
}
