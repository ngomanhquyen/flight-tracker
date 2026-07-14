// Package eventbus provides a RabbitMQ publisher for the flight.events
// topic exchange, and the shared FlightEvent wire contract consumers
// depend on (docs/events/event-catalog.md, docs/events/schemas/flight-event.schema.json).
package eventbus

import (
	"strings"
	"time"
)

// EventVersion is the current MAJOR.MINOR of the FlightEvent envelope. See
// the additive versioning rule in docs/events/event-catalog.md.
const EventVersion = "1.0"

// EventType is one of the flight.events routing keys' event types.
type EventType string

const (
	EventFlightCreated   EventType = "FlightCreated"
	EventFlightUpdated   EventType = "FlightUpdated"
	EventFlightDelayed   EventType = "FlightDelayed"
	EventFlightBoarding  EventType = "FlightBoarding"
	EventFlightDeparted  EventType = "FlightDeparted"
	EventFlightLanded    EventType = "FlightLanded"
	EventFlightCancelled EventType = "FlightCancelled"
)

// RoutingKey returns the flight.events routing key for t, e.g.
// EventFlightDelayed -> "flight.delayed".
func (t EventType) RoutingKey() string {
	return "flight." + strings.ToLower(strings.TrimPrefix(string(t), "Flight"))
}

// FlightEvent is the envelope published to the flight.events exchange by
// sync-service and consumed by notification-service (and, additively, any
// future consumer). Field shape must stay in lockstep with
// docs/events/schemas/flight-event.schema.json.
type FlightEvent struct {
	EventID       string      `json:"event_id"`
	EventType     EventType   `json:"event_type"`
	Version       string      `json:"version"`
	OccurredAt    time.Time   `json:"occurred_at"`
	CorrelationID string      `json:"correlation_id,omitempty"`
	Flight        FlightData  `json:"flight"`
	Metadata      EventMeta   `json:"metadata"`
}

// FlightData is the FlightEvent.flight payload.
type FlightData struct {
	FlightNumber       string     `json:"flight_number"`
	AirlineIATA        string     `json:"airline_iata"`
	OriginIATA         string     `json:"origin_iata"`
	DestinationIATA    string     `json:"destination_iata"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	EstimatedDeparture *time.Time `json:"estimated_departure,omitempty"`
	ActualDeparture    *time.Time `json:"actual_departure,omitempty"`
	ScheduledArrival   time.Time  `json:"scheduled_arrival"`
	EstimatedArrival   *time.Time `json:"estimated_arrival,omitempty"`
	ActualArrival      *time.Time `json:"actual_arrival,omitempty"`
	Gate               *string    `json:"gate,omitempty"`
	Terminal           *string    `json:"terminal,omitempty"`
	Status             string     `json:"status"`
	PreviousStatus     *string    `json:"previous_status,omitempty"`
}

// EventMeta is the FlightEvent.metadata payload.
type EventMeta struct {
	Source   string `json:"source"`
	Provider string `json:"provider,omitempty"`
}
