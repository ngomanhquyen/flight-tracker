// Package service implements flight-service's use cases: cache-aside
// search (docs/diagrams/sequence-diagrams.md, "/flight VN257 search") and
// ingest (docs/architecture.md section 2.3).
package service

import (
	"context"
	"time"

	"go.uber.org/zap"

	"github.com/flighttracker/services/flight-service/internal/model"
	"github.com/flighttracker/services/flight-service/internal/repository"
)

// FlightSearchService serves the two public search endpoints, checking
// Redis before falling back to Postgres (cache-aside). A cache read/write
// failure is logged and treated as a miss/no-op — Redis is an
// optimization, never a dependency search correctness relies on.
type FlightSearchService struct {
	repo            repository.FlightRepository
	cache           repository.CacheRepository
	maxRouteResults int
	logger          *zap.Logger
}

func NewFlightSearchService(repo repository.FlightRepository, cache repository.CacheRepository, maxRouteResults int, log *zap.Logger) *FlightSearchService {
	return &FlightSearchService{repo: repo, cache: cache, maxRouteResults: maxRouteResults, logger: log}
}

func (s *FlightSearchService) GetByFlightNumber(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
	if cached, hit, err := s.cache.GetFlight(ctx, flightNumber, date); err != nil {
		s.logger.Warn("cache_get_failed", zap.String("flight_number", flightNumber), zap.Error(err))
	} else if hit {
		return cached, nil
	}

	flight, err := s.repo.GetByFlightNumberAndDate(ctx, flightNumber, date)
	if err != nil {
		return model.Flight{}, err
	}

	if err := s.cache.SetFlight(ctx, flightNumber, date, flight); err != nil {
		s.logger.Warn("cache_set_failed", zap.String("flight_number", flightNumber), zap.Error(err))
	}
	return flight, nil
}

// SearchByRoute always fetches/caches up to maxRouteResults and truncates
// to the caller's requested limit afterward, so the cache key doesn't need
// to vary per limit (docs/database/flight-service.sql's Redis key
// documentation has no limit component).
func (s *FlightSearchService) SearchByRoute(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error) {
	flights, hit, err := s.cache.GetRoute(ctx, origin, destination, date)
	if err != nil {
		s.logger.Warn("cache_get_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		hit = false
	}

	if !hit {
		flights, err = s.repo.SearchByRoute(ctx, origin, destination, date, s.maxRouteResults)
		if err != nil {
			return nil, err
		}
		if err := s.cache.SetRoute(ctx, origin, destination, date, flights); err != nil {
			s.logger.Warn("cache_set_failed", zap.String("origin", origin), zap.String("destination", destination), zap.Error(err))
		}
	}

	if limit > 0 && limit < len(flights) {
		flights = flights[:limit]
	}
	return flights, nil
}
