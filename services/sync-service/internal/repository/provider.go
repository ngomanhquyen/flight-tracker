package repository

import (
	"context"

	"github.com/flighttracker/services/sync-service/internal/model"
)

// ProviderFlight pairs a normalized snapshot with the original upstream
// payload, kept for audit (IngestRequestDTO.RawPayload ->
// flight_status_history, per docs/api-contracts/flight-service.yaml).
type ProviderFlight struct {
	Snapshot   model.FlightSnapshot
	RawPayload map[string]any
}

// FlightDataProvider is sync-service's port to the public flight-data API
// (docs/architecture.md section 2.4). Each implementation owns its own
// targeting strategy (which flights/routes/airports it tracks) and its own
// provider-specific credentials/rate limiting; callers only see the
// normalized FlightSnapshot shape.
type FlightDataProvider interface {
	FetchBatch(ctx context.Context) ([]ProviderFlight, error)
}
