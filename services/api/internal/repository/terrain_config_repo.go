package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// TerrainConfigRepository handles database operations for terrain configurations.
type TerrainConfigRepository struct {
	pool *pgxpool.Pool
}

// NewTerrainConfigRepository creates a new TerrainConfigRepository.
func NewTerrainConfigRepository(pool *pgxpool.Pool) *TerrainConfigRepository {
	return &TerrainConfigRepository{pool: pool}
}

// Create inserts a new terrain configuration.
func (r *TerrainConfigRepository) Create(ctx context.Context, tc *model.TerrainConfig) error {
	query := `
		INSERT INTO terrain_configs (id, name, source_type, terrain_url, terrain_encoding, is_default, created_by)
		VALUES ($1, $2, $3, $4, $5, $6, $7)
		RETURNING created_at, updated_at`

	if tc.ID == uuid.Nil {
		tc.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		tc.ID, tc.Name, tc.SourceType, tc.TerrainURL, tc.TerrainEncoding,
		tc.IsDefault, tc.CreatedBy,
	).Scan(&tc.CreatedAt, &tc.UpdatedAt)
}

// GetByID retrieves a terrain configuration by its ID.
func (r *TerrainConfigRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.TerrainConfig, error) {
	query := `
		SELECT id, name, source_type, terrain_url, terrain_encoding,
		       is_default, created_by, created_at, updated_at
		FROM terrain_configs WHERE id = $1`

	tc := &model.TerrainConfig{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&tc.ID, &tc.Name, &tc.SourceType, &tc.TerrainURL, &tc.TerrainEncoding,
		&tc.IsDefault, &tc.CreatedBy,
		&tc.CreatedAt, &tc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("terrain config")
		}
		return nil, err
	}
	return tc, nil
}

// GetDefault retrieves the default terrain configuration, if any.
func (r *TerrainConfigRepository) GetDefault(ctx context.Context) (*model.TerrainConfig, error) {
	query := `
		SELECT id, name, source_type, terrain_url, terrain_encoding,
		       is_default, created_by, created_at, updated_at
		FROM terrain_configs WHERE is_default = true
		LIMIT 1`

	tc := &model.TerrainConfig{}
	err := r.pool.QueryRow(ctx, query).Scan(
		&tc.ID, &tc.Name, &tc.SourceType, &tc.TerrainURL, &tc.TerrainEncoding,
		&tc.IsDefault, &tc.CreatedBy,
		&tc.CreatedAt, &tc.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil // no default configured
		}
		return nil, err
	}
	return tc, nil
}

// List retrieves all terrain configurations ordered by name.
func (r *TerrainConfigRepository) List(ctx context.Context) ([]model.TerrainConfig, error) {
	query := `
		SELECT id, name, source_type, terrain_url, terrain_encoding,
		       is_default, created_by, created_at, updated_at
		FROM terrain_configs
		ORDER BY is_default DESC, name ASC`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var configs []model.TerrainConfig
	for rows.Next() {
		var tc model.TerrainConfig
		if err := rows.Scan(
			&tc.ID, &tc.Name, &tc.SourceType, &tc.TerrainURL, &tc.TerrainEncoding,
			&tc.IsDefault, &tc.CreatedBy,
			&tc.CreatedAt, &tc.UpdatedAt,
		); err != nil {
			return nil, err
		}
		configs = append(configs, tc)
	}

	return configs, rows.Err()
}

// Update modifies an existing terrain configuration.
func (r *TerrainConfigRepository) Update(ctx context.Context, tc *model.TerrainConfig) error {
	query := `
		UPDATE terrain_configs
		SET name = $2, source_type = $3, terrain_url = $4, terrain_encoding = $5,
		    is_default = $6, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		tc.ID, tc.Name, tc.SourceType, tc.TerrainURL, tc.TerrainEncoding,
		tc.IsDefault,
	).Scan(&tc.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("terrain config")
		}
		return err
	}
	return nil
}

// Delete removes a terrain configuration by ID.
func (r *TerrainConfigRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM terrain_configs WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("terrain config")
	}
	return nil
}

// ClearDefault sets is_default=false on all terrain configs.
// Call this before setting a new default.
func (r *TerrainConfigRepository) ClearDefault(ctx context.Context) error {
	query := `UPDATE terrain_configs SET is_default = false WHERE is_default = true`
	_, err := r.pool.Exec(ctx, query)
	return err
}
