package auth

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/sitaware/api/internal/config"
)

// Claims represents the JWT payload for access tokens.
type Claims struct {
	jwt.RegisteredClaims
	UserID  uuid.UUID `json:"uid"`
	IsAdmin bool      `json:"adm"`
}

// JWTService handles token generation and validation.
type JWTService struct {
	secret     []byte
	accessTTL  time.Duration
	refreshTTL time.Duration
}

// NewJWTService creates a new JWTService from config.
func NewJWTService(cfg config.JWTConfig) *JWTService {
	return &JWTService{
		secret:     []byte(cfg.Secret),
		accessTTL:  cfg.AccessTokenTTL,
		refreshTTL: cfg.RefreshTokenTTL,
	}
}

// GenerateAccessToken creates a signed JWT access token for the given user.
func (s *JWTService) GenerateAccessToken(userID uuid.UUID, isAdmin bool) (string, error) {
	now := time.Now()
	claims := Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			Subject:   userID.String(),
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(s.accessTTL)),
			Issuer:    "sitaware",
		},
		UserID:  userID,
		IsAdmin: isAdmin,
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(s.secret)
}

// GenerateRefreshToken creates a random opaque refresh token (UUID).
func (s *JWTService) GenerateRefreshToken() string {
	return uuid.New().String()
}

// ValidateAccessToken parses and validates a JWT access token string.
func (s *JWTService) ValidateAccessToken(tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return s.secret, nil
	})
	if err != nil {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token claims")
	}

	return claims, nil
}

// HashRefreshToken returns a SHA-256 hex digest of the refresh token for DB storage.
func (s *JWTService) HashRefreshToken(token string) string {
	h := sha256.Sum256([]byte(token))
	return hex.EncodeToString(h[:])
}

// RefreshTokenTTL returns the configured refresh token lifetime.
func (s *JWTService) RefreshTokenTTL() time.Duration {
	return s.refreshTTL
}
