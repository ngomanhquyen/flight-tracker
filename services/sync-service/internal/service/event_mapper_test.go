package service

import (
	"testing"

	"github.com/flighttracker/pkg/eventbus"

	"github.com/flighttracker/services/sync-service/internal/model"
)

func strPtr(s string) *string { return &s }

func TestEventTypeFor(t *testing.T) {
	tests := []struct {
		name           string
		previousStatus *string
		newStatus      model.FlightStatus
		want           eventbus.EventType
	}{
		{"first observation", nil, model.StatusScheduled, eventbus.EventFlightCreated},
		{"no status change, other field changed", strPtr("SCHEDULED"), model.StatusScheduled, eventbus.EventFlightUpdated},
		{"transition to delayed", strPtr("SCHEDULED"), model.StatusDelayed, eventbus.EventFlightDelayed},
		{"transition to boarding", strPtr("DELAYED"), model.StatusBoarding, eventbus.EventFlightBoarding},
		{"transition to departed", strPtr("BOARDING"), model.StatusDeparted, eventbus.EventFlightDeparted},
		{"transition to landed", strPtr("DEPARTED"), model.StatusLanded, eventbus.EventFlightLanded},
		{"transition to cancelled", strPtr("SCHEDULED"), model.StatusCancelled, eventbus.EventFlightCancelled},
		{"transition back to scheduled has no dedicated event", strPtr("DELAYED"), model.StatusScheduled, eventbus.EventFlightUpdated},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := eventTypeFor(tt.previousStatus, tt.newStatus)
			if got != tt.want {
				t.Errorf("eventTypeFor(%v, %v) = %v, want %v", tt.previousStatus, tt.newStatus, got, tt.want)
			}
		})
	}
}
