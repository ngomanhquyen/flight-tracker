package model

import "time"

// FlightStatus mirrors flight-service's flight.flight_status enum
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

// FlightInfo is bot-service's domain view of a flight, decoded from
// flight-service's FlightResponse (docs/api-contracts/flight-service.yaml).
type FlightInfo struct {
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
	Gate               string
	Terminal           string
	Status             FlightStatus
	AircraftType       string
}
