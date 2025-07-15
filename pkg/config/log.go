package config

type LogConfig struct {
	Level string `koanf:"level"`
}

func (c *LogConfig) Validate() error {
	return nil
}
