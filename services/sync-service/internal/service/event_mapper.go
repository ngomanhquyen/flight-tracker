package service

import (
	"github.com/flighttracker/pkg/eventbus"

	"github.com/flighttracker/services/sync-service/internal/model"
)

// eventTypeFor picks the flight.events routing/event type for a status
// change, per the rule in docs/events/event-catalog.md: FlightCreated on
// first observation, a status-specific event when new_status differs from
// previous_status, and FlightUpdated for any other change (e.g. a gate
// change while status stays the same, or a transition into SCHEDULED,
// which has no dedicated event type).
func eventTypeFor(previousStatus *string, newStatus model.FlightStatus) eventbus.EventType {
	if previousStatus == nil {
		return eventbus.EventFlightCreated
	}
	if *previousStatus == string(newStatus) {
		return eventbus.EventFlightUpdated
	}

	switch newStatus {
	case model.StatusDelayed:
		return eventbus.EventFlightDelayed
	case model.StatusBoarding:
		return eventbus.EventFlightBoarding
	case model.StatusDeparted:
		return eventbus.EventFlightDeparted
	case model.StatusLanded:
		return eventbus.EventFlightLanded
	case model.StatusCancelled:
		return eventbus.EventFlightCancelled
	default:
		return eventbus.EventFlightUpdated
	}
}
