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
		URL string `koanf:"url"`
	} `koanf:"database"`

	Log struct {
		Level string `koanf:"level"`
	} `koanf:"log"`

	PProf struct {
		Enabled bool   `koanf:"enabled"`
		Addr    string `koanf:"addr"`
	} `koanf:"pprof"`

	Services struct {
		ProductGrpcAddr string `koanf:"productGrpcAddr"`
	} `koanf:"services"`
}

func (c *Config) String() string {
	return fmt.Sprintf("\n server.port = %d\n server.maxHeaderBytes = %d\n server.timeout.read = %v\n server.timeout.write = %v\n"+
		" server.timeout.idle = %v\n server.timeout.readHeader = %v\n database_url = %s\n log_level = %s\n pprof_enabled = %t\n"+
		" pprof_address = %s\n"+
		" services.productGrpcAddr = %s\n",
		c.HTTPServer.Port,
		c.HTTPServer.MaxHeaderBytes,
		c.HTTPServer.Timeout.Read,
		c.HTTPServer.Timeout.Write,
		c.HTTPServer.Timeout.Idle,
		c.HTTPServer.Timeout.ReadHeader,
		maskURL(c.Database.URL),
		c.Log.Level,
		c.PProf.Enabled,
		c.PProf.Addr,
		c.Services.ProductGrpcAddr,
	)
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
	if c.PProf.Enabled && c.PProf.Addr == "" {
		return fmt.Errorf("pprof is enabled but address is not configured")
	}
	return nil
}

// isValidPostgresURL checks if the provided URL is a valid PostgreSQL URL
func isValidPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") ||
		strings.HasPrefix(url, "postgresql://")
}
