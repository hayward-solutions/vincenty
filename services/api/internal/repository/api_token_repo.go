package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// APITokenRepository handles database operations for API tokens.
type APITokenRepository struct {
	pool *pgxpool.Pool
}

// NewAPITokenRepository creates a new APITokenRepository.
func NewAPITokenRepository(pool *pgxpool.Pool) *APITokenRepository {
	return &APITokenRepository{pool: pool}
}

// Create inserts a new API token.
func (r *APITokenRepository) Create(ctx context.Context, token *model.APIToken) error {
	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}

	query := `
		INSERT INTO api_tokens (id, user_id, name, token_hash, expires_at)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at`

	return r.pool.QueryRow(ctx, query,
		token.ID, token.UserID, token.Name, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
}

// GetByTokenHash retrieves an API token by its SHA-256 hash, joining the
// users table to return the associated user. Returns ErrNotFound if the
// token does not exist or has expired.
func (r *APITokenRepository) GetByTokenHash(ctx context.Context, hash string) (*model.APIToken, *model.User, error) {
	query := `
		SELECT t.id, t.user_id, t.name, t.token_hash, t.expires_at, t.last_used_at, t.created_at,
		       u.id, u.username, u.email, u.password_hash, u.display_name, u.avatar_url,
		       u.marker_icon, u.marker_color, u.is_admin, u.is_active, u.mfa_enabled,
		       u.created_at, u.updated_at
		FROM api_tokens t
		JOIN users u ON u.id = t.user_id
		WHERE t.token_hash = $1
		  AND (t.expires_at IS NULL OR t.expires_at > NOW())`

	token := &model.APIToken{}
	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&token.ID, &token.UserID, &token.Name, &token.TokenHash,
		&token.ExpiresAt, &token.LastUsedAt, &token.CreatedAt,
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL,
		&user.MarkerIcon, &user.MarkerColor, &user.IsAdmin, &user.IsActive,
		&user.MFAEnabled, &user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil, model.ErrNotFound("api token")
		}
		return nil, nil, err
	}
	return token, user, nil
}

// ListByUserID retrieves all API tokens for a user, ordered by creation time.
func (r *APITokenRepository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]model.APIToken, error) {
	query := `
		SELECT id, user_id, name, token_hash, expires_at, last_used_at, created_at
		FROM api_tokens
		WHERE user_id = $1
		ORDER BY created_at DESC`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tokens []model.APIToken
	for rows.Next() {
		var t model.APIToken
		if err := rows.Scan(
			&t.ID, &t.UserID, &t.Name, &t.TokenHash,
			&t.ExpiresAt, &t.LastUsedAt, &t.CreatedAt,
		); err != nil {
			return nil, err
		}
		tokens = append(tokens, t)
	}
	return tokens, rows.Err()
}

// Delete removes an API token by ID, scoped to the owning user.
func (r *APITokenRepository) Delete(ctx context.Context, userID, tokenID uuid.UUID) error {
	query := `DELETE FROM api_tokens WHERE id = $1 AND user_id = $2`
	tag, err := r.pool.Exec(ctx, query, tokenID, userID)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("api token")
	}
	return nil
}

// TouchLastUsed updates the last_used_at timestamp for a token.
func (r *APITokenRepository) TouchLastUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE api_tokens SET last_used_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteExpired removes all expired API tokens. Returns the count deleted.
func (r *APITokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM api_tokens WHERE expires_at IS NOT NULL AND expires_at <= NOW()`
	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
