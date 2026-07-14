package repository

import (
	"context"
	"testing"
	"time"

	"github.com/flighttracker/services/sync-service/internal/model"
)

func TestFakeProvider_VN257CyclesThroughEveryStatus(t *testing.T) {
	// A 30-minute-aligned reference instant: block start 10:00, so
	// scheduled departure lands at 10:08 (fakeDepartureOffset), matching
	// the phase boundaries documented in fake_provider.go.
	blockStart := time.Date(2026, 7, 10, 10, 0, 0, 0, time.UTC)

	tests := []struct {
		name       string
		offset     time.Duration
		wantStatus model.FlightStatus
	}{
		{"before delay window", 0, model.StatusScheduled},
		{"in delay window", 3 * time.Minute, model.StatusDelayed},
		{"in boarding window", 7 * time.Minute, model.StatusBoarding},
		{"after actual departure", 15 * time.Minute, model.StatusDeparted},
		{"after actual arrival", 25 * time.Minute, model.StatusLanded},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			now := blockStart.Add(tt.offset)
			p := NewFakeProvider(func() time.Time { return now })

			flights, err := p.FetchBatch(context.Background())
			if err != nil {
				t.Fatalf("FetchBatch: %v", err)
			}

			vn257 := findFlight(t, flights, "VN257")
			if vn257.Status != tt.wantStatus {
				t.Errorf("status = %v, want %v", vn257.Status, tt.wantStatus)
			}
		})
	}
}

func TestFakeProvider_VN9999AlwaysCancelledAndStableWithinDay(t *testing.T) {
	morning := time.Date(2026, 7, 10, 8, 0, 0, 0, time.UTC)
	evening := time.Date(2026, 7, 10, 20, 0, 0, 0, time.UTC)

	pMorning := NewFakeProvider(func() time.Time { return morning })
	pEvening := NewFakeProvider(func() time.Time { return evening })

	morningFlights, err := pMorning.FetchBatch(context.Background())
	if err != nil {
		t.Fatalf("FetchBatch: %v", err)
	}
	eveningFlights, err := pEvening.FetchBatch(context.Background())
	if err != nil {
		t.Fatalf("FetchBatch: %v", err)
	}

	morningVN9999 := findFlight(t, morningFlights, "VN9999")
	eveningVN9999 := findFlight(t, eveningFlights, "VN9999")

	if morningVN9999.Status != model.StatusCancelled || eveningVN9999.Status != model.StatusCancelled {
		t.Errorf("expected VN9999 always CANCELLED, got %v / %v", morningVN9999.Status, eveningVN9999.Status)
	}
	if !morningVN9999.ScheduledDeparture.Equal(eveningVN9999.ScheduledDeparture) {
		t.Errorf("expected stable scheduled_departure within the same day, got %v vs %v",
			morningVN9999.ScheduledDeparture, eveningVN9999.ScheduledDeparture)
	}
}

func findFlight(t *testing.T, flights []ProviderFlight, flightNumber string) model.FlightSnapshot {
	t.Helper()
	for _, f := range flights {
		if f.Snapshot.FlightNumber == flightNumber {
			return f.Snapshot
		}
	}
	t.Fatalf("flight %s not found in batch", flightNumber)
	return model.FlightSnapshot{}
}
