package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/flighttracker/services/flight-service/internal/model"
	"github.com/flighttracker/services/flight-service/internal/repository"
)

// IngestService is the only write path into flight-service's data
// (docs/architecture.md section 2.3), used exclusively by sync-service.
type IngestService struct {
	repo   repository.FlightRepository
	cache  repository.CacheRepository
	logger *zap.Logger
}

func NewIngestService(repo repository.FlightRepository, cache repository.CacheRepository, log *zap.Logger) *IngestService {
	return &IngestService{repo: repo, cache: cache, logger: log}
}

func (s *IngestService) Ingest(ctx context.Context, snapshot model.Flight, source string, rawPayload map[string]any) (repository.IngestResult, error) {
	result, err := s.repo.Ingest(ctx, snapshot, source, rawPayload)
	if err != nil {
		return repository.IngestResult{}, err
	}

	date := time.Date(
		snapshot.ScheduledDeparture.Year(), snapshot.ScheduledDeparture.Month(), snapshot.ScheduledDeparture.Day(),
		0, 0, 0, 0, time.UTC,
	)
	if err := s.cache.InvalidateFlight(ctx, snapshot.FlightNumber, date); err != nil {
		s.logger.Warn("cache_invalidate_failed", zap.String("flight_number", snapshot.FlightNumber), zap.Error(err))
	}
	if err := s.cache.InvalidateRoute(ctx, snapshot.OriginIATA, snapshot.DestinationIATA, date); err != nil {
		s.logger.Warn("cache_invalidate_failed",
			zap.String("origin", snapshot.OriginIATA), zap.String("destination", snapshot.DestinationIATA), zap.Error(err))
	}

	return result, nil
}
