package model

import (
	"time"

	"github.com/google/uuid"
)

// FlightStatus mirrors the flight.flight_status Postgres enum
// (docs/database/flight-service.sql).
type FlightStatus string

const (
	StatusScheduled FlightStatus = "SCHEDULED"
	StatusBoarding  FlightStatus = "BOARDING"
	StatusDelayed   FlightStatus = "DELAYED"
	StatusDeparted  FlightStatus = "DEPARTED"
	StatusLanded    FlightStatus = "LANDED"
	StatusCancelled FlightStatus = "CANCELLED"
)

// Flight is flight-service's canonical read-model entity — the single
// row per physical departure it owns (docs/architecture.md section 2.3).
type Flight struct {
	ID                 uuid.UUID
	FlightNumber       string
	AirlineIATA        string
	AirlineName        *string
	OriginIATA         string
	DestinationIATA    string
	ScheduledDeparture time.Time
	EstimatedDeparture *time.Time
	ActualDeparture    *time.Time
	ScheduledArrival   time.Time
	EstimatedArrival   *time.Time
	ActualArrival      *time.Time
	Gate               *string
	Terminal           *string
	Status             FlightStatus
	AircraftType       *string
	LastSyncedAt       time.Time
}
