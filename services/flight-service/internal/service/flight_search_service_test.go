package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"go.uber.org/zap"

	"github.com/flighttracker/services/flight-service/internal/model"
)

func TestGetByFlightNumber_CacheHit_SkipsRepo(t *testing.T) {
	cached := model.Flight{FlightNumber: "VN257", Status: model.StatusBoarding}
	repo := &fakeFlightRepo{
		getFlight: func(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
			t.Fatal("repo should not be called on a cache hit")
			return model.Flight{}, nil
		},
	}
	cache := &fakeCacheRepo{flight: &cached}

	svc := NewFlightSearchService(repo, cache, 50, zap.NewNop())
	got, err := svc.GetByFlightNumber(context.Background(), "VN257", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != model.StatusBoarding {
		t.Errorf("expected cached flight returned, got %+v", got)
	}
}

func TestGetByFlightNumber_CacheMiss_FallsBackToRepoAndPopulatesCache(t *testing.T) {
	repoFlight := model.Flight{FlightNumber: "VN257", Status: model.StatusScheduled}
	repo := &fakeFlightRepo{
		getFlight: func(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
			return repoFlight, nil
		},
	}
	cache := &fakeCacheRepo{} // no cached flight -> miss

	svc := NewFlightSearchService(repo, cache, 50, zap.NewNop())
	got, err := svc.GetByFlightNumber(context.Background(), "VN257", time.Now())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got.Status != model.StatusScheduled {
		t.Errorf("expected repo flight returned, got %+v", got)
	}
}

func TestGetByFlightNumber_CacheError_FallsBackToRepo(t *testing.T) {
	repoFlight := model.Flight{FlightNumber: "VN257", Status: model.StatusLanded}
	repo := &fakeFlightRepo{
		getFlight: func(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
			return repoFlight, nil
		},
	}
	cache := &fakeCacheRepo{getErr: errors.New("redis down")}

	svc := NewFlightSearchService(repo, cache, 50, zap.NewNop())
	got, err := svc.GetByFlightNumber(context.Background(), "VN257", time.Now())
	if err != nil {
		t.Fatalf("expected cache errors to be swallowed, got: %v", err)
	}
	if got.Status != model.StatusLanded {
		t.Errorf("expected repo fallback flight, got %+v", got)
	}
}

func TestSearchByRoute_TruncatesToRequestedLimit(t *testing.T) {
	full := make([]model.Flight, 5)
	for i := range full {
		full[i] = model.Flight{FlightNumber: "VN257"}
	}
	repo := &fakeFlightRepo{
		searchRoute: func(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error) {
			return full, nil
		},
	}
	cache := &fakeCacheRepo{}

	svc := NewFlightSearchService(repo, cache, 50, zap.NewNop())
	got, err := svc.SearchByRoute(context.Background(), "HAN", "SGN", time.Now(), 2)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 2 {
		t.Errorf("expected result truncated to 2, got %d", len(got))
	}
}

func TestSearchByRoute_CacheHitSkipsRepoAndAppliesLimit(t *testing.T) {
	cached := []model.Flight{{FlightNumber: "A"}, {FlightNumber: "B"}, {FlightNumber: "C"}}
	repo := &fakeFlightRepo{
		searchRoute: func(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error) {
			t.Fatal("repo should not be called on a cache hit")
			return nil, nil
		},
	}
	cache := &fakeCacheRepo{routes: cached, routeHit: true}

	svc := NewFlightSearchService(repo, cache, 50, zap.NewNop())
	got, err := svc.SearchByRoute(context.Background(), "HAN", "SGN", time.Now(), 1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(got) != 1 || got[0].FlightNumber != "A" {
		t.Errorf("expected cached results truncated to [A], got %+v", got)
	}
}
