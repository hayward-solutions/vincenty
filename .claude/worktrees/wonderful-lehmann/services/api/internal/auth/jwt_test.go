package auth

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/config"
)

func newTestJWTService() *JWTService {
	return NewJWTService(config.JWTConfig{
		Secret:          "test-secret-at-least-32-bytes-long!!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})
}

func TestGenerateAccessToken(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(userID, true)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}
	if token == "" {
		t.Error("GenerateAccessToken() returned empty token")
	}
}

func TestValidateAccessToken_Valid(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, err := svc.GenerateAccessToken(userID, true)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", claims.UserID, userID)
	}
	if !claims.IsAdmin {
		t.Error("claims.IsAdmin = false, want true")
	}
	if claims.Issuer != "vincenty" {
		t.Errorf("claims.Issuer = %q, want %q", claims.Issuer, "vincenty")
	}
	if claims.Subject != userID.String() {
		t.Errorf("claims.Subject = %q, want %q", claims.Subject, userID.String())
	}
}

func TestValidateAccessToken_NonAdmin(t *testing.T) {
	svc := newTestJWTService()
	userID := uuid.New()

	token, _ := svc.GenerateAccessToken(userID, false)
	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}
	if claims.IsAdmin {
		t.Error("claims.IsAdmin = true, want false")
	}
}

func TestValidateAccessToken_Expired(t *testing.T) {
	svc := NewJWTService(config.JWTConfig{
		Secret:          "test-secret-at-least-32-bytes-long!!",
		AccessTokenTTL:  -1 * time.Hour, // already expired
		RefreshTokenTTL: 168 * time.Hour,
	})

	token, err := svc.GenerateAccessToken(uuid.New(), false)
	if err != nil {
		t.Fatalf("GenerateAccessToken() error = %v", err)
	}

	_, err = svc.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken() expected error for expired token, got nil")
	}
}

func TestValidateAccessToken_WrongSecret(t *testing.T) {
	svc1 := newTestJWTService()
	svc2 := NewJWTService(config.JWTConfig{
		Secret:          "different-secret-than-the-original!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})

	token, _ := svc1.GenerateAccessToken(uuid.New(), false)
	_, err := svc2.ValidateAccessToken(token)
	if err == nil {
		t.Error("ValidateAccessToken() expected error for wrong secret, got nil")
	}
}

func TestValidateAccessToken_Malformed(t *testing.T) {
	svc := newTestJWTService()

	_, err := svc.ValidateAccessToken("not.a.valid.jwt")
	if err == nil {
		t.Error("ValidateAccessToken() expected error for malformed token, got nil")
	}
}

func TestValidateAccessToken_EmptyString(t *testing.T) {
	svc := newTestJWTService()

	_, err := svc.ValidateAccessToken("")
	if err == nil {
		t.Error("ValidateAccessToken() expected error for empty token, got nil")
	}
}

func TestGenerateRefreshToken_Unique(t *testing.T) {
	svc := newTestJWTService()

	tokens := make(map[string]bool)
	for i := 0; i < 100; i++ {
		token := svc.GenerateRefreshToken()
		if token == "" {
			t.Fatal("GenerateRefreshToken() returned empty token")
		}
		if tokens[token] {
			t.Fatalf("GenerateRefreshToken() produced duplicate token on iteration %d", i)
		}
		tokens[token] = true
	}
}

func TestHashRefreshToken_Deterministic(t *testing.T) {
	svc := newTestJWTService()
	token := "test-refresh-token-value"

	hash1 := svc.HashRefreshToken(token)
	hash2 := svc.HashRefreshToken(token)

	if hash1 != hash2 {
		t.Errorf("HashRefreshToken() not deterministic: %q != %q", hash1, hash2)
	}
	if hash1 == "" {
		t.Error("HashRefreshToken() returned empty string")
	}
}

func TestHashRefreshToken_DifferentTokensDifferentHashes(t *testing.T) {
	svc := newTestJWTService()

	hash1 := svc.HashRefreshToken("token-a")
	hash2 := svc.HashRefreshToken("token-b")

	if hash1 == hash2 {
		t.Error("HashRefreshToken() produced same hash for different tokens")
	}
}

func TestRefreshTokenTTL(t *testing.T) {
	svc := newTestJWTService()
	expected := 168 * time.Hour

	if svc.RefreshTokenTTL() != expected {
		t.Errorf("RefreshTokenTTL() = %v, want %v", svc.RefreshTokenTTL(), expected)
	}
}

func TestGenerateAccessToken_ExpiresInFuture(t *testing.T) {
	svc := newTestJWTService()

	token, _ := svc.GenerateAccessToken(uuid.New(), false)
	claims, err := svc.ValidateAccessToken(token)
	if err != nil {
		t.Fatalf("ValidateAccessToken() error = %v", err)
	}

	if claims.ExpiresAt == nil {
		t.Fatal("claims.ExpiresAt is nil")
	}
	if !claims.ExpiresAt.Time.After(time.Now()) {
		t.Error("token should expire in the future")
	}
	if claims.ExpiresAt.Time.After(time.Now().Add(16 * time.Minute)) {
		t.Error("token should expire within ~15 minutes")
	}
}
