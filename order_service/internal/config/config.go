package config

import (
	"fmt"
	"strings"
	"time"

	"github.com/abgdnv/gocommerce/pkg/configloader"
)

var _ configloader.Validator = (*Config)(nil)

type Config struct {
	HTTPServer struct {
		Port           int `koanf:"port"`
		MaxHeaderBytes int `koanf:"maxHeaderBytes"`
		Timeout        struct {
			Read       time.Duration `koanf:"read"`
			Write      time.Duration `koanf:"write"`
			Idle       time.Duration `koanf:"idle"`
			ReadHeader time.Duration `koanf:"readHeader"`
		} `koanf:"timeout"`
	} `koanf:"server"`

	Database struct {
		URL     string        `koanf:"url"`
		Timeout time.Duration `koanf:"timeout"`
	} `koanf:"database"`

	Log struct {
		Level string `koanf:"level"`
	} `koanf:"log"`

	PProf struct {
		Enabled bool   `koanf:"enabled"`
		Addr    string `koanf:"addr"`
	} `koanf:"pprof"`

	Services struct {
		Product struct {
			Grpc struct {
				Addr string `koanf:"addr"`
			} `koanf:"grpc"`
		} `koanf:"product"`
	} `koanf:"services"`

	Nats struct {
		Url     string        `koanf:"url"`
		Timeout time.Duration `koanf:"timeout"`
	} `koanf:"nats"`

	Shutdown struct {
		Timeout time.Duration `koanf:"timeout"`
	} `koanf:"shutdown"`
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
	b.WriteString(fmt.Sprintf("  database.timeout: %s\n", c.Database.Timeout))

	b.WriteString("\n--- External Services ---\n")
	b.WriteString(fmt.Sprintf("  services.product.grpc.addr: %s\n", c.Services.Product.Grpc.Addr))
	b.WriteString(fmt.Sprintf("  nats.url: %s\n", c.Nats.Url))
	b.WriteString(fmt.Sprintf("  nats.timeout: %s\n", c.Nats.Timeout))

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
	if c.HTTPServer.Port <= 0 || c.HTTPServer.Port > 65535 {
		return fmt.Errorf("invalid HTTP server port: %d", c.HTTPServer.Port)
	}
	if c.HTTPServer.Timeout.Read <= 0 {
		return fmt.Errorf("invalid HTTP server read timeout: %v", c.HTTPServer.Timeout.Read)
	}
	if c.HTTPServer.Timeout.Write <= 0 {
		return fmt.Errorf("invalid HTTP server write timeout: %v", c.HTTPServer.Timeout.Write)
	}
	if c.HTTPServer.Timeout.Idle <= 0 {
		return fmt.Errorf("invalid HTTP server idle timeout: %v", c.HTTPServer.Timeout.Idle)
	}
	if c.HTTPServer.Timeout.ReadHeader <= 0 {
		return fmt.Errorf("invalid HTTP server read header timeout: %v", c.HTTPServer.Timeout.ReadHeader)
	}
	if c.Database.URL == "" {
		return fmt.Errorf("database URL is not configured")
	}
	if !isValidPostgresURL(c.Database.URL) {
		return fmt.Errorf("database URL must start with 'postgres://': %s", c.Database.URL)
	}
	if c.Database.Timeout <= 0 {
		return fmt.Errorf("database connect timeout is not configured")
	}
	if c.PProf.Enabled && c.PProf.Addr == "" {
		return fmt.Errorf("pprof is enabled but address is not configured")
	}
	if c.Services.Product.Grpc.Addr == "" {
		return fmt.Errorf("product service gRPC address is not configured")
	}
	if c.Nats.Url == "" {
		return fmt.Errorf("NATS URL is not configured")
	}
	if c.Nats.Timeout <= 0 {
		return fmt.Errorf("nats dial timeout is not configured")
	}
	if c.Shutdown.Timeout <= 0 {
		return fmt.Errorf("shutdown timeout is not configured")
	}
	return nil
}

// isValidPostgresURL checks if the provided URL is a valid PostgreSQL URL
func isValidPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") ||
		strings.HasPrefix(url, "postgresql://")
}
