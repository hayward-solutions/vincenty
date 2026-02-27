package model

import (
	"strings"
	"time"

	"github.com/google/uuid"
)

// APITokenPrefix is prepended to every raw API token so the auth middleware
// can cheaply distinguish API tokens from JWTs without attempting validation.
const APITokenPrefix = "sat_"

// APIToken represents a long-lived API token bound to a user.
type APIToken struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Name       string
	TokenHash  string // SHA-256 hex of the raw token
	ExpiresAt  *time.Time
	LastUsedAt *time.Time
	CreatedAt  time.Time
}

// APITokenResponse is the JSON representation returned by list/get endpoints.
type APITokenResponse struct {
	ID         uuid.UUID  `json:"id"`
	Name       string     `json:"name"`
	ExpiresAt  *time.Time `json:"expires_at,omitempty"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

// ToResponse converts an APIToken to its safe API representation.
func (t *APIToken) ToResponse() APITokenResponse {
	return APITokenResponse{
		ID:         t.ID,
		Name:       t.Name,
		ExpiresAt:  t.ExpiresAt,
		LastUsedAt: t.LastUsedAt,
		CreatedAt:  t.CreatedAt,
	}
}

// CreateAPITokenResponse is returned once on creation and includes the raw
// token value. The raw token is never stored and cannot be retrieved again.
type CreateAPITokenResponse struct {
	Token string `json:"token"`
	APITokenResponse
}

// CreateAPITokenRequest is the expected body for creating a new API token.
type CreateAPITokenRequest struct {
	Name      string     `json:"name"`
	ExpiresAt *time.Time `json:"expires_at,omitempty"`
}

// Validate checks that required fields are present and valid.
func (r *CreateAPITokenRequest) Validate() error {
	r.Name = strings.TrimSpace(r.Name)
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	if len(r.Name) > 100 {
		return ErrValidation("name must not exceed 100 characters")
	}
	if r.ExpiresAt != nil && r.ExpiresAt.Before(time.Now()) {
		return ErrValidation("expires_at must be in the future")
	}
	return nil
}
