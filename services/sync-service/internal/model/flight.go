package model

import "time"

// FlightStatus mirrors flight-service's FlightStatus enum
// (docs/api-contracts/flight-service.yaml).
type FlightStatus string

const (
	StatusScheduled FlightStatus = "SCHEDULED"
	StatusBoarding  FlightStatus = "BOARDING"
	StatusDelayed   FlightStatus = "DELAYED"
	StatusDeparted  FlightStatus = "DEPARTED"
	StatusLanded    FlightStatus = "LANDED"
	StatusCancelled FlightStatus = "CANCELLED"
)

// FlightSnapshot is sync-service's normalized view of a single flight's
// current status, independent of any provider's wire format — this is the
// canonical shape every FlightDataProvider implementation must produce
// (docs/architecture.md, section 2.4).
type FlightSnapshot struct {
	FlightNumber       string
	AirlineIATA        string
	AirlineName        string
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
}
