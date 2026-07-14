package repository

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/flighttracker/services/flight-service/internal/model"
)

// CacheRepository is flight-service's cache-aside port in front of
// Postgres (docs/database/flight-service.sql's Redis key documentation).
// A cache miss or Redis unavailability is never fatal to a caller — every
// method degrades to "not cached" rather than returning an error for a
// miss, since search is expected to fall back to Postgres.
type CacheRepository interface {
	GetFlight(ctx context.Context, flightNumber string, date time.Time) (model.Flight, bool, error)
	SetFlight(ctx context.Context, flightNumber string, date time.Time, flight model.Flight) error
	InvalidateFlight(ctx context.Context, flightNumber string, date time.Time) error

	GetRoute(ctx context.Context, origin, destination string, date time.Time) ([]model.Flight, bool, error)
	SetRoute(ctx context.Context, origin, destination string, date time.Time, flights []model.Flight) error
	InvalidateRoute(ctx context.Context, origin, destination string, date time.Time) error
}

type redisCacheRepository struct {
	client *redis.Client
	ttl    time.Duration
}

func NewRedisCacheRepository(client *redis.Client, ttl time.Duration) CacheRepository {
	return &redisCacheRepository{client: client, ttl: ttl}
}

func flightKey(flightNumber string, date time.Time) string {
	return fmt.Sprintf("flight:search:%s:%s", flightNumber, date.Format("2006-01-02"))
}

func routeKey(origin, destination string, date time.Time) string {
	return fmt.Sprintf("flight:route:%s:%s:%s", origin, destination, date.Format("2006-01-02"))
}

func (c *redisCacheRepository) GetFlight(ctx context.Context, flightNumber string, date time.Time) (model.Flight, bool, error) {
	raw, err := c.client.Get(ctx, flightKey(flightNumber, date)).Bytes()
	if errors.Is(err, redis.Nil) {
		return model.Flight{}, false, nil
	}
	if err != nil {
		return model.Flight{}, false, err
	}

	var flight model.Flight
	if err := json.Unmarshal(raw, &flight); err != nil {
		return model.Flight{}, false, err
	}
	return flight, true, nil
}

func (c *redisCacheRepository) SetFlight(ctx context.Context, flightNumber string, date time.Time, flight model.Flight) error {
	raw, err := json.Marshal(flight)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, flightKey(flightNumber, date), raw, c.ttl).Err()
}

func (c *redisCacheRepository) InvalidateFlight(ctx context.Context, flightNumber string, date time.Time) error {
	return c.client.Del(ctx, flightKey(flightNumber, date)).Err()
}

func (c *redisCacheRepository) GetRoute(ctx context.Context, origin, destination string, date time.Time) ([]model.Flight, bool, error) {
	raw, err := c.client.Get(ctx, routeKey(origin, destination, date)).Bytes()
	if errors.Is(err, redis.Nil) {
		return nil, false, nil
	}
	if err != nil {
		return nil, false, err
	}

	var flights []model.Flight
	if err := json.Unmarshal(raw, &flights); err != nil {
		return nil, false, err
	}
	return flights, true, nil
}

func (c *redisCacheRepository) SetRoute(ctx context.Context, origin, destination string, date time.Time, flights []model.Flight) error {
	raw, err := json.Marshal(flights)
	if err != nil {
		return err
	}
	return c.client.Set(ctx, routeKey(origin, destination, date), raw, c.ttl).Err()
}

func (c *redisCacheRepository) InvalidateRoute(ctx context.Context, origin, destination string, date time.Time) error {
	return c.client.Del(ctx, routeKey(origin, destination, date)).Err()
}
