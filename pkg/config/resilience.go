package config

import (
	"fmt"
	"strings"
	"time"
)

type ResilienceConfig struct {
	Retry          RetryConfig          `koanf:"retry"`
	CircuitBreaker CircuitBreakerConfig `koanf:"circuitbreaker"`
}

type RetryConfig struct {
	MaxAttempts    uint          `koanf:"maxattempts"`
	InitialBackoff time.Duration `koanf:"initialbackoff"`
}

type CircuitBreakerConfig struct {
	ConsecutiveFailures uint32        `koanf:"consecutivefailures"`
	ErrorRatePercent    int           `koanf:"errorratepercent"`
	OpenTimeout         time.Duration `koanf:"opentimeout"`
}

// String returns a string representation of the ResilienceConfig.
func (c *ResilienceConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Retry ---\n")
	b.WriteString(fmt.Sprintf("  maxattempts: %d\n", c.Retry.MaxAttempts))
	b.WriteString(fmt.Sprintf("  initialbackoff: %v\n", c.Retry.InitialBackoff))
	b.WriteString("\n--- Circuit Breaker ---\n")
	b.WriteString(fmt.Sprintf("  consecutivefailures: %d\n", c.CircuitBreaker.ConsecutiveFailures))
	b.WriteString(fmt.Sprintf("  errorratepercent: %d\n", c.CircuitBreaker.ErrorRatePercent))
	b.WriteString(fmt.Sprintf("  opentimeout: %v\n", c.CircuitBreaker.OpenTimeout))
	return b.String()
}

func (c *ResilienceConfig) Validate() error {
	if c.Retry.MaxAttempts <= 0 {
		return fmt.Errorf("retry.max_attempts must be greater than 0")
	}
	if c.Retry.InitialBackoff <= 0 {
		return fmt.Errorf("retry.initial_backoff must be greater than 0")
	}
	if c.CircuitBreaker.ConsecutiveFailures <= 0 {
		return fmt.Errorf("circuit_breaker.consecutive_failures must be greater than 0")
	}
	if c.CircuitBreaker.ErrorRatePercent < 0 || c.CircuitBreaker.ErrorRatePercent > 100 {
		return fmt.Errorf("circuit_breaker.error_rate_percent must be between 0 and 100")
	}
	if c.CircuitBreaker.OpenTimeout <= 0 {
		return fmt.Errorf("circuit_breaker.open_timeout must be greater than 0")
	}
	return nil
}
