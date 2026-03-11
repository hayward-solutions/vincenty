package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// DrawingRepository handles database operations for drawings.
type DrawingRepository struct {
	pool *pgxpool.Pool
}

// NewDrawingRepository creates a new DrawingRepository.
func NewDrawingRepository(pool *pgxpool.Pool) *DrawingRepository {
	return &DrawingRepository{pool: pool}
}

// Create inserts a new drawing.
func (r *DrawingRepository) Create(ctx context.Context, d *model.Drawing) error {
	if d.ID == uuid.Nil {
		d.ID = uuid.New()
	}

	query := `
		INSERT INTO drawings (id, owner_id, name, geojson, created_at, updated_at)
		VALUES ($1, $2, $3, $4, NOW(), NOW())
		RETURNING created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		d.ID, d.OwnerID, d.Name, d.GeoJSON,
	).Scan(&d.CreatedAt, &d.UpdatedAt)
}

// GetByID retrieves a drawing by ID, joined with owner details.
func (r *DrawingRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
	query := `
		SELECT d.id, d.owner_id, d.name, d.geojson, d.created_at, d.updated_at,
		       u.username, u.display_name
		FROM drawings d
		INNER JOIN users u ON u.id = d.owner_id
		WHERE d.id = $1`

	dwu := &model.DrawingWithUser{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&dwu.ID, &dwu.OwnerID, &dwu.Name, &dwu.GeoJSON,
		&dwu.CreatedAt, &dwu.UpdatedAt,
		&dwu.Username, &dwu.DisplayName,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("drawing")
		}
		return nil, err
	}

	return dwu, nil
}

// ListByOwner returns all drawings owned by a user, newest first.
func (r *DrawingRepository) ListByOwner(ctx context.Context, ownerID uuid.UUID) ([]model.DrawingWithUser, error) {
	query := `
		SELECT d.id, d.owner_id, d.name, d.geojson, d.created_at, d.updated_at,
		       u.username, u.display_name
		FROM drawings d
		INNER JOIN users u ON u.id = d.owner_id
		WHERE d.owner_id = $1
		ORDER BY d.updated_at DESC`

	return r.scanDrawings(ctx, query, ownerID)
}

// ListSharedWithUser returns drawings that have been shared with a user
// via messages (message_type = 'drawing' with drawing_id in metadata).
// This includes drawings shared to the user directly or to any group
// the user belongs to.
func (r *DrawingRepository) ListSharedWithUser(ctx context.Context, userID uuid.UUID) ([]model.DrawingWithUser, error) {
	query := `
		SELECT DISTINCT d.id, d.owner_id, d.name, d.geojson, d.created_at, d.updated_at,
		       u.username, u.display_name
		FROM drawings d
		INNER JOIN users u ON u.id = d.owner_id
		INNER JOIN messages m ON m.message_type = 'drawing'
		                     AND (m.metadata->>'drawing_id')::uuid = d.id
		                     AND (m.metadata->>'revoked' IS DISTINCT FROM 'true')
		WHERE d.owner_id != $1
		  AND (
		    m.recipient_id = $1
		    OR m.group_id IN (
		      SELECT gm.group_id FROM group_members gm WHERE gm.user_id = $1
		    )
		  )
		ORDER BY d.updated_at DESC`

	return r.scanDrawings(ctx, query, userID)
}

// Update modifies a drawing's name and geojson. Returns the updated record.
func (r *DrawingRepository) Update(ctx context.Context, d *model.Drawing) error {
	query := `
		UPDATE drawings
		SET name = $2, geojson = $3, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query, d.ID, d.Name, d.GeoJSON).Scan(&d.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("drawing")
		}
		return err
	}
	return nil
}

// Delete removes a drawing by ID. Returns NotFoundError if it does not exist.
func (r *DrawingRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM drawings WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("drawing")
	}
	return nil
}

// GetShareTargets returns the group IDs and direct recipient IDs that a
// drawing has been shared with (via messages with message_type = 'drawing').
// Used for broadcasting updates.
func (r *DrawingRepository) GetShareTargets(ctx context.Context, drawingID uuid.UUID) (groupIDs []uuid.UUID, userIDs []uuid.UUID, err error) {
	query := `
		SELECT m.group_id, m.recipient_id
		FROM messages m
		WHERE m.message_type = 'drawing'
		  AND (m.metadata->>'drawing_id')::uuid = $1
		  AND (m.metadata->>'revoked' IS DISTINCT FROM 'true')`

	rows, err := r.pool.Query(ctx, query, drawingID)
	if err != nil {
		return nil, nil, err
	}
	defer rows.Close()

	groupSet := make(map[uuid.UUID]struct{})
	userSet := make(map[uuid.UUID]struct{})

	for rows.Next() {
		var groupID *uuid.UUID
		var recipientID *uuid.UUID
		if err := rows.Scan(&groupID, &recipientID); err != nil {
			return nil, nil, err
		}
		if groupID != nil {
			groupSet[*groupID] = struct{}{}
		}
		if recipientID != nil {
			userSet[*recipientID] = struct{}{}
		}
	}
	if err := rows.Err(); err != nil {
		return nil, nil, err
	}

	for id := range groupSet {
		groupIDs = append(groupIDs, id)
	}
	for id := range userSet {
		userIDs = append(userIDs, id)
	}

	return groupIDs, userIDs, nil
}

// ListShares returns all active (non-revoked) share targets for a drawing,
// including group/user names.
func (r *DrawingRepository) ListShares(ctx context.Context, drawingID uuid.UUID) ([]model.DrawingShareInfo, error) {
	query := `
		SELECT 'group' AS type, g.id, g.name, m.created_at AS shared_at, m.id AS message_id
		FROM messages m
		INNER JOIN groups g ON g.id = m.group_id
		WHERE m.message_type = 'drawing'
		  AND (m.metadata->>'drawing_id')::uuid = $1
		  AND (m.metadata->>'revoked' IS DISTINCT FROM 'true')
		UNION ALL
		SELECT 'user', u.id, COALESCE(u.display_name, u.username), m.created_at, m.id
		FROM messages m
		INNER JOIN users u ON u.id = m.recipient_id
		WHERE m.message_type = 'drawing'
		  AND (m.metadata->>'drawing_id')::uuid = $1
		  AND (m.metadata->>'revoked' IS DISTINCT FROM 'true')
		ORDER BY shared_at DESC`

	rows, err := r.pool.Query(ctx, query, drawingID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var shares []model.DrawingShareInfo
	for rows.Next() {
		var s model.DrawingShareInfo
		if err := rows.Scan(&s.Type, &s.ID, &s.Name, &s.SharedAt, &s.MessageID); err != nil {
			return nil, err
		}
		shares = append(shares, s)
	}
	return shares, rows.Err()
}

// RevokeShare marks a share message as revoked by setting metadata.revoked = true.
func (r *DrawingRepository) RevokeShare(ctx context.Context, messageID uuid.UUID) error {
	query := `UPDATE messages SET metadata = metadata || '{"revoked": true}'::jsonb WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, messageID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("share message")
	}
	return nil
}

// --------------------------------------------------------------------------
// Internal helpers
// --------------------------------------------------------------------------

// scanDrawings runs a query and scans the results into DrawingWithUser slices.
func (r *DrawingRepository) scanDrawings(ctx context.Context, query string, args ...any) ([]model.DrawingWithUser, error) {
	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var drawings []model.DrawingWithUser
	for rows.Next() {
		var d model.DrawingWithUser
		if err := rows.Scan(
			&d.ID, &d.OwnerID, &d.Name, &d.GeoJSON,
			&d.CreatedAt, &d.UpdatedAt,
			&d.Username, &d.DisplayName,
		); err != nil {
			return nil, err
		}
		drawings = append(drawings, d)
	}
	return drawings, rows.Err()
}
