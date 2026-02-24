package repository

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// StreamRepository handles database operations for streams.
type StreamRepository struct {
	pool *pgxpool.Pool
}

// NewStreamRepository creates a new StreamRepository.
func NewStreamRepository(pool *pgxpool.Pool) *StreamRepository {
	return &StreamRepository{pool: pool}
}

// Create inserts a new stream and its group associations.
func (r *StreamRepository) Create(ctx context.Context, s *model.Stream, groupIDs []uuid.UUID) error {
	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO streams (id, title, broadcaster_id, stream_key_id, source_type, status, media_path, started_at, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, NOW(), NOW(), NOW())
		RETURNING started_at, created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		s.ID, s.Title, s.BroadcasterID, s.StreamKeyID, s.SourceType, s.Status, s.MediaPath,
	).Scan(&s.StartedAt, &s.CreatedAt, &s.UpdatedAt)
	if err != nil {
		return err
	}

	// Insert group associations
	for _, gid := range groupIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO stream_groups (stream_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
			s.ID, gid,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a stream by ID, joined with broadcaster details and group IDs.
func (r *StreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.StreamWithDetails, error) {
	query := `
		SELECT s.id, s.title, s.broadcaster_id, s.stream_key_id, s.source_type, s.status,
		       s.media_path, s.recording_url, s.started_at, s.ended_at, s.created_at, s.updated_at,
		       u.username, u.display_name
		FROM streams s
		LEFT JOIN users u ON u.id = s.broadcaster_id
		WHERE s.id = $1`

	swd := &model.StreamWithDetails{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&swd.ID, &swd.Title, &swd.BroadcasterID, &swd.StreamKeyID,
		&swd.SourceType, &swd.Status, &swd.MediaPath, &swd.RecordingURL,
		&swd.StartedAt, &swd.EndedAt, &swd.CreatedAt, &swd.UpdatedAt,
		&swd.Username, &swd.DisplayName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("stream")
		}
		return nil, err
	}

	// Load group IDs
	groups, err := r.GetGroups(ctx, id)
	if err != nil {
		return nil, err
	}
	swd.Groups = groups

	return swd, nil
}

// GetByMediaPath retrieves a stream by its MediaMTX path name.
func (r *StreamRepository) GetByMediaPath(ctx context.Context, mediaPath string) (*model.StreamWithDetails, error) {
	query := `
		SELECT s.id, s.title, s.broadcaster_id, s.stream_key_id, s.source_type, s.status,
		       s.media_path, s.recording_url, s.started_at, s.ended_at, s.created_at, s.updated_at,
		       u.username, u.display_name
		FROM streams s
		LEFT JOIN users u ON u.id = s.broadcaster_id
		WHERE s.media_path = $1`

	swd := &model.StreamWithDetails{}
	err := r.pool.QueryRow(ctx, query, mediaPath).Scan(
		&swd.ID, &swd.Title, &swd.BroadcasterID, &swd.StreamKeyID,
		&swd.SourceType, &swd.Status, &swd.MediaPath, &swd.RecordingURL,
		&swd.StartedAt, &swd.EndedAt, &swd.CreatedAt, &swd.UpdatedAt,
		&swd.Username, &swd.DisplayName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("stream")
		}
		return nil, err
	}

	groups, err := r.GetGroups(ctx, swd.ID)
	if err != nil {
		return nil, err
	}
	swd.Groups = groups

	return swd, nil
}

// ListByGroupIDs returns streams visible to a user (via group membership).
// Optionally filter by status (empty string = all).
func (r *StreamRepository) ListByGroupIDs(ctx context.Context, groupIDs []uuid.UUID, status string) ([]model.StreamWithDetails, error) {
	query := `
		SELECT DISTINCT s.id, s.title, s.broadcaster_id, s.stream_key_id, s.source_type, s.status,
		       s.media_path, s.recording_url, s.started_at, s.ended_at, s.created_at, s.updated_at,
		       u.username, u.display_name
		FROM streams s
		LEFT JOIN users u ON u.id = s.broadcaster_id
		INNER JOIN stream_groups sg ON sg.stream_id = s.id
		WHERE sg.group_id = ANY($1)`

	args := []any{groupIDs}

	if status != "" {
		query += ` AND s.status = $2`
		args = append(args, status)
	}

	query += ` ORDER BY s.started_at DESC`

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []model.StreamWithDetails
	for rows.Next() {
		var s model.StreamWithDetails
		if err := rows.Scan(
			&s.ID, &s.Title, &s.BroadcasterID, &s.StreamKeyID,
			&s.SourceType, &s.Status, &s.MediaPath, &s.RecordingURL,
			&s.StartedAt, &s.EndedAt, &s.CreatedAt, &s.UpdatedAt,
			&s.Username, &s.DisplayName,
		); err != nil {
			return nil, err
		}
		streams = append(streams, s)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch-load group IDs for all streams
	streamIDs := make([]uuid.UUID, len(streams))
	for i := range streams {
		streamIDs[i] = streams[i].ID
	}

	groupMap, err := r.getGroupsBatch(ctx, streamIDs)
	if err != nil {
		return nil, err
	}
	for i := range streams {
		streams[i].Groups = groupMap[streams[i].ID]
		if streams[i].Groups == nil {
			streams[i].Groups = []uuid.UUID{}
		}
	}

	return streams, nil
}

// UpdateStatus updates a stream's status and optionally sets ended_at.
func (r *StreamRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status string, endedAt *time.Time) error {
	query := `
		UPDATE streams
		SET status = $2, ended_at = $3, updated_at = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, status, endedAt)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream")
	}
	return nil
}

// SetRecordingURL sets the S3 URL of the completed recording.
func (r *StreamRepository) SetRecordingURL(ctx context.Context, id uuid.UUID, url string) error {
	query := `UPDATE streams SET recording_url = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id, url)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream")
	}
	return nil
}

// AddGroup adds a group association to a stream.
func (r *StreamRepository) AddGroup(ctx context.Context, streamID, groupID uuid.UUID) error {
	_, err := r.pool.Exec(ctx,
		`INSERT INTO stream_groups (stream_id, group_id) VALUES ($1, $2) ON CONFLICT DO NOTHING`,
		streamID, groupID,
	)
	return err
}

// RemoveGroup removes a group association from a stream.
func (r *StreamRepository) RemoveGroup(ctx context.Context, streamID, groupID uuid.UUID) error {
	tag, err := r.pool.Exec(ctx,
		`DELETE FROM stream_groups WHERE stream_id = $1 AND group_id = $2`,
		streamID, groupID,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream group")
	}
	return nil
}

// GetGroups returns the group IDs associated with a stream.
func (r *StreamRepository) GetGroups(ctx context.Context, streamID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT group_id FROM stream_groups WHERE stream_id = $1 ORDER BY shared_at`,
		streamID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var groupIDs []uuid.UUID
	for rows.Next() {
		var gid uuid.UUID
		if err := rows.Scan(&gid); err != nil {
			return nil, err
		}
		groupIDs = append(groupIDs, gid)
	}
	return groupIDs, rows.Err()
}

// getGroupsBatch loads group IDs for multiple streams in a single query.
func (r *StreamRepository) getGroupsBatch(ctx context.Context, streamIDs []uuid.UUID) (map[uuid.UUID][]uuid.UUID, error) {
	if len(streamIDs) == 0 {
		return make(map[uuid.UUID][]uuid.UUID), nil
	}

	rows, err := r.pool.Query(ctx,
		`SELECT stream_id, group_id FROM stream_groups WHERE stream_id = ANY($1) ORDER BY shared_at`,
		streamIDs,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	m := make(map[uuid.UUID][]uuid.UUID)
	for rows.Next() {
		var sid, gid uuid.UUID
		if err := rows.Scan(&sid, &gid); err != nil {
			return nil, err
		}
		m[sid] = append(m[sid], gid)
	}
	return m, rows.Err()
}

// Delete removes a stream by ID.
func (r *StreamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM streams WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream")
	}
	return nil
}

// ---------------------------------------------------------------------------
// Stream Locations
// ---------------------------------------------------------------------------

// CreateLocation inserts a GPS telemetry point for a stream.
func (r *StreamRepository) CreateLocation(ctx context.Context, loc *model.StreamLocation) error {
	if loc.ID == uuid.Nil {
		loc.ID = uuid.New()
	}

	query := `
		INSERT INTO stream_locations (id, stream_id, location, altitude, heading, speed, recorded_at, created_at)
		VALUES ($1, $2, ST_SetSRID(ST_MakePoint($3, $4), 4326), $5, $6, $7, $8, NOW())`

	_, err := r.pool.Exec(ctx, query,
		loc.ID, loc.StreamID,
		loc.Lng, loc.Lat, // ST_MakePoint takes (lng, lat)
		loc.Altitude, loc.Heading, loc.Speed,
		loc.RecordedAt,
	)
	return err
}

// GetLocations returns all GPS telemetry points for a stream, ordered by time.
func (r *StreamRepository) GetLocations(ctx context.Context, streamID uuid.UUID) ([]model.StreamLocation, error) {
	query := `
		SELECT id, stream_id,
		       ST_Y(location) AS lat, ST_X(location) AS lng,
		       altitude, heading, speed, recorded_at
		FROM stream_locations
		WHERE stream_id = $1
		ORDER BY recorded_at ASC`

	rows, err := r.pool.Query(ctx, query, streamID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var locations []model.StreamLocation
	for rows.Next() {
		var loc model.StreamLocation
		if err := rows.Scan(
			&loc.ID, &loc.StreamID,
			&loc.Lat, &loc.Lng,
			&loc.Altitude, &loc.Heading, &loc.Speed,
			&loc.RecordedAt,
		); err != nil {
			return nil, err
		}
		locations = append(locations, loc)
	}
	return locations, rows.Err()
}
