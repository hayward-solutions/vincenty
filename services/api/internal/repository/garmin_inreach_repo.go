package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// GarminInReachRepository handles database operations for Garmin InReach feeds.
type GarminInReachRepository struct {
	pool *pgxpool.Pool
}

// NewGarminInReachRepository creates a new GarminInReachRepository.
func NewGarminInReachRepository(pool *pgxpool.Pool) *GarminInReachRepository {
	return &GarminInReachRepository{pool: pool}
}

const garminSelectBase = `
	SELECT id, user_id, device_id, mapshare_id, feed_password,
		EXTRACT(EPOCH FROM poll_interval)::bigint,
		enabled, last_polled_at, last_point_at,
		error_count, last_error,
		created_at, updated_at
	FROM garmin_inreach_feeds`

func scanFeed(row pgx.Row) (*model.GarminInReachFeed, error) {
	var f model.GarminInReachFeed
	var pollSec int64
	if err := row.Scan(
		&f.ID, &f.UserID, &f.DeviceID, &f.MapShareID, &f.FeedPassword,
		&pollSec,
		&f.Enabled, &f.LastPolledAt, &f.LastPointAt,
		&f.ErrorCount, &f.LastError,
		&f.CreatedAt, &f.UpdatedAt,
	); err != nil {
		return nil, err
	}
	f.PollInterval = time.Duration(pollSec) * time.Second
	return &f, nil
}

// Create inserts a new Garmin InReach feed.
func (r *GarminInReachRepository) Create(ctx context.Context, f *model.GarminInReachFeed) error {
	if f.ID == uuid.Nil {
		f.ID = uuid.New()
	}
	now := time.Now()
	f.CreatedAt = now
	f.UpdatedAt = now

	query := `
		INSERT INTO garmin_inreach_feeds (
			id, user_id, device_id, mapshare_id, feed_password,
			poll_interval, enabled, created_at, updated_at
		) VALUES ($1, $2, $3, $4, $5, make_interval(secs => $6), $7, $8, $9)`

	_, err := r.pool.Exec(ctx, query,
		f.ID, f.UserID, f.DeviceID, f.MapShareID, f.FeedPassword,
		int64(f.PollInterval.Seconds()), f.Enabled,
		f.CreatedAt, f.UpdatedAt,
	)
	return err
}

// GetByID returns a feed by its ID.
func (r *GarminInReachRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.GarminInReachFeed, error) {
	f, err := scanFeed(r.pool.QueryRow(ctx, garminSelectBase+" WHERE id = $1", id))
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("garmin inreach feed")
		}
		return nil, err
	}
	return f, nil
}

// List returns all configured feeds.
func (r *GarminInReachRepository) List(ctx context.Context) ([]model.GarminInReachFeed, error) {
	rows, err := r.pool.Query(ctx, garminSelectBase+" ORDER BY created_at DESC")
	if err != nil {
		return nil, fmt.Errorf("list garmin feeds: %w", err)
	}
	defer rows.Close()

	var feeds []model.GarminInReachFeed
	for rows.Next() {
		var f model.GarminInReachFeed
		var pollSec int64
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.DeviceID, &f.MapShareID, &f.FeedPassword,
			&pollSec,
			&f.Enabled, &f.LastPolledAt, &f.LastPointAt,
			&f.ErrorCount, &f.LastError,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan garmin feed: %w", err)
		}
		f.PollInterval = time.Duration(pollSec) * time.Second
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

// ListEnabled returns feeds that are enabled and due for polling.
func (r *GarminInReachRepository) ListEnabled(ctx context.Context) ([]model.GarminInReachFeed, error) {
	query := garminSelectBase + `
		WHERE enabled = true
		AND (last_polled_at IS NULL
			OR last_polled_at + poll_interval <= now())
		ORDER BY last_polled_at ASC NULLS FIRST`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("list enabled garmin feeds: %w", err)
	}
	defer rows.Close()

	var feeds []model.GarminInReachFeed
	for rows.Next() {
		var f model.GarminInReachFeed
		var pollSec int64
		if err := rows.Scan(
			&f.ID, &f.UserID, &f.DeviceID, &f.MapShareID, &f.FeedPassword,
			&pollSec,
			&f.Enabled, &f.LastPolledAt, &f.LastPointAt,
			&f.ErrorCount, &f.LastError,
			&f.CreatedAt, &f.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan garmin feed: %w", err)
		}
		f.PollInterval = time.Duration(pollSec) * time.Second
		feeds = append(feeds, f)
	}
	return feeds, rows.Err()
}

// Update updates a feed's configurable fields.
func (r *GarminInReachRepository) Update(ctx context.Context, f *model.GarminInReachFeed) error {
	f.UpdatedAt = time.Now()
	query := `
		UPDATE garmin_inreach_feeds
		SET feed_password = $2,
			poll_interval = make_interval(secs => $3),
			enabled = $4,
			updated_at = $5
		WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query,
		f.ID, f.FeedPassword,
		int64(f.PollInterval.Seconds()), f.Enabled,
		f.UpdatedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("garmin inreach feed")
	}
	return nil
}

// UpdatePollStatus records the result of a poll attempt.
func (r *GarminInReachRepository) UpdatePollStatus(ctx context.Context, id uuid.UUID, lastPointAt *time.Time, pollErr error) error {
	now := time.Now()
	var errStr *string
	var errCountExpr string

	if pollErr != nil {
		s := pollErr.Error()
		errStr = &s
		errCountExpr = "error_count + 1"
	} else {
		errCountExpr = "0"
	}

	query := fmt.Sprintf(`
		UPDATE garmin_inreach_feeds
		SET last_polled_at = $2,
			last_point_at = COALESCE($3, last_point_at),
			error_count = %s,
			last_error = $4,
			updated_at = $2
		WHERE id = $1`, errCountExpr)

	_, err := r.pool.Exec(ctx, query, id, now, lastPointAt, errStr)
	return err
}

// Delete removes a feed.
func (r *GarminInReachRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, "DELETE FROM garmin_inreach_feeds WHERE id = $1", id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("garmin inreach feed")
	}
	return nil
}
