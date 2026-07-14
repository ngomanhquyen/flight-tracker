package config

import (
	"time"

	sharedconfig "github.com/flighttracker/pkg/config"
)

// Config holds all bot-service settings, sourced from environment
// variables prefixed with BOT_ (see .env.example). Dots in the keys below
// become underscores in the ENV var name, e.g. http.port -> BOT_HTTP_PORT.
type Config struct {
	Environment string `mapstructure:"environment"`

	HTTP struct {
		Port            int           `mapstructure:"port"`
		ShutdownTimeout time.Duration `mapstructure:"shutdown_timeout"`
	} `mapstructure:"http"`

	Telegram struct {
		BotToken          string        `mapstructure:"bot_token"`
		WebhookSecret     string        `mapstructure:"webhook_secret"`
		WebhookPathSecret string        `mapstructure:"webhook_path_secret"`
		RequestTimeout    time.Duration `mapstructure:"request_timeout"`
	} `mapstructure:"telegram"`

	Clients struct {
		FlightServiceURL       string        `mapstructure:"flight_service_url"`
		SubscriptionServiceURL string        `mapstructure:"subscription_service_url"`
		Timeout                time.Duration `mapstructure:"timeout"`
	} `mapstructure:"clients"`
}

// Load reads Config from environment variables (prefix BOT_), falling
// back to the defaults below. optionalFile may point at a local .env-style
// file for development; pass "" in deployed environments.
func Load(optionalFile string) (*Config, error) {
	defaults := map[string]any{
		"environment":           "local",
		"http.port":             8080,
		"http.shutdown_timeout": "15s",
		// Viper's Unmarshal only consults AutomaticEnv for keys it already
		// knows about (i.e. keys with a registered default) — an env var
		// for a key with no default here is silently ignored. Every field
		// below therefore needs an entry, even when "" is the only sane
		// default (secrets/tokens must come from ENV/Secret, never a
		// committed default).
		"telegram.bot_token":               "",
		"telegram.webhook_secret":          "",
		"telegram.webhook_path_secret":     "",
		"telegram.request_timeout":         "10s",
		"clients.timeout":                  "5s",
		"clients.flight_service_url":       "http://flight-service:8080",
		"clients.subscription_service_url": "http://subscription-service:8080",
	}
	return sharedconfig.Load[Config]("BOT", optionalFile, defaults)
}
