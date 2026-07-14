package config

import (
	"fmt"
	"time"

	sharedconfig "github.com/flighttracker/pkg/config"
)

// Config holds all flight-service settings, sourced from environment
// variables prefixed with FLIGHT_ (see .env.example).
type Config struct {
	Environment string `mapstructure:"environment"`

	HTTP struct {
		Port            int           `mapstructure:"port"`
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	} `mapstructure:"http"`

	DB struct {
		Host     string `mapstructure:"host"`
		Port     int    `mapstructure:"port"`
		User     string `mapstructure:"user"`
		Password string `mapstructure:"password"`
		Name     string `mapstructure:"name"`
		SSLMode  string `mapstructure:"sslmode"`
	} `mapstructure:"db"`

	Redis struct {
		Addr     string `mapstructure:"addr"`
		Password string `mapstructure:"password"`
		DB       int    `mapstructure:"db"`
	} `mapstructure:"redis"`

	Cache struct {
		TTL time.Duration `mapstructure:"ttl"`
	} `mapstructure:"cache"`

	Search struct {
		// MaxRouteResults bounds both the DB query and the cached payload
		// for route search (docs/api-contracts/flight-service.yaml caps
		// `limit` at 50); a request's `limit` truncates this cached set at
		// read time rather than being part of the cache key.
		MaxRouteResults int `mapstructure:"max_route_results"`
	} `mapstructure:"search"`
}

// DSN builds a Postgres connection string for gorm.io/driver/postgres.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=flight",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode,
	)
}

// Load reads Config from environment variables (prefix FLIGHT_), falling
// back to the defaults below. optionalFile may point at a local .env-style
// file for development; pass "" in deployed environments.
func Load(optionalFile string) (*Config, error) {
	defaults := map[string]any{
		"environment":           "local",
		"http.port":             8080,
		"http.shutdown_timeout": "15s",

		"db.host":     "localhost",
		"db.port":     5432,
		"db.user":     "postgres",
		"db.password": "",
		"db.name":     "flight_tracker",
		"db.sslmode":  "disable",

		"redis.addr":     "localhost:6379",
		"redis.password": "",
		"redis.db":       0,

		"cache.ttl": "60s",

		"search.max_route_results": 50,
	}
	return sharedconfig.Load[Config]("FLIGHT", optionalFile, defaults)
}
