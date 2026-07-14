package repository

import (
	"context"
	"time"

	"github.com/flighttracker/services/sync-service/internal/model"
)

// FakeProvider is a deterministic stand-in FlightDataProvider used until a
// real public flight-data API is wired in (see config.Provider.Name, which
// cmd/main.go switches on). It has no external dependencies or
// credentials, so the service runs and integration-tests out of the box.
//
// Its demo flight (VN257, HAN->SGN) walks through
// SCHEDULED -> DELAYED -> BOARDING -> DEPARTED -> LANDED once per
// 30-minute wall-clock block, purely as a function of the current time —
// so running sync-service repeatedly over a real 30-minute window
// demonstrates every status transition (and therefore every event type)
// against real Postgres/RabbitMQ infrastructure. A second demo flight
// (VN9999) is always CANCELLED, to exercise that event path without
// waiting on the clock.
type FakeProvider struct {
	// Now defaults to time.Now; overridable (Config.Provider.FakeNowOverride)
	// so a specific phase can be exercised on demand instead of waiting
	// for the wall clock to reach it.
	Now func() time.Time
}

// NewFakeProvider builds a FakeProvider. now may be nil, in which case
// time.Now is used.
func NewFakeProvider(now func() time.Time) *FakeProvider {
	if now == nil {
		now = time.Now
	}
	return &FakeProvider{Now: now}
}

const (
	fakeCyclePeriod     = 30 * time.Minute
	fakeDepartureOffset = 8 * time.Minute  // scheduled departure = block start + this
	fakeFlightDuration  = 10 * time.Minute // scheduled arrival = scheduled departure + this
	fakeDelay           = 3 * time.Minute
)

func (p *FakeProvider) FetchBatch(ctx context.Context) ([]ProviderFlight, error) {
	now := p.Now().UTC()
	return []ProviderFlight{
		p.vn257(now),
		p.vn9999(now),
	}, nil
}

// vn257 cycles through every non-terminal-forever status based on how far
// `now` is into the current fakeCyclePeriod block.
func (p *FakeProvider) vn257(now time.Time) ProviderFlight {
	block := now.Truncate(fakeCyclePeriod)
	scheduledDeparture := block.Add(fakeDepartureOffset)
	scheduledArrival := scheduledDeparture.Add(fakeFlightDuration)
	delayedDeparture := scheduledDeparture.Add(fakeDelay)
	delayedArrival := scheduledArrival.Add(fakeDelay)

	delayStart := scheduledDeparture.Add(-6 * time.Minute)
	boardingStart := scheduledDeparture.Add(-3 * time.Minute)

	snapshot := model.FlightSnapshot{
		FlightNumber:       "VN257",
		AirlineIATA:        "VN",
		AirlineName:        "Vietnam Airlines",
		OriginIATA:         "HAN",
		DestinationIATA:    "SGN",
		ScheduledDeparture: scheduledDeparture,
		ScheduledArrival:   scheduledArrival,
		AircraftType:       strPtr("A321"),
	}

	gate, terminal := strPtr("12"), strPtr("T1")

	switch {
	case now.Before(delayStart):
		snapshot.Status = model.StatusScheduled
	case now.Before(boardingStart):
		snapshot.Status = model.StatusDelayed
		snapshot.EstimatedDeparture = &delayedDeparture
		snapshot.EstimatedArrival = &delayedArrival
	case now.Before(delayedDeparture):
		snapshot.Status = model.StatusBoarding
		snapshot.EstimatedDeparture = &delayedDeparture
		snapshot.EstimatedArrival = &delayedArrival
		snapshot.Gate, snapshot.Terminal = gate, terminal
	case now.Before(delayedArrival):
		snapshot.Status = model.StatusDeparted
		snapshot.ActualDeparture = &delayedDeparture
		snapshot.EstimatedArrival = &delayedArrival
		snapshot.Gate, snapshot.Terminal = gate, terminal
	default:
		snapshot.Status = model.StatusLanded
		snapshot.ActualDeparture = &delayedDeparture
		snapshot.ActualArrival = &delayedArrival
		snapshot.Gate, snapshot.Terminal = gate, terminal
	}

	return ProviderFlight{Snapshot: snapshot, RawPayload: rawPayloadFor(snapshot)}
}

// vn9999 is always CANCELLED, anchored to a fixed daily scheduled
// departure so repeated polls within the same day are stable (hash
// unchanged) rather than manufacturing a "new" flight instance every poll.
func (p *FakeProvider) vn9999(now time.Time) ProviderFlight {
	dayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	scheduledDeparture := dayStart.Add(9 * time.Hour)

	snapshot := model.FlightSnapshot{
		FlightNumber:       "VN9999",
		AirlineIATA:        "VN",
		AirlineName:        "Vietnam Airlines",
		OriginIATA:         "SGN",
		DestinationIATA:    "HAN",
		ScheduledDeparture: scheduledDeparture,
		ScheduledArrival:   scheduledDeparture.Add(2*time.Hour + 10*time.Minute),
		Status:             model.StatusCancelled,
		AircraftType:       strPtr("A321"),
	}
	return ProviderFlight{Snapshot: snapshot, RawPayload: rawPayloadFor(snapshot)}
}

func strPtr(s string) *string { return &s }

func rawPayloadFor(s model.FlightSnapshot) map[string]any {
	payload := map[string]any{
		"flight_number":       s.FlightNumber,
		"airline_iata":        s.AirlineIATA,
		"origin_iata":         s.OriginIATA,
		"destination_iata":    s.DestinationIATA,
		"scheduled_departure": s.ScheduledDeparture,
		"scheduled_arrival":   s.ScheduledArrival,
		"status":              string(s.Status),
	}
	if s.EstimatedDeparture != nil {
		payload["estimated_departure"] = *s.EstimatedDeparture
	}
	if s.ActualDeparture != nil {
		payload["actual_departure"] = *s.ActualDeparture
	}
	if s.EstimatedArrival != nil {
		payload["estimated_arrival"] = *s.EstimatedArrival
	}
	if s.ActualArrival != nil {
		payload["actual_arrival"] = *s.ActualArrival
	}
	if s.Gate != nil {
		payload["gate"] = *s.Gate
	}
	if s.Terminal != nil {
		payload["terminal"] = *s.Terminal
	}
	return payload
}
