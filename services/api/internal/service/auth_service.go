package service

import (
	"context"
	"errors"
	"log/slog"
	"time"

	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

// AuthService handles authentication business logic.
type AuthService struct {
	userRepo  *repository.UserRepository
	tokenRepo *repository.TokenRepository
	jwt       *auth.JWTService
}

// NewAuthService creates a new AuthService.
func NewAuthService(
	userRepo *repository.UserRepository,
	tokenRepo *repository.TokenRepository,
	jwt *auth.JWTService,
) *AuthService {
	return &AuthService{
		userRepo:  userRepo,
		tokenRepo: tokenRepo,
		jwt:       jwt,
	}
}

// Login authenticates a user and returns access + refresh tokens.
func (s *AuthService) Login(ctx context.Context, req *model.LoginRequest) (*model.AuthResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		var notFound *model.NotFoundError
		if errors.As(err, &notFound) {
			return nil, model.ErrValidation("invalid username or password")
		}
		return nil, err
	}

	if !user.IsActive {
		return nil, model.ErrForbidden("account is disabled")
	}

	if err := auth.CheckPassword(req.Password, user.PasswordHash); err != nil {
		return nil, model.ErrValidation("invalid username or password")
	}

	return s.generateTokens(ctx, user)
}

// Refresh exchanges a valid refresh token for a new token pair.
// Implements token rotation: the old refresh token is deleted.
func (s *AuthService) Refresh(ctx context.Context, refreshToken string) (*model.AuthResponse, error) {
	hash := s.jwt.HashRefreshToken(refreshToken)

	stored, err := s.tokenRepo.GetByHash(ctx, hash)
	if err != nil {
		var notFound *model.NotFoundError
		if errors.As(err, &notFound) {
			return nil, model.ErrValidation("invalid or expired refresh token")
		}
		return nil, err
	}

	// Delete the old token (rotation)
	if err := s.tokenRepo.DeleteByHash(ctx, hash); err != nil {
		return nil, err
	}

	// Look up the user to get current state (admin status may have changed)
	user, err := s.userRepo.GetByID(ctx, stored.UserID)
	if err != nil {
		return nil, err
	}

	if !user.IsActive {
		return nil, model.ErrForbidden("account is disabled")
	}

	return s.generateTokens(ctx, user)
}

// Logout revokes a refresh token.
func (s *AuthService) Logout(ctx context.Context, refreshToken string) error {
	hash := s.jwt.HashRefreshToken(refreshToken)
	return s.tokenRepo.DeleteByHash(ctx, hash)
}

// BootstrapAdmin creates the initial admin user if no admins exist.
func (s *AuthService) BootstrapAdmin(ctx context.Context, cfg config.AdminConfig) error {
	count, err := s.userRepo.CountAdmins(ctx)
	if err != nil {
		return err
	}
	if count > 0 {
		slog.Debug("admin user already exists, skipping bootstrap")
		return nil
	}

	hash, err := auth.HashPassword(cfg.Password)
	if err != nil {
		return err
	}

	user := &model.User{
		Username:     cfg.Username,
		Email:        cfg.Email,
		PasswordHash: hash,
		IsAdmin:      true,
		IsActive:     true,
	}

	if err := s.userRepo.Create(ctx, user); err != nil {
		return err
	}

	slog.Info("bootstrap admin user created",
		"username", cfg.Username,
		"email", cfg.Email,
	)
	return nil
}

// generateTokens creates a new access/refresh token pair and stores the refresh token.
func (s *AuthService) generateTokens(ctx context.Context, user *model.User) (*model.AuthResponse, error) {
	accessToken, err := s.jwt.GenerateAccessToken(user.ID, user.IsAdmin)
	if err != nil {
		return nil, err
	}

	refreshToken := s.jwt.GenerateRefreshToken()
	hash := s.jwt.HashRefreshToken(refreshToken)

	stored := &model.RefreshToken{
		UserID:    user.ID,
		TokenHash: hash,
		ExpiresAt: time.Now().Add(s.jwt.RefreshTokenTTL()),
	}
	if err := s.tokenRepo.Create(ctx, stored); err != nil {
		return nil, err
	}

	return &model.AuthResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		User:         user.ToResponse(),
	}, nil
}
