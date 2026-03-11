package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// RecordingRepository handles database operations for recordings.
type RecordingRepository struct {
	pool *pgxpool.Pool
}

// NewRecordingRepository creates a new RecordingRepository.
func NewRecordingRepository(pool *pgxpool.Pool) *RecordingRepository {
	return &RecordingRepository{pool: pool}
}

// Create inserts a new recording.
func (r *RecordingRepository) Create(ctx context.Context, rec *model.Recording) error {
	query := `
		INSERT INTO recordings (id, room_id, stream_id, egress_id, storage_path, file_type, status)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING started_at`

	if rec.ID == uuid.Nil {
		rec.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		rec.ID, rec.RoomID, rec.StreamID, rec.EgressID,
		rec.StoragePath, rec.FileType, rec.Status,
	).Scan(&rec.StartedAt)
}

// GetByID retrieves a recording by its ID.
func (r *RecordingRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Recording, error) {
	query := `
		SELECT id, room_id, stream_id, egress_id, storage_path, file_type,
		       duration_secs, file_size_bytes, status, started_at, ended_at
		FROM recordings WHERE id = $1`

	rec := &model.Recording{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&rec.ID, &rec.RoomID, &rec.StreamID, &rec.EgressID,
		&rec.StoragePath, &rec.FileType, &rec.DurationSecs,
		&rec.FileSizeBytes, &rec.Status, &rec.StartedAt, &rec.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("recording")
		}
		return nil, err
	}
	return rec, nil
}

// GetByEgressID retrieves a recording by its LiveKit egress ID.
func (r *RecordingRepository) GetByEgressID(ctx context.Context, egressID string) (*model.Recording, error) {
	query := `
		SELECT id, room_id, stream_id, egress_id, storage_path, file_type,
		       duration_secs, file_size_bytes, status, started_at, ended_at
		FROM recordings WHERE egress_id = $1`

	rec := &model.Recording{}
	err := r.pool.QueryRow(ctx, query, egressID).Scan(
		&rec.ID, &rec.RoomID, &rec.StreamID, &rec.EgressID,
		&rec.StoragePath, &rec.FileType, &rec.DurationSecs,
		&rec.FileSizeBytes, &rec.Status, &rec.StartedAt, &rec.EndedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("recording")
		}
		return nil, err
	}
	return rec, nil
}

// ListByRoomID retrieves all recordings for a room.
func (r *RecordingRepository) ListByRoomID(ctx context.Context, roomID uuid.UUID) ([]model.Recording, error) {
	query := `
		SELECT id, room_id, stream_id, egress_id, storage_path, file_type,
		       duration_secs, file_size_bytes, status, started_at, ended_at
		FROM recordings
		WHERE room_id = $1
		ORDER BY started_at DESC`

	return r.scanRecordings(ctx, query, roomID)
}

// ListByStreamID retrieves all recordings for a stream.
func (r *RecordingRepository) ListByStreamID(ctx context.Context, streamID uuid.UUID) ([]model.Recording, error) {
	query := `
		SELECT id, room_id, stream_id, egress_id, storage_path, file_type,
		       duration_secs, file_size_bytes, status, started_at, ended_at
		FROM recordings
		WHERE stream_id = $1
		ORDER BY started_at DESC`

	return r.scanRecordings(ctx, query, streamID)
}

// Update modifies an existing recording.
func (r *RecordingRepository) Update(ctx context.Context, rec *model.Recording) error {
	query := `
		UPDATE recordings
		SET storage_path = $2, duration_secs = $3, file_size_bytes = $4,
		    status = $5, ended_at = $6
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query,
		rec.ID, rec.StoragePath, rec.DurationSecs,
		rec.FileSizeBytes, rec.Status, rec.EndedAt,
	)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("recording")
	}
	return nil
}

func (r *RecordingRepository) scanRecordings(ctx context.Context, query string, arg any) ([]model.Recording, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var recordings []model.Recording
	for rows.Next() {
		var rec model.Recording
		if err := rows.Scan(
			&rec.ID, &rec.RoomID, &rec.StreamID, &rec.EgressID,
			&rec.StoragePath, &rec.FileType, &rec.DurationSecs,
			&rec.FileSizeBytes, &rec.Status, &rec.StartedAt, &rec.EndedAt,
		); err != nil {
			return nil, err
		}
		recordings = append(recordings, rec)
	}
	return recordings, rows.Err()
}
