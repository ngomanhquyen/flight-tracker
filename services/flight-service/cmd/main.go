// Command flight-service runs the public search API and internal ingest
// endpoint described in docs/architecture.md section 2.3. Unlike
// sync-service, this is a long-running server (Kubernetes Deployment),
// mirroring bot-service's cmd/main.go shape.
package main

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"os/signal"
	"syscall"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"

	"github.com/flighttracker/pkg/httpserver"
	"github.com/flighttracker/pkg/logger"
	sharedmiddleware "github.com/flighttracker/pkg/middleware"

	"github.com/flighttracker/services/flight-service/internal/config"
	"github.com/flighttracker/services/flight-service/internal/handler"
	"github.com/flighttracker/services/flight-service/internal/repository"
	"github.com/flighttracker/services/flight-service/internal/service"
)

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, "flight-service: fatal:", err)
		os.Exit(1)
	}
}

func run() error {
	cfg, err := config.Load(".env")
	if err != nil {
		return fmt.Errorf("load config: %w", err)
	}

	log, err := logger.New(cfg.Environment, "flight-service")
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

	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	flightRepo := repository.NewFlightRepository(db)
	cacheRepo := repository.NewRedisCacheRepository(redisClient, cfg.Cache.TTL)

	searchService := service.NewFlightSearchService(flightRepo, cacheRepo, cfg.Search.MaxRouteResults, log)
	ingestService := service.NewIngestService(flightRepo, cacheRepo, log)
	flightHandler := handler.NewFlightHandler(handler.SearchAndIngest{Search: searchService, Ingest: ingestService})

	router := newRouter(cfg, flightHandler, sqlDB, redisClient, log)
	server := httpserver.New(cfg.HTTP.Port, router, log, cfg.HTTP.ShutdownTimeout)

	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	log.Info("flight_service_starting", zap.Int("port", cfg.HTTP.Port), zap.String("environment", cfg.Environment))
	return server.Run(ctx)
}

func newRouter(cfg *config.Config, flightHandler *handler.FlightHandler, sqlDB *sql.DB, redisClient *redis.Client, log *zap.Logger) *gin.Engine {
	if cfg.Environment != "local" {
		gin.SetMode(gin.ReleaseMode)
	}

	router := gin.New()
	router.Use(sharedmiddleware.CorrelationID(), sharedmiddleware.RequestLogger(log), sharedmiddleware.Recovery(log))

	httpserver.RegisterProbes(router, func(ctx context.Context) error {
		if err := sqlDB.PingContext(ctx); err != nil {
			return fmt.Errorf("postgres: %w", err)
		}
		if err := redisClient.Ping(ctx).Err(); err != nil {
			return fmt.Errorf("redis: %w", err)
		}
		return nil
	})

	router.GET("/api/v1/flights/route", flightHandler.SearchByRoute)
	router.GET("/api/v1/flights/:flightNumber", flightHandler.GetByFlightNumber)
	router.POST("/internal/v1/flights/ingest", flightHandler.Ingest)

	return router
}
