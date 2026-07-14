package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// flightSnapshotRow maps to sync.flight_snapshots
// (docs/database/sync-service.sql). It is private to this package — callers
// depend on the SnapshotRepository interface, not this GORM model
// (repository pattern, docs/architecture.md section 4).
type flightSnapshotRow struct {
	ID                 uuid.UUID `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	FlightNumber       string    `gorm:"column:flight_number"`
	ScheduledDeparture time.Time `gorm:"column:scheduled_departure"`
	RawHash            string    `gorm:"column:raw_hash"`
	RawPayload         []byte    `gorm:"column:raw_payload"`
	LastPolledAt       time.Time `gorm:"column:last_polled_at"`
	LastChangedAt      time.Time `gorm:"column:last_changed_at"`
	CreatedAt          time.Time `gorm:"column:created_at"`
}

func (flightSnapshotRow) TableName() string { return "sync.flight_snapshots" }

// SnapshotRepository is sync-service's private change-detection cache
// port. It is NOT a copy of flight-service's canonical data — see the
// schema comment in docs/database/sync-service.sql.
type SnapshotRepository interface {
	// GetHash returns the last stored raw_hash for (flightNumber,
	// scheduledDeparture), and whether a row exists at all.
	GetHash(ctx context.Context, flightNumber string, scheduledDeparture time.Time) (hash string, found bool, err error)
	// TouchPolledAt bumps last_polled_at without changing the stored
	// hash/payload, for a flight observed unchanged this poll.
	TouchPolledAt(ctx context.Context, flightNumber string, scheduledDeparture time.Time) error
	// Upsert stores/updates the snapshot's hash and raw payload after a
	// change has been successfully ingested (and, if applicable,
	// published).
	Upsert(ctx context.Context, flightNumber string, scheduledDeparture time.Time, hash string, rawPayload map[string]any) error
}

type gormSnapshotRepository struct {
	db *gorm.DB
}

func NewSnapshotRepository(db *gorm.DB) SnapshotRepository {
	return &gormSnapshotRepository{db: db}
}

func (r *gormSnapshotRepository) GetHash(ctx context.Context, flightNumber string, scheduledDeparture time.Time) (string, bool, error) {
	var row flightSnapshotRow
	err := r.db.WithContext(ctx).
		Where("flight_number = ? AND scheduled_departure = ?", flightNumber, scheduledDeparture).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return "", false, nil
	}
	if err != nil {
		return "", false, err
	}
	return row.RawHash, true, nil
}

func (r *gormSnapshotRepository) TouchPolledAt(ctx context.Context, flightNumber string, scheduledDeparture time.Time) error {
	return r.db.WithContext(ctx).
		Model(&flightSnapshotRow{}).
		Where("flight_number = ? AND scheduled_departure = ?", flightNumber, scheduledDeparture).
		Update("last_polled_at", time.Now().UTC()).Error
}

func (r *gormSnapshotRepository) Upsert(ctx context.Context, flightNumber string, scheduledDeparture time.Time, hash string, rawPayload map[string]any) error {
	payload, err := json.Marshal(rawPayload)
	if err != nil {
		return err
	}

	// row's ID is deliberately left zero: FirstOrCreate treats a non-zero
	// primary key on the destination struct as an extra search condition
	// (id = <value>), which would never match an existing row and cause
	// this to always attempt an INSERT — colliding with the unique
	// (flight_number, scheduled_departure) constraint instead of updating
	// it. Leaving it zero lets Postgres's column default assign one only
	// when a row is actually created.
	now := time.Now().UTC()
	row := flightSnapshotRow{
		FlightNumber:       flightNumber,
		ScheduledDeparture: scheduledDeparture,
		RawHash:            hash,
		RawPayload:         payload,
		LastPolledAt:       now,
		LastChangedAt:      now,
	}

	return r.db.WithContext(ctx).
		Where("flight_number = ? AND scheduled_departure = ?", flightNumber, scheduledDeparture).
		Assign(map[string]any{
			"raw_hash":        hash,
			"raw_payload":     payload,
			"last_polled_at":  now,
			"last_changed_at": now,
		}).
		FirstOrCreate(&row).Error
}

// pollRunRow maps to sync.poll_runs (docs/database/sync-service.sql).
type pollRunRow struct {
	ID              uuid.UUID  `gorm:"column:id;primaryKey"`
	StartedAt       time.Time  `gorm:"column:started_at"`
	FinishedAt      *time.Time `gorm:"column:finished_at"`
	FlightsSeen     int        `gorm:"column:flights_seen"`
	FlightsChanged  int        `gorm:"column:flights_changed"`
	EventsPublished int        `gorm:"column:events_published"`
	Error           *string    `gorm:"column:error"`
}

func (pollRunRow) TableName() string { return "sync.poll_runs" }

// PollRunRepository records one row per CronJob run for observability
// (docs/deployment/deployment-guide.md's "did the 07:10 poll run" audit
// trail).
type PollRunRepository interface {
	Start(ctx context.Context) (uuid.UUID, error)
	Finish(ctx context.Context, runID uuid.UUID, flightsSeen, flightsChanged, eventsPublished int, runErr error) error
}

type gormPollRunRepository struct {
	db *gorm.DB
}

func NewPollRunRepository(db *gorm.DB) PollRunRepository {
	return &gormPollRunRepository{db: db}
}

func (r *gormPollRunRepository) Start(ctx context.Context) (uuid.UUID, error) {
	row := pollRunRow{ID: uuid.New(), StartedAt: time.Now().UTC()}
	if err := r.db.WithContext(ctx).Create(&row).Error; err != nil {
		return uuid.Nil, err
	}
	return row.ID, nil
}

func (r *gormPollRunRepository) Finish(ctx context.Context, runID uuid.UUID, flightsSeen, flightsChanged, eventsPublished int, runErr error) error {
	finishedAt := time.Now().UTC()
	updates := map[string]any{
		"finished_at":      finishedAt,
		"flights_seen":     flightsSeen,
		"flights_changed":  flightsChanged,
		"events_published": eventsPublished,
	}
	if runErr != nil {
		msg := runErr.Error()
		updates["error"] = msg
	}
	return r.db.WithContext(ctx).Model(&pollRunRow{}).Where("id = ?", runID).Updates(updates).Error
}
