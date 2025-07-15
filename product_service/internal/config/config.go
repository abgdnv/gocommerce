package config

import (
	"fmt"
	"strings"

	"github.com/abgdnv/gocommerce/pkg/config"
	"github.com/abgdnv/gocommerce/pkg/config/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	HTTPServer config.HTTPConfig       `koanf:"server"`
	Database   config.DatabaseConfig   `koanf:"database"`
	Log        config.LogConfig        `koanf:"log"`
	PProf      config.PProfConfig      `koanf:"pprof"`
	GRPC       config.GrpcServerConfig `koanf:"grpc"`
	Shutdown   config.ShutdownConfig   `koanf:"shutdown"`
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

	b.WriteString("\n--- Database Configuration ---\n")
	b.WriteString(fmt.Sprintf("  database.url: %s\n", maskURL(c.Database.URL)))
	b.WriteString(fmt.Sprintf("  database.connect.timeout: %s\n", c.Database.Timeout))

	b.WriteString("\n--- gRPC Configuration ---\n")
	b.WriteString(fmt.Sprintf("  grpc.port: %s\n", c.GRPC.Port))
	b.WriteString(fmt.Sprintf("  grpc.reflection_enabled: %t\n", c.GRPC.ReflectionEnabled))

	b.WriteString("\n--- Observability & Logging ---\n")
	b.WriteString(fmt.Sprintf("  log.level: %s\n", c.Log.Level))
	b.WriteString(fmt.Sprintf("  pprof.enabled: %t\n", c.PProf.Enabled))
	b.WriteString(fmt.Sprintf("  pprof.address: %s\n", c.PProf.Addr))

	b.WriteString("\n--- Application Behavior ---\n")
	b.WriteString(fmt.Sprintf("  shutdown.timeout: %s\n", c.Shutdown.Timeout))

	return b.String()
}

func maskURL(url string) string {
	if url == "" {
		return "<not configured>"
	}
	// Mask the URL by replacing the username and password with "****"
	parts := strings.Split(url, "@")
	if len(parts) == 2 {
		return "****@" + parts[1]
	}
	return "****"
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
	if err := c.Shutdown.Validate(); err != nil {
		return nil
	}
	if err := c.GRPC.Validate(); err != nil {
		return err
	}
	return nil
}
