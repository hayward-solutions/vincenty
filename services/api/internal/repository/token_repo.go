package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// TokenRepository handles database operations for refresh tokens.
type TokenRepository struct {
	pool *pgxpool.Pool
}

// NewTokenRepository creates a new TokenRepository.
func NewTokenRepository(pool *pgxpool.Pool) *TokenRepository {
	return &TokenRepository{pool: pool}
}

// Create inserts a new refresh token.
func (r *TokenRepository) Create(ctx context.Context, token *model.RefreshToken) error {
	query := `
		INSERT INTO refresh_tokens (id, user_id, token_hash, expires_at)
		VALUES ($1, $2, $3, $4)
		RETURNING created_at`

	if token.ID == uuid.Nil {
		token.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		token.ID, token.UserID, token.TokenHash, token.ExpiresAt,
	).Scan(&token.CreatedAt)
}

// GetByHash retrieves a refresh token by its hash.
func (r *TokenRepository) GetByHash(ctx context.Context, hash string) (*model.RefreshToken, error) {
	query := `
		SELECT id, user_id, token_hash, expires_at, created_at
		FROM refresh_tokens
		WHERE token_hash = $1 AND expires_at > NOW()`

	token := &model.RefreshToken{}
	err := r.pool.QueryRow(ctx, query, hash).Scan(
		&token.ID, &token.UserID, &token.TokenHash,
		&token.ExpiresAt, &token.CreatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("refresh token")
		}
		return nil, err
	}
	return token, nil
}

// DeleteByHash removes a refresh token by its hash.
func (r *TokenRepository) DeleteByHash(ctx context.Context, hash string) error {
	query := `DELETE FROM refresh_tokens WHERE token_hash = $1`
	_, err := r.pool.Exec(ctx, query, hash)
	return err
}

// DeleteAllForUser removes all refresh tokens for a user.
func (r *TokenRepository) DeleteAllForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM refresh_tokens WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// DeleteExpired removes all expired refresh tokens. Returns the count deleted.
func (r *TokenRepository) DeleteExpired(ctx context.Context) (int64, error) {
	query := `DELETE FROM refresh_tokens WHERE expires_at <= NOW()`
	tag, err := r.pool.Exec(ctx, query)
	if err != nil {
		return 0, err
	}
	return tag.RowsAffected(), nil
}
