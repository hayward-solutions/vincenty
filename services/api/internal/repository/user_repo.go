package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/sitaware/api/internal/model"
)

// UserRepository handles database operations for users.
type UserRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(pool *pgxpool.Pool) *UserRepository {
	return &UserRepository{pool: pool}
}

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, user *model.User) error {
	query := `
		INSERT INTO users (id, username, email, password_hash, display_name, avatar_url, is_admin, is_active)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
		RETURNING created_at, updated_at`

	if user.ID == uuid.Nil {
		user.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.DisplayName, user.AvatarURL, user.IsAdmin, user.IsActive,
	).Scan(&user.CreatedAt, &user.UpdatedAt)
}

// GetByID retrieves a user by their ID.
func (r *UserRepository) GetByID(ctx context.Context, id uuid.UUID) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, is_admin, is_active, created_at, updated_at
		FROM users WHERE id = $1`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.IsAdmin, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("user")
		}
		return nil, err
	}
	return user, nil
}

// GetByUsername retrieves a user by their username.
func (r *UserRepository) GetByUsername(ctx context.Context, username string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, is_admin, is_active, created_at, updated_at
		FROM users WHERE username = $1`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, username).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.IsAdmin, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("user")
		}
		return nil, err
	}
	return user, nil
}

// GetByEmail retrieves a user by their email.
func (r *UserRepository) GetByEmail(ctx context.Context, email string) (*model.User, error) {
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, is_admin, is_active, created_at, updated_at
		FROM users WHERE email = $1`

	user := &model.User{}
	err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID, &user.Username, &user.Email, &user.PasswordHash,
		&user.DisplayName, &user.AvatarURL, &user.IsAdmin, &user.IsActive,
		&user.CreatedAt, &user.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("user")
		}
		return nil, err
	}
	return user, nil
}

// List retrieves a paginated list of users.
func (r *UserRepository) List(ctx context.Context, page, pageSize int) ([]model.User, int, error) {
	// Count total
	var total int
	countQuery := `SELECT COUNT(*) FROM users`
	if err := r.pool.QueryRow(ctx, countQuery).Scan(&total); err != nil {
		return nil, 0, err
	}

	// Fetch page
	offset := (page - 1) * pageSize
	query := `
		SELECT id, username, email, password_hash, display_name, avatar_url, is_admin, is_active, created_at, updated_at
		FROM users
		ORDER BY created_at DESC
		LIMIT $1 OFFSET $2`

	rows, err := r.pool.Query(ctx, query, pageSize, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var users []model.User
	for rows.Next() {
		var u model.User
		if err := rows.Scan(
			&u.ID, &u.Username, &u.Email, &u.PasswordHash,
			&u.DisplayName, &u.AvatarURL, &u.IsAdmin, &u.IsActive,
			&u.CreatedAt, &u.UpdatedAt,
		); err != nil {
			return nil, 0, err
		}
		users = append(users, u)
	}

	return users, total, rows.Err()
}

// Update modifies an existing user.
func (r *UserRepository) Update(ctx context.Context, user *model.User) error {
	query := `
		UPDATE users
		SET username = $2, email = $3, password_hash = $4, display_name = $5,
		    avatar_url = $6, is_admin = $7, is_active = $8, updated_at = NOW()
		WHERE id = $1
		RETURNING updated_at`

	err := r.pool.QueryRow(ctx, query,
		user.ID, user.Username, user.Email, user.PasswordHash,
		user.DisplayName, user.AvatarURL, user.IsAdmin, user.IsActive,
	).Scan(&user.UpdatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return model.ErrNotFound("user")
		}
		return err
	}
	return nil
}

// Delete removes a user by ID.
func (r *UserRepository) Delete(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("user")
	}
	return nil
}

// CountAdmins returns the number of active admin users.
func (r *UserRepository) CountAdmins(ctx context.Context) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM users WHERE is_admin = true AND is_active = true`
	err := r.pool.QueryRow(ctx, query).Scan(&count)
	return count, err
}

// ExistsByUsername checks if a user with the given username exists.
func (r *UserRepository) ExistsByUsername(ctx context.Context, username string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE username = $1)`
	err := r.pool.QueryRow(ctx, query, username).Scan(&exists)
	return exists, err
}

// ExistsByEmail checks if a user with the given email exists.
func (r *UserRepository) ExistsByEmail(ctx context.Context, email string) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM users WHERE email = $1)`
	err := r.pool.QueryRow(ctx, query, email).Scan(&exists)
	return exists, err
}
