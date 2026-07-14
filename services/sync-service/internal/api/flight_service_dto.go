// Package api holds wire-format DTOs kept separate from domain models
// (internal/model), per docs/architecture.md's Clean Architecture layering.
package api

import "time"

// IngestRequestDTO is the request body of flight-service's
// POST /internal/v1/flights/ingest (docs/api-contracts/flight-service.yaml).
type IngestRequestDTO struct {
	FlightNumber       string         `json:"flight_number"`
	AirlineIATA        string         `json:"airline_iata,omitempty"`
	AirlineName        string         `json:"airline_name,omitempty"`
	OriginIATA         string         `json:"origin_iata"`
	DestinationIATA    string         `json:"destination_iata"`
	ScheduledDeparture time.Time      `json:"scheduled_departure"`
	EstimatedDeparture *time.Time     `json:"estimated_departure,omitempty"`
	ActualDeparture    *time.Time     `json:"actual_departure,omitempty"`
	ScheduledArrival   time.Time      `json:"scheduled_arrival"`
	EstimatedArrival   *time.Time     `json:"estimated_arrival,omitempty"`
	ActualArrival      *time.Time     `json:"actual_arrival,omitempty"`
	Gate               *string        `json:"gate,omitempty"`
	Terminal           *string        `json:"terminal,omitempty"`
	Status             string         `json:"status"`
	AircraftType       *string        `json:"aircraft_type,omitempty"`
	Source             string         `json:"source"`
	RawPayload         map[string]any `json:"raw_payload,omitempty"`
}

// FlightResponseDTO mirrors flight-service's FlightResponse schema.
type FlightResponseDTO struct {
	ID                 string     `json:"id"`
	FlightNumber       string     `json:"flight_number"`
	Status             string     `json:"status"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	LastSyncedAt       time.Time  `json:"last_synced_at"`
	Gate               *string    `json:"gate"`
	Terminal           *string    `json:"terminal"`
}

// IngestResponseDTO is the response body of the ingest endpoint.
type IngestResponseDTO struct {
	Flight         FlightResponseDTO `json:"flight"`
	PreviousStatus *string           `json:"previous_status"`
	Changed        bool              `json:"changed"`
}

// ErrorResponseDTO is the standard error envelope shared across services.
type ErrorResponseDTO struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}
