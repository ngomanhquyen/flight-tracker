package service

import (
	"context"
	"time"

	"github.com/flighttracker/services/flight-service/internal/model"
	"github.com/flighttracker/services/flight-service/internal/repository"
)

type fakeFlightRepo struct {
	getFlight    func(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error)
	searchRoute  func(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error)
	ingestResult repository.IngestResult
	ingestErr    error
	ingestCalls  int
}

func (r *fakeFlightRepo) GetByFlightNumberAndDate(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
	return r.getFlight(ctx, flightNumber, date)
}

func (r *fakeFlightRepo) SearchByRoute(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error) {
	return r.searchRoute(ctx, origin, destination, date, limit)
}

func (r *fakeFlightRepo) Ingest(ctx context.Context, snapshot model.Flight, source string, rawPayload map[string]any) (repository.IngestResult, error) {
	r.ingestCalls++
	return r.ingestResult, r.ingestErr
}

type fakeCacheRepo struct {
	flight    *model.Flight
	getErr    error
	setErr    error
	routes    []model.Flight
	routeHit  bool
	getRouteErr error
	setRouteErr error

	invalidatedFlights []string
	invalidatedRoutes  []string
}

func (c *fakeCacheRepo) GetFlight(ctx context.Context, flightNumber string, date time.Time) (model.Flight, bool, error) {
	if c.getErr != nil {
		return model.Flight{}, false, c.getErr
	}
	if c.flight == nil {
		return model.Flight{}, false, nil
	}
	return *c.flight, true, nil
}

func (c *fakeCacheRepo) SetFlight(ctx context.Context, flightNumber string, date time.Time, flight model.Flight) error {
	return c.setErr
}

func (c *fakeCacheRepo) InvalidateFlight(ctx context.Context, flightNumber string, date time.Time) error {
	c.invalidatedFlights = append(c.invalidatedFlights, flightNumber+"|"+date.Format("2006-01-02"))
	return nil
}

func (c *fakeCacheRepo) GetRoute(ctx context.Context, origin, destination string, date time.Time) ([]model.Flight, bool, error) {
	if c.getRouteErr != nil {
		return nil, false, c.getRouteErr
	}
	return c.routes, c.routeHit, nil
}

func (c *fakeCacheRepo) SetRoute(ctx context.Context, origin, destination string, date time.Time, flights []model.Flight) error {
	return c.setRouteErr
}

func (c *fakeCacheRepo) InvalidateRoute(ctx context.Context, origin, destination string, date time.Time) error {
	c.invalidatedRoutes = append(c.invalidatedRoutes, origin+"-"+destination+"|"+date.Format("2006-01-02"))
	return nil
}
