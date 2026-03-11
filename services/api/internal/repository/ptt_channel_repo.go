package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// PTTChannelRepository handles database operations for PTT channels.
type PTTChannelRepository struct {
	pool *pgxpool.Pool
}

// NewPTTChannelRepository creates a new PTTChannelRepository.
func NewPTTChannelRepository(pool *pgxpool.Pool) *PTTChannelRepository {
	return &PTTChannelRepository{pool: pool}
}

// Create inserts a new PTT channel.
func (r *PTTChannelRepository) Create(ctx context.Context, ch *model.PTTChannel) error {
	query := `
		INSERT INTO ptt_channels (id, group_id, room_id, name, is_default)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`

	if ch.ID == uuid.Nil {
		ch.ID = uuid.New()
	}

	err := r.pool.QueryRow(ctx, query,
		ch.ID, ch.GroupID, ch.RoomID, ch.Name, ch.IsDefault,
	).Scan(&ch.CreatedAt)
	if err != nil {
		if isDuplicateKeyError(err) {
			return model.ErrConflict("PTT channel with this name already exists in the group")
		}
		return err
	}
	return nil
}

// GetByID retrieves a PTT channel by its ID.
func (r *PTTChannelRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.PTTChannel, error) {
	query := `
		SELECT id, group_id, room_id, name, is_default, created_at
		FROM ptt_channels WHERE id = $1`

	ch := &model.PTTChannel{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&ch.ID, &ch.GroupID, &ch.RoomID, &ch.Name, &ch.IsDefault, &ch.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("PTT channel")
		}
		return nil, err
	}
	return ch, nil
}

// ListByGroupID retrieves all PTT channels for a group.
func (r *PTTChannelRepository) ListByGroupID(ctx context.Context, groupID uuid.UUID) ([]model.PTTChannel, error) {
	query := `
		SELECT id, group_id, room_id, name, is_default, created_at
		FROM ptt_channels
		WHERE group_id = $1
		ORDER BY is_default DESC, name ASC`

	rows, err := r.pool.Query(ctx, query, groupID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var channels []model.PTTChannel
	for rows.Next() {
		var ch model.PTTChannel
		if err := rows.Scan(
			&ch.ID, &ch.GroupID, &ch.RoomID, &ch.Name, &ch.IsDefault, &ch.CreatedAt,
		); err != nil {
			return nil, err
		}
		channels = append(channels, ch)
	}
	return channels, rows.Err()
}

// GetDefaultByGroupID retrieves the default PTT channel for a group.
func (r *PTTChannelRepository) GetDefaultByGroupID(ctx context.Context, groupID uuid.UUID) (*model.PTTChannel, error) {
	query := `
		SELECT id, group_id, room_id, name, is_default, created_at
		FROM ptt_channels
		WHERE group_id = $1 AND is_default = true`

	ch := &model.PTTChannel{}
	err := r.pool.QueryRow(ctx, query, groupID).Scan(
		&ch.ID, &ch.GroupID, &ch.RoomID, &ch.Name, &ch.IsDefault, &ch.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("PTT channel")
		}
		return nil, err
	}
	return ch, nil
}

// Delete removes a PTT channel.
func (r *PTTChannelRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM ptt_channels WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("PTT channel")
	}
	return nil
}
