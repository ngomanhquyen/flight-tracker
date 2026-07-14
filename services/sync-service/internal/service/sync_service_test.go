package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"go.uber.org/zap"

	"github.com/flighttracker/pkg/eventbus"

	"github.com/flighttracker/services/sync-service/internal/model"
	"github.com/flighttracker/services/sync-service/internal/repository"
)

// --- test doubles -----------------------------------------------------

type fakeProvider struct {
	flights []repository.ProviderFlight
	err     error
}

func (f *fakeProvider) FetchBatch(ctx context.Context) ([]repository.ProviderFlight, error) {
	return f.flights, f.err
}

type snapshotKey struct {
	flightNumber string
	scheduled    time.Time
}

type fakeSnapshotRepo struct {
	hashes   map[snapshotKey]string
	touched  []snapshotKey
	upserted []snapshotKey
}

func newFakeSnapshotRepo() *fakeSnapshotRepo {
	return &fakeSnapshotRepo{hashes: map[snapshotKey]string{}}
}

func (r *fakeSnapshotRepo) GetHash(ctx context.Context, flightNumber string, scheduledDeparture time.Time) (string, bool, error) {
	h, ok := r.hashes[snapshotKey{flightNumber, scheduledDeparture}]
	return h, ok, nil
}

func (r *fakeSnapshotRepo) TouchPolledAt(ctx context.Context, flightNumber string, scheduledDeparture time.Time) error {
	r.touched = append(r.touched, snapshotKey{flightNumber, scheduledDeparture})
	return nil
}

func (r *fakeSnapshotRepo) Upsert(ctx context.Context, flightNumber string, scheduledDeparture time.Time, hash string, rawPayload map[string]any) error {
	key := snapshotKey{flightNumber, scheduledDeparture}
	r.hashes[key] = hash
	r.upserted = append(r.upserted, key)
	return nil
}

type fakePollRunRepo struct {
	finishedSeen, finishedChanged, finishedPublished int
	finishedErr                                      error
	finishCalled                                     bool
}

func (r *fakePollRunRepo) Start(ctx context.Context) (uuid.UUID, error) { return uuid.New(), nil }

func (r *fakePollRunRepo) Finish(ctx context.Context, runID uuid.UUID, seen, changed, published int, runErr error) error {
	r.finishCalled = true
	r.finishedSeen, r.finishedChanged, r.finishedPublished, r.finishedErr = seen, changed, published, runErr
	return nil
}

type fakeFlightClient struct {
	result repository.IngestResult
	err    error
	calls  int
}

func (c *fakeFlightClient) Ingest(ctx context.Context, snapshot model.FlightSnapshot, rawPayload map[string]any, source string) (repository.IngestResult, error) {
	c.calls++
	return c.result, c.err
}

type fakePublisher struct {
	err       error
	published []eventbus.FlightEvent
}

func (p *fakePublisher) Publish(ctx context.Context, event eventbus.FlightEvent) error {
	if p.err != nil {
		return p.err
	}
	p.published = append(p.published, event)
	return nil
}

func demoFlight() repository.ProviderFlight {
	return repository.ProviderFlight{
		Snapshot: model.FlightSnapshot{
			FlightNumber:       "VN257",
			AirlineIATA:        "VN",
			OriginIATA:         "HAN",
			DestinationIATA:    "SGN",
			ScheduledDeparture: time.Date(2026, 7, 10, 7, 30, 0, 0, time.UTC),
			ScheduledArrival:   time.Date(2026, 7, 10, 9, 40, 0, 0, time.UTC),
			Status:             model.StatusScheduled,
		},
		RawPayload: map[string]any{"status": "SCHEDULED"},
	}
}

// --- tests --------------------------------------------------------------

func TestRunOnce_NewFlight_IngestsAndPublishesCreated(t *testing.T) {
	provider := &fakeProvider{flights: []repository.ProviderFlight{demoFlight()}}
	snapshots := newFakeSnapshotRepo()
	pollRuns := &fakePollRunRepo{}
	flightClient := &fakeFlightClient{result: repository.IngestResult{PreviousStatus: nil, NewStatus: "SCHEDULED", Changed: true}}
	publisher := &fakePublisher{}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	summary, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if summary.FlightsSeen != 1 || summary.FlightsChanged != 1 || summary.EventsPublished != 1 || len(summary.Errors) != 0 {
		t.Fatalf("unexpected summary: %+v", summary)
	}
	if flightClient.calls != 1 {
		t.Errorf("expected 1 ingest call, got %d", flightClient.calls)
	}
	if len(publisher.published) != 1 || publisher.published[0].EventType != eventbus.EventFlightCreated {
		t.Fatalf("expected one FlightCreated event, got %+v", publisher.published)
	}
	if len(snapshots.upserted) != 1 {
		t.Errorf("expected snapshot to be upserted once, got %d", len(snapshots.upserted))
	}
	if !pollRuns.finishCalled || pollRuns.finishedErr != nil {
		t.Errorf("expected poll run finished with no error, got finishCalled=%v err=%v", pollRuns.finishCalled, pollRuns.finishedErr)
	}
}

func TestRunOnce_UnchangedFlight_OnlyTouchesPolledAt(t *testing.T) {
	flight := demoFlight()
	hash, err := snapshotHash(flight.Snapshot)
	if err != nil {
		t.Fatalf("snapshotHash: %v", err)
	}

	provider := &fakeProvider{flights: []repository.ProviderFlight{flight}}
	snapshots := newFakeSnapshotRepo()
	snapshots.hashes[snapshotKey{flight.Snapshot.FlightNumber, flight.Snapshot.ScheduledDeparture}] = hash
	pollRuns := &fakePollRunRepo{}
	flightClient := &fakeFlightClient{}
	publisher := &fakePublisher{}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	summary, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if summary.FlightsChanged != 0 || summary.EventsPublished != 0 {
		t.Fatalf("expected no changes/events for an unchanged flight, got %+v", summary)
	}
	if flightClient.calls != 0 {
		t.Errorf("expected no ingest call for an unchanged flight, got %d", flightClient.calls)
	}
	if len(snapshots.touched) != 1 {
		t.Errorf("expected TouchPolledAt to be called once, got %d", len(snapshots.touched))
	}
	if len(snapshots.upserted) != 0 {
		t.Errorf("expected no upsert for an unchanged flight, got %d", len(snapshots.upserted))
	}
}

func TestRunOnce_StatusTransition_PublishesSpecificEvent(t *testing.T) {
	flight := demoFlight()
	flight.Snapshot.Status = model.StatusBoarding

	provider := &fakeProvider{flights: []repository.ProviderFlight{flight}}
	snapshots := newFakeSnapshotRepo()
	snapshots.hashes[snapshotKey{flight.Snapshot.FlightNumber, flight.Snapshot.ScheduledDeparture}] = "stale-hash"
	pollRuns := &fakePollRunRepo{}
	previous := "SCHEDULED"
	flightClient := &fakeFlightClient{result: repository.IngestResult{PreviousStatus: &previous, NewStatus: "BOARDING", Changed: true}}
	publisher := &fakePublisher{}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	summary, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(publisher.published) != 1 || publisher.published[0].EventType != eventbus.EventFlightBoarding {
		t.Fatalf("expected one FlightBoarding event, got %+v", publisher.published)
	}
	if summary.EventsPublished != 1 {
		t.Errorf("expected 1 event published, got %d", summary.EventsPublished)
	}
}

func TestRunOnce_PublishFails_LeavesHashUnchangedForRetry(t *testing.T) {
	flight := demoFlight()
	provider := &fakeProvider{flights: []repository.ProviderFlight{flight}}
	snapshots := newFakeSnapshotRepo()
	pollRuns := &fakePollRunRepo{}
	flightClient := &fakeFlightClient{result: repository.IngestResult{PreviousStatus: nil, NewStatus: "SCHEDULED", Changed: true}}
	publisher := &fakePublisher{err: errors.New("broker unavailable")}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	summary, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(summary.Errors) != 1 {
		t.Fatalf("expected 1 error recorded, got %+v", summary.Errors)
	}
	if summary.EventsPublished != 0 {
		t.Errorf("expected 0 events published, got %d", summary.EventsPublished)
	}
	if len(snapshots.upserted) != 0 {
		t.Errorf("expected hash NOT to be upserted when publish fails (so the next poll retries), got %d upserts", len(snapshots.upserted))
	}
}

func TestRunOnce_IngestFails_NoEventNoUpsert(t *testing.T) {
	flight := demoFlight()
	provider := &fakeProvider{flights: []repository.ProviderFlight{flight}}
	snapshots := newFakeSnapshotRepo()
	pollRuns := &fakePollRunRepo{}
	flightClient := &fakeFlightClient{err: errors.New("flight-service unreachable")}
	publisher := &fakePublisher{}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	summary, err := svc.RunOnce(context.Background())
	if err != nil {
		t.Fatalf("RunOnce returned error: %v", err)
	}

	if len(summary.Errors) != 1 {
		t.Fatalf("expected 1 error recorded, got %+v", summary.Errors)
	}
	if summary.FlightsChanged != 0 || summary.EventsPublished != 0 {
		t.Errorf("expected no changed/published counters on ingest failure, got %+v", summary)
	}
	if len(publisher.published) != 0 || len(snapshots.upserted) != 0 {
		t.Errorf("expected no publish/upsert on ingest failure")
	}
}

func TestRunOnce_ProviderFetchFails_AbortsRun(t *testing.T) {
	provider := &fakeProvider{err: errors.New("provider down")}
	snapshots := newFakeSnapshotRepo()
	pollRuns := &fakePollRunRepo{}
	flightClient := &fakeFlightClient{}
	publisher := &fakePublisher{}

	svc := NewSyncService(provider, snapshots, pollRuns, flightClient, publisher, "fake-provider", zap.NewNop())
	_, err := svc.RunOnce(context.Background())
	if err == nil {
		t.Fatal("expected RunOnce to return an error when the provider fetch fails")
	}
	if !pollRuns.finishCalled || pollRuns.finishedErr == nil {
		t.Errorf("expected poll run to be finished with an error, got finishCalled=%v err=%v", pollRuns.finishCalled, pollRuns.finishedErr)
	}
	if pollRuns.finishedSeen != 0 || pollRuns.finishedChanged != 0 || pollRuns.finishedPublished != 0 {
		t.Errorf("expected zeroed counters on abort, got seen=%d changed=%d published=%d",
			pollRuns.finishedSeen, pollRuns.finishedChanged, pollRuns.finishedPublished)
	}
}
