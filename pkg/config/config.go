// Package config loads service configuration from environment variables
// (with an optional .env-style file for local development) via Viper, into
// a typed struct.
package config

import (
	"fmt"
	"os"
	"strings"

	"github.com/spf13/viper"
)

// Load reads configuration into a new T, using envPrefix to namespace
// environment variables (e.g. prefix "BOT" reads BOT_HTTP_PORT into a
// field mapped by the mapstructure tag `http.port`). defaults are applied
// before ENV overrides. optionalFile, if non-empty and present on disk, is
// loaded into the process environment first (used for local development
// only — it must never be relied on in a deployed environment, where ENV
// vars populated from ConfigMap/Secret are the only source of truth).
// Real environment variables always take precedence over optionalFile.
func Load[T any](envPrefix, optionalFile string, defaults map[string]any) (*T, error) {
	if optionalFile != "" {
		if err := loadDotEnvIntoEnvironment(optionalFile); err != nil {
			return nil, fmt.Errorf("config: reading %s: %w", optionalFile, err)
		}
	}

	v := viper.New()
	for key, val := range defaults {
		v.SetDefault(key, val)
	}

	v.SetEnvPrefix(envPrefix)
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()

	var cfg T
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("config: unmarshal: %w", err)
	}
	return &cfg, nil
}

// loadDotEnvIntoEnvironment parses a KEY=VALUE-per-line file and calls
// os.Setenv for each key not already present in the environment. It
// deliberately sets real process environment variables (rather than
// handing the file to Viper as a "config file") so BOT_HTTP_PORT works
// identically whether it comes from a real env var or from .env — Viper's
// AutomaticEnv/SetEnvPrefix machinery only ever inspects the OS
// environment, not config-file contents.
func loadDotEnvIntoEnvironment(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}

	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		key, value, ok := strings.Cut(line, "=")
		if !ok {
			continue
		}
		key = strings.TrimSpace(key)
		value = strings.Trim(strings.TrimSpace(value), `"'`)
		if _, exists := os.LookupEnv(key); !exists {
			if err := os.Setenv(key, value); err != nil {
				return err
			}
		}
	}
	return nil
}
