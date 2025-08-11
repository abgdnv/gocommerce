package config

import (
	"fmt"
	"log"
	"strings"
	"time"
)

type ProbesConfig struct {
	ReadinessFileName string        `koanf:"readinessfilename"`
	LivenessFileName  string        `koanf:"livenessfilename"`
	LivenessInterval  time.Duration `koanf:"livenessinterval"`
}

const defaultReadinessFileName = "/tmp/ready"
const defaultLivenessFileName = "/tmp/live"
const defaultLivenessInterval = 20 * time.Second

// String returns a string representation of the ProbesConfig.
func (c *ProbesConfig) String() string {
	var b strings.Builder
	b.WriteString("\n--- Probes ---\n")
	b.WriteString(fmt.Sprintf("  readinessfilename: %s\n", c.ReadinessFileName))
	b.WriteString(fmt.Sprintf("  livenessfilename: %s\n", c.LivenessFileName))
	b.WriteString(fmt.Sprintf("  livenessinterval: %s\n", c.LivenessInterval))
	return b.String()
}

func (c *ProbesConfig) Validate() error {
	if c.ReadinessFileName == "" {
		log.Println("Using default value for readinessfilename")
		c.ReadinessFileName = defaultReadinessFileName
	}
	if c.LivenessFileName == "" {
		log.Println("Using default value for livenessfilename")
		c.LivenessFileName = defaultLivenessFileName
	}
	if c.LivenessInterval <= 0 {
		log.Println("Using default value for livenessinterval")
		c.LivenessInterval = defaultLivenessInterval
	}

	return nil
}
