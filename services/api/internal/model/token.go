package model

import (
	"time"

	"github.com/google/uuid"
)

// RefreshToken represents a stored refresh token.
type RefreshToken struct {
	ID        uuid.UUID
	UserID    uuid.UUID
	TokenHash string
	ExpiresAt time.Time
	CreatedAt time.Time
}
