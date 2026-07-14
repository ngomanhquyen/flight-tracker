package repository

import (
	"testing"
	"time"
)

func TestFlightsDiffer(t *testing.T) {
	base := flightRow{
		Status:           "SCHEDULED",
		ScheduledArrival: time.Date(2026, 7, 10, 9, 40, 0, 0, time.UTC),
	}

	tests := []struct {
		name     string
		mutate   func(r flightRow) flightRow
		wantDiff bool
	}{
		{
			name:     "identical",
			mutate:   func(r flightRow) flightRow { return r },
			wantDiff: false,
		},
		{
			name: "status changed",
			mutate: func(r flightRow) flightRow {
				r.Status = "BOARDING"
				return r
			},
			wantDiff: true,
		},
		{
			name: "gate assigned",
			mutate: func(r flightRow) flightRow {
				gate := "12"
				r.Gate = &gate
				return r
			},
			wantDiff: true,
		},
		{
			name: "estimated departure set",
			mutate: func(r flightRow) flightRow {
				t := time.Date(2026, 7, 10, 7, 41, 0, 0, time.UTC)
				r.EstimatedDeparture = &t
				return r
			},
			wantDiff: true,
		},
		{
			name: "scheduled arrival revised",
			mutate: func(r flightRow) flightRow {
				r.ScheduledArrival = base.ScheduledArrival.Add(10 * time.Minute)
				return r
			},
			wantDiff: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			incoming := tt.mutate(base)
			if got := flightsDiffer(base, incoming); got != tt.wantDiff {
				t.Errorf("flightsDiffer() = %v, want %v", got, tt.wantDiff)
			}
		})
	}
}
