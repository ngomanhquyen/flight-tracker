package config

import (
	"fmt"
	"time"

	sharedconfig "github.com/flighttracker/pkg/config"
)

// Config holds all sync-service settings, sourced from environment
// variables prefixed with SYNC_ (see .env.example). Dots in the keys below
// become underscores in the ENV var name, e.g. db.host -> SYNC_DB_HOST.
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

	RabbitMQ struct {
		URL      string `mapstructure:"url"`
		Exchange string `mapstructure:"exchange"`
	} `mapstructure:"rabbitmq"`

	Clients struct {
		FlightServiceURL string        `mapstructure:"flight_service_url"`
		Timeout          time.Duration `mapstructure:"timeout"`
	} `mapstructure:"clients"`

	Provider struct {
		// Name identifies the FlightDataProvider implementation to wire up
		// in cmd/main.go. Only "fake" exists today (see
		// internal/repository/fake_provider.go); a real provider is added
		// as an additional case, never by changing this field's meaning.
		Name string `mapstructure:"name"`
		// FakeNowOverride (RFC3339), if set, pins the fake provider's
		// notion of "now" for manual testing of specific status
		// transitions without waiting on the wall clock. Only consulted
		// when Provider.Name is "fake"; never set in a real deployment.
		FakeNowOverride string `mapstructure:"fake_now_override"`
	} `mapstructure:"provider"`
}

// DSN builds a Postgres connection string for gorm.io/driver/postgres.
func (c *Config) DSN() string {
	return fmt.Sprintf(
		"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s search_path=sync",
		c.DB.Host, c.DB.Port, c.DB.User, c.DB.Password, c.DB.Name, c.DB.SSLMode,
	)
}

// Load reads Config from environment variables (prefix SYNC_), falling
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

		"rabbitmq.url":      "amqp://guest:guest@localhost:5672/",
		"rabbitmq.exchange": "flight.events",

		"clients.flight_service_url": "http://flight-service:8080",
		"clients.timeout":            "5s",

		"provider.name":              "fake",
		"provider.fake_now_override": "",
	}
	return sharedconfig.Load[Config]("SYNC", optionalFile, defaults)
}
