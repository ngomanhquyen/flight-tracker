// Package api holds wire-format DTOs for the REST APIs bot-service calls
// (flight-service, subscription-service), kept separate from the domain
// model in internal/model so a change to an upstream contract touches only
// this package plus the repository layer that converts DTO <-> model.
package api

import "time"

// FlightResponseDTO mirrors FlightResponse in
// docs/api-contracts/flight-service.yaml.
type FlightResponseDTO struct {
	ID                 string     `json:"id"`
	FlightNumber       string     `json:"flight_number"`
	AirlineIATA        string     `json:"airline_iata"`
	AirlineName        string     `json:"airline_name"`
	OriginIATA         string     `json:"origin_iata"`
	DestinationIATA    string     `json:"destination_iata"`
	ScheduledDeparture time.Time  `json:"scheduled_departure"`
	EstimatedDeparture *time.Time `json:"estimated_departure"`
	ActualDeparture    *time.Time `json:"actual_departure"`
	ScheduledArrival   time.Time  `json:"scheduled_arrival"`
	EstimatedArrival   *time.Time `json:"estimated_arrival"`
	ActualArrival      *time.Time `json:"actual_arrival"`
	Gate               string     `json:"gate"`
	Terminal           string     `json:"terminal"`
	Status             string     `json:"status"`
	AircraftType       string     `json:"aircraft_type"`
	LastSyncedAt       time.Time  `json:"last_synced_at"`
}

// RouteSearchResponseDTO mirrors the /api/v1/flights/route response body.
type RouteSearchResponseDTO struct {
	Items []FlightResponseDTO `json:"items"`
	Total int                 `json:"total"`
}

// ErrorResponseDTO mirrors ErrorResponse, shared by both upstream APIs.
type ErrorResponseDTO struct {
	Code          string `json:"code"`
	Message       string `json:"message"`
	CorrelationID string `json:"correlation_id"`
}
