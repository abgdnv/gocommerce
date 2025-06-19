package config

import (
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

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
}

func (c Config) String() string {
	return fmt.Sprintf("server.port=%d, server.maxHeaderBytes=%d , server.timeout.read=%v, server.timeout.write=%v, server.timeout.idle=%v, server.timeout.readHeader=%v, database_url=%v, log_level= %s.",
		c.HTTPServer.Port,
		c.HTTPServer.MaxHeaderBytes,
		c.HTTPServer.Timeout.Read,
		c.HTTPServer.Timeout.Write,
		c.HTTPServer.Timeout.Idle,
		c.HTTPServer.Timeout.ReadHeader,
		maskURL(c.Database.URL),
		c.Log.Level)
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

const (
	envPrefix      = "product_svc_"
	defaultEnvFile = ".env"
	configFile     = "config.yaml"
)

// Load reads the configuration from a file and environment variables
func Load() (*Config, error) {
	// Create a new Koanf instance
	var k = koanf.New(".")

	// 1. Load configuration from yaml file
	if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("WARN: error loading YAML config: %v", err)
		}
	}

	// 2. Load environment variables from .env file
	if envFileMap, err := godotenv.Read(defaultEnvFile); err == nil {
		envMap := make(map[string]interface{})
		for key, value := range envFileMap {
			envMap[keyTransformer(key)] = value
		}
		// Load the envMap into Koanf
		if err := k.Load(confmap.Provider(envMap, "."), nil); err != nil {
			log.Printf("WARN: error loading .env config: %v", err)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("WARN: error reading .env file: %v", err)
	}

	// 3. Load environment variables from the system, the highest priority
	if err := k.Load(env.Provider("", ".", keyTransformer), nil); err != nil {
		log.Printf("WARN: error loading env vars: %v", err)
	}

	var cfg Config
	// 4. Unmarshal the configuration into the Config struct
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// 5. Validate the configuration
	if err := validateConfig(cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// validateConfig checks if the configuration values are valid
func validateConfig(cfg Config) error {
	if cfg.HTTPServer.Port <= 0 || cfg.HTTPServer.Port > 65535 {
		return fmt.Errorf("invalid HTTP server port: %d", cfg.HTTPServer.Port)
	}
	if cfg.HTTPServer.Timeout.Read <= 0 {
		return fmt.Errorf("invalid HTTP server read timeout: %v", cfg.HTTPServer.Timeout.Read)
	}
	if cfg.HTTPServer.Timeout.Write <= 0 {
		return fmt.Errorf("invalid HTTP server write timeout: %v", cfg.HTTPServer.Timeout.Write)
	}
	if cfg.HTTPServer.Timeout.Idle <= 0 {
		return fmt.Errorf("invalid HTTP server idle timeout: %v", cfg.HTTPServer.Timeout.Idle)
	}
	if cfg.Database.URL == "" {
		return fmt.Errorf("database URL is not configured")
	}
	if !isValidPostgresURL(cfg.Database.URL) {
		return fmt.Errorf("database URL must start with 'postgres://': %s", cfg.Database.URL)
	}
	return nil
}

// isValidPostgresURL checks if the provided URL is a valid PostgreSQL URL
func isValidPostgresURL(url string) bool {
	return strings.HasPrefix(url, "postgres://") ||
		strings.HasPrefix(url, "postgresql://")
}

// keyTransformer transforms environment variable keys to match the expected format
func keyTransformer(key string) string {
	key = strings.ToLower(key)
	key = strings.TrimPrefix(key, envPrefix)
	return strings.ReplaceAll(key, "_", ".")
}
