package repository

import (
	"context"
	"net/http"
	"time"

	"github.com/flighttracker/services/sync-service/internal/api"
	"github.com/flighttracker/services/sync-service/internal/model"
)

// IngestResult is the outcome of ingesting one snapshot into flight-service.
type IngestResult struct {
	PreviousStatus *string
	NewStatus      string
	Changed        bool
}

// FlightServiceClient is sync-service's port to flight-service's internal
// ingest endpoint (docs/api-contracts/flight-service.yaml) — the only
// write path into the flight schema, per docs/architecture.md.
type FlightServiceClient interface {
	Ingest(ctx context.Context, snapshot model.FlightSnapshot, rawPayload map[string]any, source string) (IngestResult, error)
}

type httpFlightServiceClient struct {
	restClient
}

func NewFlightServiceClient(baseURL string, timeout time.Duration) FlightServiceClient {
	return &httpFlightServiceClient{
		restClient: restClient{
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: timeout},
		},
	}
}

func (c *httpFlightServiceClient) Ingest(ctx context.Context, s model.FlightSnapshot, rawPayload map[string]any, source string) (IngestResult, error) {
	req := api.IngestRequestDTO{
		FlightNumber:       s.FlightNumber,
		AirlineIATA:        s.AirlineIATA,
		AirlineName:        s.AirlineName,
		OriginIATA:         s.OriginIATA,
		DestinationIATA:    s.DestinationIATA,
		ScheduledDeparture: s.ScheduledDeparture,
		EstimatedDeparture: s.EstimatedDeparture,
		ActualDeparture:    s.ActualDeparture,
		ScheduledArrival:   s.ScheduledArrival,
		EstimatedArrival:   s.EstimatedArrival,
		ActualArrival:      s.ActualArrival,
		Gate:               s.Gate,
		Terminal:           s.Terminal,
		Status:             string(s.Status),
		AircraftType:       s.AircraftType,
		Source:             source,
		RawPayload:         rawPayload,
	}

	var dto api.IngestResponseDTO
	if err := c.do(ctx, http.MethodPost, "/internal/v1/flights/ingest", req, &dto); err != nil {
		return IngestResult{}, err
	}

	return IngestResult{
		PreviousStatus: dto.PreviousStatus,
		NewStatus:      dto.Flight.Status,
		Changed:        dto.Changed,
	}, nil
}
