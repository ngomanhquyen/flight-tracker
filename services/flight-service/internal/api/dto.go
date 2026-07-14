// Package api holds wire-format DTOs matching
// docs/api-contracts/flight-service.yaml, kept separate from the domain
// model in internal/model.
package api

import "time"

// FlightResponseDTO mirrors the FlightResponse schema. Nullable fields are
// always present in the JSON (as null when unset) rather than omitted, so
// clients that decode into non-pointer fields (see bot-service's
// FlightResponseDTO) behave identically whether a field is null or absent.
type FlightResponseDTO struct {
	ID                 string     `json:"id"`
	FlightNumber       string     `json:"flight_number"`
	AirlineIATA        string     `json:"airline_iata"`
	AirlineName        *string    `json:"airline_name"`
	OriginIATA         string     `json:"origin_iata"`
	DestinationIATA    string     `json:"destination_iata"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	EstimatedDeparture *time.Time `json:"estimated_departure"`
	ActualDeparture    *time.Time `json:"actual_departure"`
	ScheduledArrival   time.Time  `json:"scheduled_arrival"`
	EstimatedArrival   *time.Time `json:"estimated_arrival"`
	ActualArrival      *time.Time `json:"actual_arrival"`
	Gate               *string    `json:"gate"`
	Terminal           *string    `json:"terminal"`
	Status             string     `json:"status"`
	AircraftType       *string    `json:"aircraft_type"`
	LastSyncedAt       time.Time  `json:"last_synced_at"`
}

// RouteSearchResponseDTO is the body of GET /api/v1/flights/route.
type RouteSearchResponseDTO struct {
	Items []FlightResponseDTO `json:"items"`
	Total int                 `json:"total"`
}

// IngestRequestDTO is the body of POST /internal/v1/flights/ingest, sent
// by sync-service (see services/sync-service/internal/api's identically
// shaped DTO on the client side).
type IngestRequestDTO struct {
	FlightNumber       string         `json:"flight_number" binding:"required"`
	AirlineIATA        string         `json:"airline_iata"`
	AirlineName        string         `json:"airline_name"`
	OriginIATA         string         `json:"origin_iata" binding:"required"`
	DestinationIATA    string         `json:"destination_iata" binding:"required"`
	ScheduledDeparture time.Time      `json:"scheduled_departure" binding:"required"`
	EstimatedDeparture *time.Time     `json:"estimated_departure"`
	ActualDeparture    *time.Time     `json:"actual_departure"`
	ScheduledArrival   time.Time      `json:"scheduled_arrival" binding:"required"`
	EstimatedArrival   *time.Time     `json:"estimated_arrival"`
	ActualArrival      *time.Time     `json:"actual_arrival"`
	Gate               *string        `json:"gate"`
	Terminal           *string        `json:"terminal"`
	Status             string         `json:"status" binding:"required"`
	AircraftType       *string        `json:"aircraft_type"`
	Source             string         `json:"source" binding:"required"`
	RawPayload         map[string]any `json:"raw_payload"`
}

// IngestResponseDTO is the body returned by the ingest endpoint.
type IngestResponseDTO struct {
	Flight         FlightResponseDTO `json:"flight"`
	PreviousStatus *string           `json:"previous_status"`
	Changed        bool              `json:"changed"`
}

// ErrorResponseDTO is the standard error envelope shared across services.
type ErrorResponseDTO struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id,omitempty"`
}
