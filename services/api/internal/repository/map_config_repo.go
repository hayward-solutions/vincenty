package repository

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// MapConfigRepository handles database operations for map configurations.
type MapConfigRepository struct {
	pool *pgxpool.Pool
}

// NewMapConfigRepository creates a new MapConfigRepository.
func NewMapConfigRepository(pool *pgxpool.Pool) *MapConfigRepository {
	return &MapConfigRepository{pool: pool}
}

// Create inserts a new map configuration.
func (r *MapConfigRepository) Create(ctx context.Context, mc *model.MapConfig) error {
	query := `
		INSERT INTO map_configs (id, name, source_type, tile_url, style_json, min_zoom, max_zoom, is_default, is_builtin, is_enabled, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if mc.ID == uuid.Nil {
		mc.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		mc.ID, mc.Name, mc.SourceType, mc.TileURL, mc.StyleJSON,
		mc.MinZoom, mc.MaxZoom,
		mc.IsDefault, mc.IsBuiltin, mc.IsEnabled, mc.CreatedBy,
	).Scan(&mc.CreatedAt, &mc.UpdatedAt)
}

// GetByID retrieves a map configuration by its ID.
func (r *MapConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.MapConfig, error) {
	query := `
		SELECT id, name, source_type, tile_url, style_json,
		       min_zoom, max_zoom,
		       is_default, is_builtin, is_enabled, created_by, created_at, updated_at
		FROM map_configs WHERE id = $1`

	mc := &model.MapConfig{}
	var styleBytes []byte
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&mc.ID, &mc.Name, &mc.SourceType, &mc.TileURL, &styleBytes,
		&mc.MinZoom, &mc.MaxZoom,
		&mc.IsDefault, &mc.IsBuiltin, &mc.IsEnabled, &mc.CreatedBy,
		&mc.CreatedAt, &mc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("map config")
		}
		return nil, err
	}
	if styleBytes != nil {
		raw := json.RawMessage(styleBytes)
		mc.StyleJSON = &raw
	}
	return mc, nil
}

// GetDefault retrieves the default map configuration that is also enabled, if any.
func (r *MapConfigRepository) GetDefault(ctx context.Context) (*model.MapConfig, error) {
	query := `
		SELECT id, name, source_type, tile_url, style_json,
		       min_zoom, max_zoom,
		       is_default, is_builtin, is_enabled, created_by, created_at, updated_at
		FROM map_configs WHERE is_default = true AND is_enabled = true
		LIMIT 1`

	mc := &model.MapConfig{}
	var styleBytes []byte
	err := r.pool.QueryRow(ctx, query).Scan(
		&mc.ID, &mc.Name, &mc.SourceType, &mc.TileURL, &styleBytes,
		&mc.MinZoom, &mc.MaxZoom,
		&mc.IsDefault, &mc.IsBuiltin, &mc.IsEnabled, &mc.CreatedBy,
		&mc.CreatedAt, &mc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no default configured
		}
		return nil, err
	}
	if styleBytes != nil {
		raw := json.RawMessage(styleBytes)
		mc.StyleJSON = &raw
	}
	return mc, nil
}

// List retrieves all map configurations ordered by name.
func (r *MapConfigRepository) List(ctx context.Context) ([]model.MapConfig, error) {
	query := `
		SELECT id, name, source_type, tile_url, style_json,
		       min_zoom, max_zoom,
		       is_default, is_builtin, is_enabled, created_by, created_at, updated_at
		FROM map_configs
		ORDER BY is_default DESC, name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.MapConfig
	for rows.Next() {
		var mc model.MapConfig
		var styleBytes []byte
		if err := rows.Scan(
			&mc.ID, &mc.Name, &mc.SourceType, &mc.TileURL, &styleBytes,
			&mc.MinZoom, &mc.MaxZoom,
			&mc.IsDefault, &mc.IsBuiltin, &mc.IsEnabled, &mc.CreatedBy,
			&mc.CreatedAt, &mc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		if styleBytes != nil {
			raw := json.RawMessage(styleBytes)
			mc.StyleJSON = &raw
		}
		configs = append(configs, mc)
	}

	return configs, rows.Err()
}

// Update modifies an existing map configuration.
func (r *MapConfigRepository) Update(ctx context.Context, mc *model.MapConfig) error {
	query := `
		UPDATE map_configs
		SET name = $2, source_type = $3, tile_url = $4, style_json = $5,
		    min_zoom = $6, max_zoom = $7,
		    is_default = $8, is_enabled = $9, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		mc.ID, mc.Name, mc.SourceType, mc.TileURL, mc.StyleJSON,
		mc.MinZoom, mc.MaxZoom,
		mc.IsDefault, mc.IsEnabled,
	).Scan(&mc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("map config")
		}
		return err
	}
	return nil
}

// Delete removes a map configuration by ID.
func (r *MapConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM map_configs WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("map config")
	}
	return nil
}

// ClearDefault sets is_default=false on all map configs.
// Call this before setting a new default.
func (r *MapConfigRepository) ClearDefault(ctx context.Context) error {
	query := `UPDATE map_configs SET is_default = false WHERE is_default = true`
	_, err := r.pool.Exec(ctx, query)
	return err
}

// CountBuiltin returns the number of built-in map configurations.
func (r *MapConfigRepository) CountBuiltin(ctx context.Context) (int64, error) {
	query := `SELECT COUNT(*) FROM map_configs WHERE is_builtin = true`
	var count int64
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}
