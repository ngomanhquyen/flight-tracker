// Command bot-service runs the Telegram webhook receiver and command
// dispatcher described in docs/architecture.md.
package main

import (
	"context"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"

	"github.com/flighttracker/pkg/httpserver"
	"github.com/flighttracker/pkg/logger"
	sharedmiddleware "github.com/flighttracker/pkg/middleware"
	"github.com/flighttracker/pkg/telegram"

	"github.com/flighttracker/services/bot-service/internal/config"
	"github.com/flighttracker/services/bot-service/internal/handler"
	"github.com/flighttracker/services/bot-service/internal/repository"
	"github.com/flighttracker/services/bot-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "bot-service: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.Environment, "bot-service")
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	telegramClient := telegram.NewClient(cfg.Telegram.BotToken, cfg.Telegram.RequestTimeout)
	flightClient := repository.NewFlightClient(cfg.Clients.FlightServiceURL, cfg.Clients.Timeout)
	subscriptionClient := repository.NewSubscriptionClient(cfg.Clients.SubscriptionServiceURL, cfg.Clients.Timeout)

	botService := service.NewBotService(flightClient, subscriptionClient, telegramClient, log)
	webhookHandler := handler.NewWebhookHandler(
		botService,
		cfg.Telegram.WebhookSecret,
		cfg.Telegram.WebhookPathSecret,
		cfg.Telegram.RequestTimeout,
		log,
	)

	router := newRouter(cfg, webhookHandler, log)
	server := httpserver.New(cfg.HTTP.Port, router, log, cfg.HTTP.ShutdownTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("bot_service_starting", zap.Int("port", cfg.HTTP.Port), zap.String("environment", cfg.Environment))
	return server.Run(ctx)
}

func newRouter(cfg *config.Config, webhookHandler *handler.WebhookHandler, log *zap.Logger) *gin.Engine {
	if cfg.Environment != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(sharedmiddleware.CorrelationID(), sharedmiddleware.RequestLogger(log), sharedmiddleware.Recovery(log))

	httpserver.RegisterProbes(router, readinessCheck(cfg))
	router.POST("/webhook/telegram/:secret", webhookHandler.Handle)

	return router
}

// readinessCheck confirms bot-service's own downstream dependencies are
// reachable. bot-service holds no database/cache/broker connection
// itself, so this is the entirety of its readiness surface.
//
// TEMPORARY: subscription-service isn't implemented yet, so it's
// deliberately left out of this check — otherwise bot-service would never
// report ready. Add cfg.Clients.SubscriptionServiceURL back to the slice
// below once subscription-service exists and is deployed.
func readinessCheck(cfg *config.Config) httpserver.ReadyCheckFunc {
	client := &http.Client{Timeout: cfg.Clients.Timeout}
	return func(ctx context.Context) error {
		for _, base := range []string{cfg.Clients.FlightServiceURL} {
			req, err := http.NewRequestWithContext(ctx, http.MethodGet, base+"/health", nil)
			if err != nil {
				return err
			}
			resp, err := client.Do(req)
			if err != nil {
				return fmt.Errorf("%s: %w", base, err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusOK {
				return fmt.Errorf("%s: unhealthy (status %d)", base, resp.StatusCode)
			}
		}
		return nil
	}
}
