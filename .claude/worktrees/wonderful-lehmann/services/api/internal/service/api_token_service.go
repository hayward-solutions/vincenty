package service

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"strings"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository"
)

// APITokenService handles API token operations.
type APITokenService struct {
	repo repository.APITokenRepo
}

// NewAPITokenService creates a new APITokenService.
func NewAPITokenService(repo repository.APITokenRepo) *APITokenService {
	return &APITokenService{repo: repo}
}

// Create generates a new API token for the given user.
// The raw token string is returned only once; only its SHA-256 hash is persisted.
func (s *APITokenService) Create(ctx context.Context, userID uuid.UUID, req *model.CreateAPITokenRequest) (*model.CreateAPITokenResponse, error) {
	if err := req.Validate(); err != nil {
		return nil, err
	}

	raw, err := generateRawToken()
	if err != nil {
		return nil, fmt.Errorf("generate token: %w", err)
	}

	token := &model.APIToken{
		UserID:    userID,
		Name:      req.Name,
		TokenHash: hashToken(raw),
		ExpiresAt: req.ExpiresAt,
	}

	if err := s.repo.Create(ctx, token); err != nil {
		return nil, fmt.Errorf("create api token: %w", err)
	}

	return &model.CreateAPITokenResponse{
		Token:            raw,
		APITokenResponse: token.ToResponse(),
	}, nil
}

// List returns all API tokens for the given user.
func (s *APITokenService) List(ctx context.Context, userID uuid.UUID) ([]model.APITokenResponse, error) {
	tokens, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("list api tokens: %w", err)
	}

	resp := make([]model.APITokenResponse, len(tokens))
	for i, t := range tokens {
		resp[i] = t.ToResponse()
	}
	return resp, nil
}

// Delete revokes an API token by ID, scoped to the owning user.
func (s *APITokenService) Delete(ctx context.Context, userID, tokenID uuid.UUID) error {
	return s.repo.Delete(ctx, userID, tokenID)
}

// ValidateToken checks a raw token string (with sat_ prefix) against the
// database and returns auth claims if valid. This method satisfies the
// middleware.TokenValidator interface.
func (s *APITokenService) ValidateToken(ctx context.Context, raw string) (*auth.Claims, error) {
	if !strings.HasPrefix(raw, model.APITokenPrefix) {
		return nil, fmt.Errorf("not an api token")
	}

	hash := hashToken(raw)

	token, user, err := s.repo.GetByTokenHash(ctx, hash)
	if err != nil {
		return nil, fmt.Errorf("invalid api token: %w", err)
	}

	if !user.IsActive {
		return nil, fmt.Errorf("user account is disabled")
	}

	// Update last_used_at asynchronously so it doesn't slow down the request.
	go func() {
		if err := s.repo.TouchLastUsed(context.Background(), token.ID); err != nil {
			slog.Error("failed to touch api token last_used_at", "error", err, "token_id", token.ID)
		}
	}()

	return &auth.Claims{
		UserID:  user.ID,
		IsAdmin: user.IsAdmin,
	}, nil
}

// generateRawToken produces a token string: sat_ + 32 random bytes hex-encoded.
func generateRawToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return model.APITokenPrefix + hex.EncodeToString(b), nil
}

// hashToken returns the SHA-256 hex digest of a raw token string.
func hashToken(raw string) string {
	h := sha256.Sum256([]byte(raw))
	return hex.EncodeToString(h[:])
}
