package service

import (
	"context"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/flighttracker/services/flight-service/internal/model"
	"github.com/flighttracker/services/flight-service/internal/repository"
)

func TestIngest_InvalidatesFlightAndRouteCache(t *testing.T) {
	snapshot := model.Flight{
		FlightNumber:       "VN257",
		OriginIATA:         "HAN",
		DestinationIATA:    "SGN",
		ScheduledDeparture: time.Date(2026, 7, 10, 7, 30, 0, 0, time.UTC),
		Status:             model.StatusBoarding,
	}
	repo := &fakeFlightRepo{ingestResult: repository.IngestResult{Flight: snapshot, Changed: true}}
	cache := &fakeCacheRepo{}

	svc := NewIngestService(repo, cache, zap.NewNop())
	result, err := svc.Ingest(context.Background(), snapshot, "sync-service", map[string]any{"status": "BOARDING"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !result.Changed {
		t.Errorf("expected Changed=true to be passed through from the repository result")
	}
	if repo.ingestCalls != 1 {
		t.Errorf("expected 1 repository Ingest call, got %d", repo.ingestCalls)
	}

	wantFlightKey := "VN257|2026-07-10"
	if len(cache.invalidatedFlights) != 1 || cache.invalidatedFlights[0] != wantFlightKey {
		t.Errorf("expected flight cache invalidated for %q, got %v", wantFlightKey, cache.invalidatedFlights)
	}
	wantRouteKey := "HAN-SGN|2026-07-10"
	if len(cache.invalidatedRoutes) != 1 || cache.invalidatedRoutes[0] != wantRouteKey {
		t.Errorf("expected route cache invalidated for %q, got %v", wantRouteKey, cache.invalidatedRoutes)
	}
}

func TestIngest_RepositoryErrorSkipsCacheInvalidation(t *testing.T) {
	repo := &fakeFlightRepo{ingestErr: context.DeadlineExceeded}
	cache := &fakeCacheRepo{}

	svc := NewIngestService(repo, cache, zap.NewNop())
	_, err := svc.Ingest(context.Background(), model.Flight{FlightNumber: "VN257"}, "sync-service", nil)
	if err == nil {
		t.Fatal("expected error to propagate from the repository")
	}
	if len(cache.invalidatedFlights) != 0 || len(cache.invalidatedRoutes) != 0 {
		t.Errorf("expected no cache invalidation when ingest fails")
	}
}
