// Command sync-service runs one poll-diff-ingest-publish cycle
// (docs/architecture.md section 2.4) and exits — it is invoked by a
// Kubernetes CronJob (deployments/helm/flight-tracker/charts/sync-service),
// not an in-process ticker, so there is deliberately no long-running
// scheduler loop here.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/flighttracker/pkg/eventbus"
	"github.com/flighttracker/pkg/httpserver"
	"github.com/flighttracker/pkg/logger"
	sharedmiddleware "github.com/flighttracker/pkg/middleware"

	"github.com/flighttracker/services/sync-service/internal/config"
	"github.com/flighttracker/services/sync-service/internal/repository"
	"github.com/flighttracker/services/sync-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "sync-service: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.Environment, "sync-service")
	if err != nil {
		return fmt.Errorf("init logger: %w", err)
	}
	defer log.Sync() //nolint:errcheck

	db, err := gorm.Open(postgres.Open(cfg.DSN()), &gorm.Config{})
	if err != nil {
		return fmt.Errorf("connect db: %w", err)
	}
	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("get sql.DB: %w", err)
	}
	defer sqlDB.Close()

	publisher, err := eventbus.NewPublisher(eventbus.PublisherConfig{
		URL:      cfg.RabbitMQ.URL,
		Exchange: cfg.RabbitMQ.Exchange,
	}, log)
	if err != nil {
		return fmt.Errorf("connect rabbitmq: %w", err)
	}
	defer publisher.Close()

	provider, providerName, err := newProvider(cfg)
	if err != nil {
		return fmt.Errorf("init provider: %w", err)
	}

	flightClient := repository.NewFlightServiceClient(cfg.Clients.FlightServiceURL, cfg.Clients.Timeout)
	snapshotRepo := repository.NewSnapshotRepository(db)
	pollRunRepo := repository.NewPollRunRepository(db)

	syncService := service.NewSyncService(provider, snapshotRepo, pollRunRepo, flightClient, publisher, providerName, log)

	router := newRouter(cfg, sqlDB, log)
	server := httpserver.New(cfg.HTTP.Port, router, log, cfg.HTTP.ShutdownTimeout)

	// The HTTP server (/health, /ready, /metrics) only needs to live for
	// the duration of this run — sync-service is a batch job, not a
	// Deployment (see deployments/helm/.../sync-service/values.yaml).
	serverCtx, cancelServer := context.WithCancel(context.Background())
	serverErrCh := make(chan error, 1)
	go func() { serverErrCh <- server.Run(serverCtx) }()

	log.Info("sync_run_starting", zap.String("provider", providerName))
	runCtx, cancelRun := context.WithTimeout(context.Background(), 3*time.Minute)
	summary, runErr := syncService.RunOnce(runCtx)
	cancelRun()

	cancelServer()
	if err := <-serverErrCh; err != nil {
		log.Error("http_server_error", zap.Error(err))
	}

	if runErr != nil {
		return fmt.Errorf("run once: %w", runErr)
	}
	if len(summary.Errors) > 0 {
		return fmt.Errorf("%d flight(s) failed during sync", len(summary.Errors))
	}
	return nil
}

// newProvider builds the configured FlightDataProvider. Only "fake" exists
// today; a real provider is added as an additional case here when one is
// chosen (see internal/config's Provider.Name doc comment).
func newProvider(cfg *config.Config) (repository.FlightDataProvider, string, error) {
	switch cfg.Provider.Name {
	case "fake", "":
		var nowFn func() time.Time
		if cfg.Provider.FakeNowOverride != "" {
			fixed, err := time.Parse(time.RFC3339, cfg.Provider.FakeNowOverride)
			if err != nil {
				return nil, "", fmt.Errorf("parse provider.fake_now_override: %w", err)
			}
			nowFn = func() time.Time { return fixed }
		}
		return repository.NewFakeProvider(nowFn), "fake-provider", nil
	default:
		return nil, "", fmt.Errorf("unknown provider %q", cfg.Provider.Name)
	}
}

func newRouter(cfg *config.Config, sqlDB *sql.DB, log *zap.Logger) *gin.Engine {
	if cfg.Environment != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(sharedmiddleware.CorrelationID(), sharedmiddleware.RequestLogger(log), sharedmiddleware.Recovery(log))
	httpserver.RegisterProbes(router, func(ctx context.Context) error {
		return sqlDB.PingContext(ctx)
	})
	return router
}
