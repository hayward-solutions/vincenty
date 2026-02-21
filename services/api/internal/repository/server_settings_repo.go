package repository

import (
	"context"
	"errors"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// ServerSettingsRepository handles database operations for server settings.
type ServerSettingsRepository struct {
	pool *pgxpool.Pool
}

// NewServerSettingsRepository creates a new ServerSettingsRepository.
func NewServerSettingsRepository(pool *pgxpool.Pool) *ServerSettingsRepository {
	return &ServerSettingsRepository{pool: pool}
}

// Get retrieves a server setting by key.
func (r *ServerSettingsRepository) Get(ctx context.Context, key string) (*model.ServerSetting, error) {
	query := `SELECT key, value, updated_at FROM server_settings WHERE key = $1`

	s := &model.ServerSetting{}
	err := r.pool.QueryRow(ctx, query, key).Scan(&s.Key, &s.Value, &s.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("server setting")
		}
		return nil, err
	}
	return s, nil
}

// Set upserts a server setting.
func (r *ServerSettingsRepository) Set(ctx context.Context, key, value string) error {
	query := `
		INSERT INTO server_settings (key, value, updated_at)
		VALUES ($1, $2, NOW())
		ON CONFLICT (key) DO UPDATE SET value = $2, updated_at = NOW()`

	_, err := r.pool.Exec(ctx, query, key, value)
	return err
}

// GetAll retrieves all server settings.
func (r *ServerSettingsRepository) GetAll(ctx context.Context) ([]model.ServerSetting, error) {
	query := `SELECT key, value, updated_at FROM server_settings ORDER BY key`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var settings []model.ServerSetting
	for rows.Next() {
		var s model.ServerSetting
		if err := rows.Scan(&s.Key, &s.Value, &s.UpdatedAt); err != nil {
			return nil, err
		}
		settings = append(settings, s)
	}
	return settings, rows.Err()
}
