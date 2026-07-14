// Package service implements sync-service's poll-diff-ingest-publish cycle
// (docs/diagrams/sequence-diagrams.md, "sync-service poll -> event ->
// notification").
package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/flighttracker/pkg/eventbus"
	"github.com/flighttracker/pkg/logger"

	"github.com/flighttracker/services/sync-service/internal/model"
	"github.com/flighttracker/services/sync-service/internal/repository"
)

// EventPublisher is the subset of *eventbus.Publisher this package
// depends on, so tests can substitute a stub instead of a real broker
// connection.
type EventPublisher interface {
	Publish(ctx context.Context, event eventbus.FlightEvent) error
}

// Summary reports the outcome of one RunOnce call, for logging and for
// cmd/main.go's process exit code.
type Summary struct {
	FlightsSeen     int
	FlightsChanged  int
	EventsPublished int
	Errors          []error
}

// SyncService runs one poll cycle: fetch the provider's current batch,
// diff each flight against the last known hash, ingest changes into
// flight-service, and publish the resulting domain event.
type SyncService struct {
	provider     repository.FlightDataProvider
	snapshots    repository.SnapshotRepository
	pollRuns     repository.PollRunRepository
	flightClient repository.FlightServiceClient
	publisher    EventPublisher
	providerName string
	logger       *zap.Logger
}

func NewSyncService(
	provider repository.FlightDataProvider,
	snapshots repository.SnapshotRepository,
	pollRuns repository.PollRunRepository,
	flightClient repository.FlightServiceClient,
	publisher EventPublisher,
	providerName string,
	log *zap.Logger,
) *SyncService {
	return &SyncService{
		provider:     provider,
		snapshots:    snapshots,
		pollRuns:     pollRuns,
		flightClient: flightClient,
		publisher:    publisher,
		providerName: providerName,
		logger:       log,
	}
}

const eventSource = "sync-service"

// RunOnce executes exactly one poll cycle and records it in sync.poll_runs.
// It never returns an error for a single flight's failure — those are
// collected in Summary.Errors and logged — only for a failure that aborts
// the whole run (starting/finishing the poll_runs row, or the provider
// call itself).
func (s *SyncService) RunOnce(ctx context.Context) (Summary, error) {
	runID, err := s.pollRuns.Start(ctx)
	if err != nil {
		return Summary{}, fmt.Errorf("start poll run: %w", err)
	}

	correlationID := fmt.Sprintf("sync-run-%s", time.Now().UTC().Format("20060102-1504"))
	ctx = logger.WithCorrelationID(ctx, correlationID)
	log := s.logger.With(zap.String("correlation_id", correlationID))

	var summary Summary
	flights, err := s.provider.FetchBatch(ctx)
	if err != nil {
		finishErr := fmt.Errorf("fetch batch: %w", err)
		_ = s.pollRuns.Finish(ctx, runID, 0, 0, 0, finishErr)
		return summary, finishErr
	}

	for _, pf := range flights {
		summary.FlightsSeen++
		if procErr := s.processFlight(ctx, pf, &summary); procErr != nil {
			summary.Errors = append(summary.Errors, procErr)
			log.Error("sync_flight_failed",
				zap.String("flight_number", pf.Snapshot.FlightNumber),
				zap.Error(procErr))
		}
	}

	var runErr error
	if len(summary.Errors) > 0 {
		runErr = fmt.Errorf("%d/%d flights failed", len(summary.Errors), summary.FlightsSeen)
	}
	if err := s.pollRuns.Finish(ctx, runID, summary.FlightsSeen, summary.FlightsChanged, summary.EventsPublished, runErr); err != nil {
		log.Error("poll_run_finish_failed", zap.Error(err))
	}

	log.Info("sync_run_complete",
		zap.Int("flights_seen", summary.FlightsSeen),
		zap.Int("flights_changed", summary.FlightsChanged),
		zap.Int("events_published", summary.EventsPublished),
		zap.Int("errors", len(summary.Errors)),
	)
	return summary, nil
}

func (s *SyncService) processFlight(ctx context.Context, pf repository.ProviderFlight, summary *Summary) error {
	hash, err := snapshotHash(pf.Snapshot)
	if err != nil {
		return fmt.Errorf("hash snapshot: %w", err)
	}

	prevHash, found, err := s.snapshots.GetHash(ctx, pf.Snapshot.FlightNumber, pf.Snapshot.ScheduledDeparture)
	if err != nil {
		return fmt.Errorf("get stored hash: %w", err)
	}
	if found && prevHash == hash {
		if err := s.snapshots.TouchPolledAt(ctx, pf.Snapshot.FlightNumber, pf.Snapshot.ScheduledDeparture); err != nil {
			return fmt.Errorf("touch polled_at: %w", err)
		}
		return nil
	}

	ingestResult, err := s.flightClient.Ingest(ctx, pf.Snapshot, pf.RawPayload, eventSource)
	if err != nil {
		return fmt.Errorf("ingest: %w", err)
	}
	summary.FlightsChanged++

	if ingestResult.Changed {
		event := s.buildEvent(pf.Snapshot, ingestResult.PreviousStatus, logger.CorrelationID(ctx))
		if err := s.publisher.Publish(ctx, event); err != nil {
			// Publish failed: leave the stored hash untouched so the next
			// poll re-detects this as a change against the provider and
			// retries ingest+publish. flight-service's ingest is
			// idempotent, so the retry is safe — its only known gap is if
			// the provider's own state moves on again before the retry,
			// in which case ingest would report changed=false for this
			// now-stale transition and the missed event is not
			// reconstructed (docs/events/event-catalog.md's publish
			// guarantee is explicitly best-effort, not exactly-once).
			return fmt.Errorf("publish event: %w", err)
		}
		summary.EventsPublished++
	}

	if err := s.snapshots.Upsert(ctx, pf.Snapshot.FlightNumber, pf.Snapshot.ScheduledDeparture, hash, pf.RawPayload); err != nil {
		return fmt.Errorf("upsert snapshot: %w", err)
	}
	return nil
}

func (s *SyncService) buildEvent(snap model.FlightSnapshot, previousStatus *string, correlationID string) eventbus.FlightEvent {
	return eventbus.FlightEvent{
		EventID:       uuid.NewString(),
		EventType:     eventTypeFor(previousStatus, snap.Status),
		Version:       eventbus.EventVersion,
		OccurredAt:    time.Now().UTC(),
		CorrelationID: correlationID,
		Flight: eventbus.FlightData{
			FlightNumber:       snap.FlightNumber,
			AirlineIATA:        snap.AirlineIATA,
			OriginIATA:         snap.OriginIATA,
			DestinationIATA:    snap.DestinationIATA,
			ScheduledDeparture: snap.ScheduledDeparture,
			EstimatedDeparture: snap.EstimatedDeparture,
			ActualDeparture:    snap.ActualDeparture,
			ScheduledArrival:   snap.ScheduledArrival,
			EstimatedArrival:   snap.EstimatedArrival,
			ActualArrival:      snap.ActualArrival,
			Gate:               snap.Gate,
			Terminal:           snap.Terminal,
			Status:             string(snap.Status),
			PreviousStatus:     previousStatus,
		},
		Metadata: eventbus.EventMeta{
			Source:   eventSource,
			Provider: s.providerName,
		},
	}
}
