package config

import (
	rootconfig "github.com/nexaCampus/backend-school-go/config"
)

// Config is a type alias to the canonical root configuration struct.
type Config = rootconfig.Config

// Load populates configuration from environment variables.
func Load() *Config {
	return rootconfig.Load()
}
