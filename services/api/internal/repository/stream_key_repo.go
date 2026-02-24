package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// StreamKeyRepository handles database operations for stream keys.
type StreamKeyRepository struct {
	pool *pgxpool.Pool
}

// NewStreamKeyRepository creates a new StreamKeyRepository.
func NewStreamKeyRepository(pool *pgxpool.Pool) *StreamKeyRepository {
	return &StreamKeyRepository{pool: pool}
}

// Create inserts a new stream key and its default group associations.
func (r *StreamKeyRepository) Create(ctx context.Context, k *model.StreamKey, groupIDs []uuid.UUID) error {
	if k.ID == uuid.Nil {
		k.ID = uuid.New()
	}

	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	query := `
		INSERT INTO stream_keys (id, label, key_hash, created_by, is_active, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, NOW(), NOW())
		RETURNING created_at, updated_at`

	err = tx.QueryRow(ctx, query,
		k.ID, k.Label, k.KeyHash, k.CreatedBy, k.IsActive,
	).Scan(&k.CreatedAt, &k.UpdatedAt)
	if err != nil {
		return err
	}

	for _, gid := range groupIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO stream_key_groups (stream_key_id, group_id) VALUES ($1, $2)`,
			k.ID, gid,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// GetByID retrieves a stream key by ID, with its group IDs.
func (r *StreamKeyRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.StreamKeyWithGroups, error) {
	query := `
		SELECT id, label, key_hash, created_by, is_active, created_at, updated_at
		FROM stream_keys
		WHERE id = $1`

	k := &model.StreamKeyWithGroups{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&k.ID, &k.Label, &k.KeyHash, &k.CreatedBy, &k.IsActive,
		&k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("stream key")
		}
		return nil, err
	}

	groups, err := r.getGroups(ctx, k.ID)
	if err != nil {
		return nil, err
	}
	k.GroupIDs = groups

	return k, nil
}

// GetByKeyHash retrieves an active stream key by its SHA-256 hash.
func (r *StreamKeyRepository) GetByKeyHash(ctx context.Context, keyHash string) (*model.StreamKeyWithGroups, error) {
	query := `
		SELECT id, label, key_hash, created_by, is_active, created_at, updated_at
		FROM stream_keys
		WHERE key_hash = $1 AND is_active = true`

	k := &model.StreamKeyWithGroups{}
	err := r.pool.QueryRow(ctx, query, keyHash).Scan(
		&k.ID, &k.Label, &k.KeyHash, &k.CreatedBy, &k.IsActive,
		&k.CreatedAt, &k.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("stream key")
		}
		return nil, err
	}

	groups, err := r.getGroups(ctx, k.ID)
	if err != nil {
		return nil, err
	}
	k.GroupIDs = groups

	return k, nil
}

// List returns all stream keys, newest first.
func (r *StreamKeyRepository) List(ctx context.Context) ([]model.StreamKeyWithGroups, error) {
	query := `
		SELECT id, label, key_hash, created_by, is_active, created_at, updated_at
		FROM stream_keys
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var keys []model.StreamKeyWithGroups
	for rows.Next() {
		var k model.StreamKeyWithGroups
		if err := rows.Scan(
			&k.ID, &k.Label, &k.KeyHash, &k.CreatedBy, &k.IsActive,
			&k.CreatedAt, &k.UpdatedAt,
		); err != nil {
			return nil, err
		}
		keys = append(keys, k)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	// Batch-load group IDs
	for i := range keys {
		groups, err := r.getGroups(ctx, keys[i].ID)
		if err != nil {
			return nil, err
		}
		keys[i].GroupIDs = groups
	}

	return keys, nil
}

// Update modifies a stream key's label and active status.
func (r *StreamKeyRepository) Update(ctx context.Context, id uuid.UUID, label string, isActive bool) error {
	query := `
		UPDATE stream_keys
		SET label = $2, is_active = $3, updated_at = NOW()
		WHERE id = $1`

	tag, err := r.pool.Exec(ctx, query, id, label, isActive)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream key")
	}
	return nil
}

// SetGroups replaces a stream key's group associations.
func (r *StreamKeyRepository) SetGroups(ctx context.Context, keyID uuid.UUID, groupIDs []uuid.UUID) error {
	tx, err := r.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `DELETE FROM stream_key_groups WHERE stream_key_id = $1`, keyID)
	if err != nil {
		return err
	}

	for _, gid := range groupIDs {
		_, err = tx.Exec(ctx,
			`INSERT INTO stream_key_groups (stream_key_id, group_id) VALUES ($1, $2)`,
			keyID, gid,
		)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

// Delete removes a stream key by ID.
func (r *StreamKeyRepository) Delete(ctx context.Context, id uuid.UUID) error {
	tag, err := r.pool.Exec(ctx, `DELETE FROM stream_keys WHERE id = $1`, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("stream key")
	}
	return nil
}

// getGroups returns the group IDs associated with a stream key.
func (r *StreamKeyRepository) getGroups(ctx context.Context, keyID uuid.UUID) ([]uuid.UUID, error) {
	rows, err := r.pool.Query(ctx,
		`SELECT group_id FROM stream_key_groups WHERE stream_key_id = $1`,
		keyID,
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
