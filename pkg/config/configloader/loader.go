package configloader

import (
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/joho/godotenv"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

type Validator interface {
	Validate() error
}

func Load[T Validator](serviceName string) (T, error) {
	var cfg T
	// Create a new Koanf instance
	k := koanf.New(".")

	// Convention: config file is named as <service_name>_service.yaml
	// and located in the "configs" directory.
	// envPrefix is set to <service_name>_SVC_ to match environment variables.
	configFile := "config.yaml"
	envPrefix := fmt.Sprintf("%s_", strings.ToUpper(serviceName))

	// 1. Load configuration from yaml file
	if err := k.Load(file.Provider(configFile), yaml.Parser()); err != nil {
		if !os.IsNotExist(err) {
			log.Printf("WARN: error loading YAML config file '%s': %v", configFile, err)
		}
	}

	// 2. Load environment variables from .env file
	envTransformer := func(key string) string {
		key = strings.ToLower(key)
		key = strings.TrimPrefix(key, strings.ToLower(envPrefix))
		return strings.ReplaceAll(key, "_", ".")
	}
	if envFileMap, err := godotenv.Read(".env"); err == nil {
		envMap := make(map[string]any)
		for key, value := range envFileMap {
			envMap[envTransformer(key)] = value
		}
		// Load the envMap into Koanf
		if err := k.Load(confmap.Provider(envMap, "."), nil); err != nil {
			log.Printf("WARN: error loading .env config: %v", err)
		}
	} else if !os.IsNotExist(err) {
		log.Printf("WARN: error reading .env file: %v", err)
	}

	// 3. Load environment variables from the system, the highest priority
	if err := k.Load(env.Provider(envPrefix, ".", envTransformer), nil); err != nil {
		log.Printf("WARN: error loading system env vars: %v", err)
	}

	// 4. Unmarshal the configuration into the Config struct
	if err := k.Unmarshal("", &cfg); err != nil {
		return cfg, fmt.Errorf("error unmarshalling config: %w", err)
	}

	// 5. Validate the configuration
	if err := cfg.Validate(); err != nil {
		return cfg, fmt.Errorf("config validation failed: %w", err)
	}

	return cfg, nil
}
