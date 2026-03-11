package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// CotRepository handles database operations for CoT events.
type CotRepository struct {
	pool *pgxpool.Pool
}

// NewCotRepository creates a new CotRepository.
func NewCotRepository(pool *pgxpool.Pool) *CotRepository {
	return &CotRepository{pool: pool}
}

// Create inserts a new CoT event into the database.
func (r *CotRepository) Create(ctx context.Context, evt *model.CotEvent) error {
	query := `
		INSERT INTO cot_events (
			id, event_uid, event_type, how, user_id, device_id, callsign,
			location, hae, ce, le, speed, course,
			detail_xml, raw_xml,
			event_time, start_time, stale_time, created_at
		) VALUES (
			$1, $2, $3, $4, $5, $6, $7,
			ST_SetSRID(ST_MakePoint($8, $9), 4326),
			$10, $11, $12, $13, $14,
			$15, $16,
			$17, $18, $19, $20
		)`

	if evt.ID == uuid.Nil {
		evt.ID = uuid.New()
	}

	_, err := r.pool.Exec(ctx, query,
		evt.ID, evt.EventUID, evt.EventType, evt.How,
		evt.UserID, evt.DeviceID, evt.Callsign,
		evt.Lng, evt.Lat, // ST_MakePoint(lng, lat)
		evt.HAE, evt.CE, evt.LE, evt.Speed, evt.Course,
		evt.DetailXML, evt.RawXML,
		evt.EventTime, evt.StartTime, evt.StaleTime, evt.CreatedAt,
	)
	return err
}

// selectBase is the common SELECT for CoT event queries.
const cotSelectBase = `
	SELECT ce.id, ce.event_uid, ce.event_type, ce.how,
		ce.user_id, ce.device_id, ce.callsign,
		ST_Y(ce.location) AS lat, ST_X(ce.location) AS lng,
		ce.hae, ce.ce, ce.le, ce.speed, ce.course,
		ce.detail_xml, ce.raw_xml,
		ce.event_time, ce.start_time, ce.stale_time, ce.created_at
	FROM cot_events ce`

const cotCountBase = `
	SELECT COUNT(*)
	FROM cot_events ce`

// List returns CoT events matching the filters with pagination.
func (r *CotRepository) List(ctx context.Context, f model.CotEventFilters) ([]model.CotEvent, int, error) {
	where := " WHERE 1=1"
	var args []any

	where, args = appendCotFilters(where, args, f)

	// Count
	var total int
	if err := r.pool.QueryRow(ctx, cotCountBase+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count cot events: %w", err)
	}

	// Data
	offset := (f.Page - 1) * f.PageSize
	args = append(args, f.PageSize, offset)
	dataQuery := cotSelectBase + where +
		" ORDER BY ce.event_time DESC" +
		fmt.Sprintf(" LIMIT $%d OFFSET $%d", len(args)-1, len(args))

	rows, err := r.pool.Query(ctx, dataQuery, args...)
	if err != nil {
		return nil, 0, fmt.Errorf("query cot events: %w", err)
	}
	defer rows.Close()

	var events []model.CotEvent
	for rows.Next() {
		var e model.CotEvent
		if err := rows.Scan(
			&e.ID, &e.EventUID, &e.EventType, &e.How,
			&e.UserID, &e.DeviceID, &e.Callsign,
			&e.Lat, &e.Lng,
			&e.HAE, &e.CE, &e.LE, &e.Speed, &e.Course,
			&e.DetailXML, &e.RawXML,
			&e.EventTime, &e.StartTime, &e.StaleTime, &e.CreatedAt,
		); err != nil {
			return nil, 0, fmt.Errorf("scan cot event: %w", err)
		}
		events = append(events, e)
	}
	return events, total, rows.Err()
}

// GetLatestByUID returns the most recent CoT event for a given event UID.
func (r *CotRepository) GetLatestByUID(ctx context.Context, eventUID string) (*model.CotEvent, error) {
	query := cotSelectBase + `
		WHERE ce.event_uid = $1
		ORDER BY ce.event_time DESC
		LIMIT 1`

	var e model.CotEvent
	err := r.pool.QueryRow(ctx, query, eventUID).Scan(
		&e.ID, &e.EventUID, &e.EventType, &e.How,
		&e.UserID, &e.DeviceID, &e.Callsign,
		&e.Lat, &e.Lng,
		&e.HAE, &e.CE, &e.LE, &e.Speed, &e.Course,
		&e.DetailXML, &e.RawXML,
		&e.EventTime, &e.StartTime, &e.StaleTime, &e.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("cot event")
		}
		return nil, err
	}
	return &e, nil
}

// appendCotFilters adds optional WHERE clauses for CoT event queries.
func appendCotFilters(where string, args []any, f model.CotEventFilters) (string, []any) {
	var clauses []string

	if f.EventUID != "" {
		args = append(args, f.EventUID)
		clauses = append(clauses, fmt.Sprintf("ce.event_uid = $%d", len(args)))
	}
	if f.EventType != "" {
		// Prefix match: "a-f" matches "a-f-G-U-C"
		args = append(args, f.EventType+"%")
		clauses = append(clauses, fmt.Sprintf("ce.event_type LIKE $%d", len(args)))
	}
	if f.From != nil {
		args = append(args, *f.From)
		clauses = append(clauses, fmt.Sprintf("ce.event_time >= $%d", len(args)))
	}
	if f.To != nil {
		args = append(args, *f.To)
		clauses = append(clauses, fmt.Sprintf("ce.event_time <= $%d", len(args)))
	}

	if len(clauses) > 0 {
		where += " AND " + strings.Join(clauses, " AND ")
	}
	return where, args
}
