package repository

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"

	sharederrors "github.com/flighttracker/pkg/errors"

	"github.com/flighttracker/services/flight-service/internal/model"
)

// flightRow maps to flight.flights (docs/database/flight-service.sql). It
// is private to this package — callers depend on the FlightRepository
// interface, not this GORM model (repository pattern,
// docs/architecture.md section 4).
type flightRow struct {
	ID                 uuid.UUID `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	FlightNumber       string    `gorm:"column:flight_number"`
	AirlineIATA        string    `gorm:"column:airline_iata"`
	AirlineName        *string   `gorm:"column:airline_name"`
	OriginIATA         string    `gorm:"column:origin_iata"`
	DestinationIATA    string    `gorm:"column:destination_iata"`
	ScheduledDeparture time.Time `gorm:"column:scheduled_departure"`
	EstimatedDeparture *time.Time `gorm:"column:estimated_departure"`
	ActualDeparture    *time.Time `gorm:"column:actual_departure"`
	ScheduledArrival   time.Time  `gorm:"column:scheduled_arrival"`
	EstimatedArrival   *time.Time `gorm:"column:estimated_arrival"`
	ActualArrival      *time.Time `gorm:"column:actual_arrival"`
	Gate               *string    `gorm:"column:gate"`
	Terminal           *string    `gorm:"column:terminal"`
	Status             string     `gorm:"column:status;type:flight.flight_status"`
	AircraftType       *string    `gorm:"column:aircraft_type"`
	LastSyncedAt       time.Time  `gorm:"column:last_synced_at"`
	CreatedAt          time.Time  `gorm:"column:created_at"`
	UpdatedAt          time.Time  `gorm:"column:updated_at"`
}

func (flightRow) TableName() string { return "flight.flights" }

func (r flightRow) toModel() model.Flight {
	return model.Flight{
		ID:                 r.ID,
		FlightNumber:       r.FlightNumber,
		AirlineIATA:        r.AirlineIATA,
		AirlineName:        r.AirlineName,
		OriginIATA:         r.OriginIATA,
		DestinationIATA:    r.DestinationIATA,
		ScheduledDeparture: r.ScheduledDeparture,
		EstimatedDeparture: r.EstimatedDeparture,
		ActualDeparture:    r.ActualDeparture,
		ScheduledArrival:   r.ScheduledArrival,
		EstimatedArrival:   r.EstimatedArrival,
		ActualArrival:      r.ActualArrival,
		Gate:               r.Gate,
		Terminal:           r.Terminal,
		Status:             model.FlightStatus(r.Status),
		AircraftType:       r.AircraftType,
		LastSyncedAt:       r.LastSyncedAt,
	}
}

func fromModel(f model.Flight) flightRow {
	return flightRow{
		ID:                 f.ID,
		FlightNumber:       f.FlightNumber,
		AirlineIATA:        f.AirlineIATA,
		AirlineName:        f.AirlineName,
		OriginIATA:         f.OriginIATA,
		DestinationIATA:    f.DestinationIATA,
		ScheduledDeparture: f.ScheduledDeparture,
		EstimatedDeparture: f.EstimatedDeparture,
		ActualDeparture:    f.ActualDeparture,
		ScheduledArrival:   f.ScheduledArrival,
		EstimatedArrival:   f.EstimatedArrival,
		ActualArrival:      f.ActualArrival,
		Gate:               f.Gate,
		Terminal:           f.Terminal,
		Status:             string(f.Status),
		AircraftType:       f.AircraftType,
	}
}

// flightStatusHistoryRow maps to flight.flight_status_history.
type flightStatusHistoryRow struct {
	ID         uuid.UUID `gorm:"column:id;primaryKey;default:gen_random_uuid()"`
	FlightID   uuid.UUID `gorm:"column:flight_id"`
	OldStatus  *string   `gorm:"column:old_status;type:flight.flight_status"`
	NewStatus  string    `gorm:"column:new_status;type:flight.flight_status"`
	ChangedAt  time.Time `gorm:"column:changed_at"`
	Source     string    `gorm:"column:source"`
	RawPayload []byte    `gorm:"column:raw_payload"`
}

func (flightStatusHistoryRow) TableName() string { return "flight.flight_status_history" }

// IngestResult is the outcome of ingesting one snapshot.
type IngestResult struct {
	Flight         model.Flight
	PreviousStatus *string
	Changed        bool
}

// FlightRepository is flight-service's Postgres port
// (docs/database/flight-service.sql).
type FlightRepository interface {
	GetByFlightNumberAndDate(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error)
	SearchByRoute(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error)
	// Ingest upserts snapshot (the only write path into this schema,
	// docs/architecture.md section 2.3) and returns the previous status
	// (nil on first observation) plus whether anything differed, so the
	// caller (sync-service, via the ingest HTTP endpoint) can decide
	// whether to publish a domain event.
	Ingest(ctx context.Context, snapshot model.Flight, source string, rawPayload map[string]any) (IngestResult, error)
}

type gormFlightRepository struct {
	db *gorm.DB
}

func NewFlightRepository(db *gorm.DB) FlightRepository {
	return &gormFlightRepository{db: db}
}

func dayRange(date time.Time) (start, end time.Time) {
	start = time.Date(date.Year(), date.Month(), date.Day(), 0, 0, 0, 0, time.UTC)
	return start, start.Add(24 * time.Hour)
}

func (r *gormFlightRepository) GetByFlightNumberAndDate(ctx context.Context, flightNumber string, date time.Time) (model.Flight, error) {
	start, end := dayRange(date)

	var row flightRow
	err := r.db.WithContext(ctx).
		Where("flight_number = ? AND scheduled_departure >= ? AND scheduled_departure < ?", flightNumber, start, end).
		First(&row).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return model.Flight{}, sharederrors.NotFound("FLIGHT_NOT_FOUND", "no flight found for that number/date")
	}
	if err != nil {
		return model.Flight{}, err
	}
	return row.toModel(), nil
}

func (r *gormFlightRepository) SearchByRoute(ctx context.Context, origin, destination string, date time.Time, limit int) ([]model.Flight, error) {
	start, end := dayRange(date)

	var rows []flightRow
	err := r.db.WithContext(ctx).
		Where("origin_iata = ? AND destination_iata = ? AND scheduled_departure >= ? AND scheduled_departure < ?", origin, destination, start, end).
		Order("scheduled_departure ASC").
		Limit(limit).
		Find(&rows).Error
	if err != nil {
		return nil, err
	}

	flights := make([]model.Flight, 0, len(rows))
	for _, row := range rows {
		flights = append(flights, row.toModel())
	}
	return flights, nil
}

func (r *gormFlightRepository) Ingest(ctx context.Context, snapshot model.Flight, source string, rawPayload map[string]any) (IngestResult, error) {
	payload, err := json.Marshal(rawPayload)
	if err != nil {
		return IngestResult{}, err
	}

	var result IngestResult
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var existing flightRow
		err := tx.Where("flight_number = ? AND scheduled_departure = ?", snapshot.FlightNumber, snapshot.ScheduledDeparture).
			First(&existing).Error

		now := time.Now().UTC()
		incoming := fromModel(snapshot)
		incoming.LastSyncedAt = now

		switch {
		case errors.Is(err, gorm.ErrRecordNotFound):
			incoming.CreatedAt, incoming.UpdatedAt = now, now
			if err := tx.Create(&incoming).Error; err != nil {
				return err
			}
			result = IngestResult{Flight: incoming.toModel(), PreviousStatus: nil, Changed: true}
			return tx.Create(&flightStatusHistoryRow{
				FlightID:   incoming.ID,
				OldStatus:  nil,
				NewStatus:  incoming.Status,
				ChangedAt:  now,
				Source:     source,
				RawPayload: payload,
			}).Error

		case err != nil:
			return err

		default:
			changed := flightsDiffer(existing, incoming)
			statusChanged := existing.Status != incoming.Status
			previous := existing.Status

			incoming.ID = existing.ID
			incoming.CreatedAt = existing.CreatedAt
			incoming.UpdatedAt = now
			if err := tx.Save(&incoming).Error; err != nil {
				return err
			}

			result = IngestResult{Flight: incoming.toModel(), Changed: changed}
			if changed {
				result.PreviousStatus = &previous
			}

			if statusChanged {
				return tx.Create(&flightStatusHistoryRow{
					FlightID:   incoming.ID,
					OldStatus:  &previous,
					NewStatus:  incoming.Status,
					ChangedAt:  now,
					Source:     source,
					RawPayload: payload,
				}).Error
			}
			return nil
		}
	})
	if err != nil {
		return IngestResult{}, err
	}
	return result, nil
}

// flightsDiffer reports whether any field an ingest can legitimately
// update has changed, driving the IngestResponse.changed flag
// (docs/api-contracts/flight-service.yaml).
func flightsDiffer(existing, incoming flightRow) bool {
	if existing.Status != incoming.Status {
		return true
	}
	if !timePtrEqual(existing.EstimatedDeparture, incoming.EstimatedDeparture) ||
		!timePtrEqual(existing.ActualDeparture, incoming.ActualDeparture) ||
		!timePtrEqual(existing.EstimatedArrival, incoming.EstimatedArrival) ||
		!timePtrEqual(existing.ActualArrival, incoming.ActualArrival) {
		return true
	}
	if !strPtrEqual(existing.Gate, incoming.Gate) || !strPtrEqual(existing.Terminal, incoming.Terminal) {
		return true
	}
	if !existing.ScheduledArrival.Equal(incoming.ScheduledArrival) {
		return true
	}
	return false
}

func timePtrEqual(a, b *time.Time) bool {
	if a == nil || b == nil {
		return a == b
	}
	return a.Equal(*b)
}

func strPtrEqual(a, b *string) bool {
	if a == nil || b == nil {
		return a == b
	}
	return *a == *b
}
