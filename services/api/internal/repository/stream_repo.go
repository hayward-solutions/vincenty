package repository

import (
	"context"
	"errors"

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

// Create inserts a new stream.
func (r *StreamRepository) Create(ctx context.Context, s *model.Stream) error {
	query := `
		INSERT INTO streams (id, name, source_type, source_url, group_id, created_by,
		                     livekit_ingress_id, livekit_room, stream_key, is_active, metadata)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if s.ID == uuid.Nil {
		s.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		s.ID, s.Name, s.SourceType, s.SourceURL,
		s.GroupID, s.CreatedBy, s.LiveKitIngressID,
		s.LiveKitRoom, s.StreamKey, s.IsActive, s.Metadata,
	).Scan(&s.CreatedAt, &s.UpdatedAt)
}

// GetByID retrieves a stream by its ID.
func (r *StreamRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.Stream, error) {
	query := `
		SELECT id, name, source_type, source_url, group_id, created_by,
		       livekit_ingress_id, livekit_room, stream_key, is_active, metadata,
		       created_at, updated_at
		FROM streams WHERE id = $1`

	s := &model.Stream{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&s.ID, &s.Name, &s.SourceType, &s.SourceURL,
		&s.GroupID, &s.CreatedBy, &s.LiveKitIngressID,
		&s.LiveKitRoom, &s.StreamKey, &s.IsActive, &s.Metadata,
		&s.CreatedAt, &s.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("stream")
		}
		return nil, err
	}
	return s, nil
}

// ListByGroupID retrieves all streams for a group.
func (r *StreamRepository) ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]model.Stream, error) {
	query := `
		SELECT id, name, source_type, source_url, group_id, created_by,
		       livekit_ingress_id, livekit_room, stream_key, is_active, metadata,
		       created_at, updated_at
		FROM streams
		WHERE group_id = $1
		ORDER BY created_at DESC`

	return r.scanStreams(ctx, query, groupID)
}

// ListActiveByUserGroups retrieves all active streams across groups the user belongs to.
func (r *StreamRepository) ListActiveByUserGroups(ctx context.Context, userID uuid.UUID) ([]model.Stream, error) {
	query := `
		SELECT s.id, s.name, s.source_type, s.source_url, s.group_id, s.created_by,
		       s.livekit_ingress_id, s.livekit_room, s.stream_key, s.is_active, s.metadata,
		       s.created_at, s.updated_at
		FROM streams s
		INNER JOIN group_members gm ON gm.group_id = s.group_id
		WHERE gm.user_id = $1 AND s.is_active = true
		ORDER BY s.created_at DESC`

	return r.scanStreams(ctx, query, userID)
}

// Update modifies an existing stream.
func (r *StreamRepository) Update(ctx context.Context, s *model.Stream) error {
	query := `
		UPDATE streams
		SET name = $2, source_url = $3, livekit_ingress_id = $4,
		    livekit_room = $5, stream_key = $6, is_active = $7,
		    metadata = $8, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		s.ID, s.Name, s.SourceURL, s.LiveKitIngressID,
		s.LiveKitRoom, s.StreamKey, s.IsActive, s.Metadata,
	).Scan(&s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("stream")
		}
		return err
	}
	return nil
}

// Delete removes a stream.
func (r *StreamRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM streams WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream")
	}
	return nil
}

// SetActive updates the active status of a stream.
func (r *StreamRepository) SetActive(ctx context.Context, id uuid.UUID, active bool) error {
	query := `UPDATE streams SET is_active = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id, active)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream")
	}
	return nil
}

func (r *StreamRepository) scanStreams(ctx context.Context, query string, arg any) ([]model.Stream, error) {
	rows, err := r.pool.Query(ctx, query, arg)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var streams []model.Stream
	for rows.Next() {
		var s model.Stream
		if err := rows.Scan(
			&s.ID, &s.Name, &s.SourceType, &s.SourceURL,
			&s.GroupID, &s.CreatedBy, &s.LiveKitIngressID,
			&s.LiveKitRoom, &s.StreamKey, &s.IsActive, &s.Metadata,
			&s.CreatedAt, &s.UpdatedAt,
		); err != nil {
			return nil, err
		}
		streams = append(streams, s)
	}
	return streams, rows.Err()
}
