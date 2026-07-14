package repository

import (
	"context"
	"fmt"
	"net/http"
	"net/url"
	"time"

	"github.com/flighttracker/services/bot-service/internal/api"
	"github.com/flighttracker/services/bot-service/internal/model"
)

// FlightClient is bot-service's port to flight-service's public search API
// (docs/api-contracts/flight-service.yaml). It is the only place in
// bot-service that knows flight-service's wire format.
type FlightClient interface {
	GetByFlightNumber(ctx context.Context, flightNumber string) (*model.FlightInfo, error)
	SearchByRoute(ctx context.Context, origin, destination string, limit int) ([]model.FlightInfo, error)
}

type httpFlightClient struct {
	restClient
}

func NewFlightClient(baseURL string, timeout time.Duration) FlightClient {
	return &httpFlightClient{
		restClient: restClient{
			baseURL:    baseURL,
			httpClient: &http.Client{Timeout: timeout},
		},
	}
}

func (c *httpFlightClient) GetByFlightNumber(ctx context.Context, flightNumber string) (*model.FlightInfo, error) {
	var dto api.FlightResponseDTO
	path := fmt.Sprintf("/api/v1/flights/%s", url.PathEscape(flightNumber))
	if err := c.do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}
	info := toFlightInfo(dto)
	return &info, nil
}

func (c *httpFlightClient) SearchByRoute(ctx context.Context, origin, destination string, limit int) ([]model.FlightInfo, error) {
	q := url.Values{}
	q.Set("origin", origin)
	q.Set("destination", destination)
	if limit > 0 {
		q.Set("limit", fmt.Sprintf("%d", limit))
	}

	var dto api.RouteSearchResponseDTO
	path := "/api/v1/flights/route?" + q.Encode()
	if err := c.do(ctx, http.MethodGet, path, nil, &dto); err != nil {
		return nil, err
	}

	flights := make([]model.FlightInfo, 0, len(dto.Items))
	for _, item := range dto.Items {
		flights = append(flights, toFlightInfo(item))
	}
	return flights, nil
}

func toFlightInfo(dto api.FlightResponseDTO) model.FlightInfo {
	return model.FlightInfo{
		FlightNumber:       dto.FlightNumber,
		AirlineIATA:        dto.AirlineIATA,
		AirlineName:        dto.AirlineName,
		OriginIATA:         dto.OriginIATA,
		DestinationIATA:    dto.DestinationIATA,
		ScheduledDeparture: dto.ScheduledDeparture,
		EstimatedDeparture: dto.EstimatedDeparture,
		ActualDeparture:    dto.ActualDeparture,
		ScheduledArrival:   dto.ScheduledArrival,
		EstimatedArrival:   dto.EstimatedArrival,
		ActualArrival:      dto.ActualArrival,
		Gate:               dto.Gate,
		Terminal:           dto.Terminal,
		Status:             model.FlightStatus(dto.Status),
		AircraftType:       dto.AircraftType,
	}
}
