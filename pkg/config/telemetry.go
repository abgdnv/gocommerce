package config

import (
	"fmt"
	"strings"
	"time"
)

type TelemetryConfig struct {
	Traces TracesConfig `koanf:"traces"`
}

type TracesConfig struct {
	OtlpHttp OtlpHttpConfig `koanf:"otlphttp"`
}

type OtlpHttpConfig struct {
	Endpoint string        `koanf:"endpoint"`
	Insecure bool          `koanf:"insecure"`
	Timeout  time.Duration `koanf:"timeout"`
}

// String returns a string representation of the TelemetryConfig.
func (c *TelemetryConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Telemetry ---\n")
	b.WriteString(fmt.Sprintf("  traces.otlphttp.endpoint: %s\n", c.Traces.OtlpHttp.Endpoint))
	b.WriteString(fmt.Sprintf("  traces.otlphttp.insecure: %v\n", c.Traces.OtlpHttp.Insecure))
	b.WriteString(fmt.Sprintf("  traces.otlphttp.timeout: %v\n", c.Traces.OtlpHttp.Timeout))
	return b.String()
}

func (c *TelemetryConfig) Validate() error {
	if c.Traces.OtlpHttp.Endpoint == "" {
		return fmt.Errorf("OTel endpoint is not configured")
	}
	if c.Traces.OtlpHttp.Timeout <= 0 {
		return fmt.Errorf("telemetry timeout must be greater than 0")
	}

	return nil
}
