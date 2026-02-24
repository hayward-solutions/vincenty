package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository/mock"
)

// testJWT creates a lightweight JWTService suitable for unit tests.
func testJWT() *auth.JWTService {
	return auth.NewJWTService(config.JWTConfig{
		Secret:          "test-secret-key-that-is-long-enough",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
}

// testUser returns a user with a bcrypt-hashed password for "password123".
func testUser() *model.User {
	hash, _ := auth.HashPassword("password123")
	return &model.User{
		ID:           uuid.New(),
		Username:     "alice",
		Email:        "alice@example.com",
		PasswordHash: hash,
		IsAdmin:      false,
		IsActive:     true,
	}
}

// ---------------------------------------------------------------------------
// BootstrapAdmin
// ---------------------------------------------------------------------------

func TestAuthService_BootstrapAdmin_CreatesAdmin(t *testing.T) {
	var createdUser *model.User
	userRepo := &mock.UserRepo{
		CountAdminsFn: func(ctx context.Context) (int, error) { return 0, nil },
		CreateFn: func(ctx context.Context, u *model.User) error {
			createdUser = u
			u.ID = uuid.New()
			return nil
		},
	}
	tokenRepo := &mock.TokenRepo{}
	svc := NewAuthService(userRepo, tokenRepo, testJWT(), nil)

	err := svc.BootstrapAdmin(context.Background(), config.AdminConfig{
		Username: "admin",
		Password: "admin-pass-1234",
		Email:    "admin@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if createdUser == nil {
		t.Fatal("expected user to be created")
	}
	if createdUser.Username != "admin" {
		t.Errorf("expected username admin, got %s", createdUser.Username)
	}
	if !createdUser.IsAdmin {
		t.Error("expected user to be admin")
	}
	if !createdUser.IsActive {
		t.Error("expected user to be active")
	}
	// Verify the password was hashed (not stored plaintext)
	if createdUser.PasswordHash == "admin-pass-1234" {
		t.Error("password should be hashed, not plaintext")
	}
	if err := auth.CheckPassword("admin-pass-1234", createdUser.PasswordHash); err != nil {
		t.Errorf("password hash should match: %v", err)
	}
}

func TestAuthService_BootstrapAdmin_SkipsIfAdminExists(t *testing.T) {
	userRepo := &mock.UserRepo{
		CountAdminsFn: func(ctx context.Context) (int, error) { return 1, nil },
		CreateFn: func(ctx context.Context, u *model.User) error {
			t.Fatal("Create should not be called")
			return nil
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	err := svc.BootstrapAdmin(context.Background(), config.AdminConfig{
		Username: "admin",
		Password: "admin-pass-1234",
		Email:    "admin@example.com",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ---------------------------------------------------------------------------
// Login
// ---------------------------------------------------------------------------

func TestAuthService_Login_Success(t *testing.T) {
	user := testUser()
	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			if username == user.Username {
				return user, nil
			}
			return nil, model.ErrNotFound("user")
		},
	}
	var storedToken *model.RefreshToken
	tokenRepo := &mock.TokenRepo{
		CreateFn: func(ctx context.Context, tok *model.RefreshToken) error {
			storedToken = tok
			return nil
		},
	}
	svc := NewAuthService(userRepo, tokenRepo, testJWT(), nil)

	resp, err := svc.Login(context.Background(), &model.LoginRequest{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected refresh token")
	}
	if resp.User.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.User.Username)
	}
	if storedToken == nil {
		t.Fatal("expected refresh token to be stored")
	}
	if storedToken.UserID != user.ID {
		t.Errorf("stored token user ID mismatch")
	}
}

func TestAuthService_Login_UserNotFound(t *testing.T) {
	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return nil, model.ErrNotFound("user")
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	_, err := svc.Login(context.Background(), &model.LoginRequest{
		Username: "nobody",
		Password: "password123",
	})
	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_WrongPassword(t *testing.T) {
	user := testUser()
	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return user, nil
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	_, err := svc.Login(context.Background(), &model.LoginRequest{
		Username: "alice",
		Password: "wrong-password",
	})
	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_DisabledAccount(t *testing.T) {
	user := testUser()
	user.IsActive = false
	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return user, nil
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	_, err := svc.Login(context.Background(), &model.LoginRequest{
		Username: "alice",
		Password: "password123",
	})
	var fe *model.ForbiddenError
	if !errors.As(err, &fe) {
		t.Fatalf("expected ForbiddenError, got %T: %v", err, err)
	}
}

func TestAuthService_Login_MFARequired(t *testing.T) {
	user := testUser()
	user.MFAEnabled = true

	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return user, nil
		},
	}

	// When mfaService is nil but user has MFA, the MFA path is skipped
	// (Login checks s.mfaService != nil). Verify that with nil mfaService
	// a normal login succeeds even when MFAEnabled is set.
	tokenRepo := &mock.TokenRepo{
		CreateFn: func(ctx context.Context, tok *model.RefreshToken) error { return nil },
	}
	svc := NewAuthService(userRepo, tokenRepo, testJWT(), nil)

	resp, err := svc.Login(context.Background(), &model.LoginRequest{
		Username: "alice",
		Password: "password123",
	})
	if err != nil {
		t.Fatalf("unexpected error (mfaService=nil should skip MFA): %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
}

func TestAuthService_Login_MFARequired_WithService(t *testing.T) {
	// This test verifies the MFA code path is entered when mfaService is non-nil
	// and user has MFA enabled. CreateMFAToken uses Redis, which requires a real
	// or mock Redis. We test that the code reaches CreateMFAToken by recovering
	// from the nil-pointer panic on rdb.
	user := testUser()
	user.MFAEnabled = true

	userRepo := &mock.UserRepo{
		GetByUsernameFn: func(ctx context.Context, username string) (*model.User, error) {
			return user, nil
		},
	}

	mfaRepo := &mock.MFARepo{
		HasVerifiedTOTPFn:          func(ctx context.Context, userID uuid.UUID) (bool, error) { return true, nil },
		HasWebAuthnFn:              func(ctx context.Context, userID uuid.UUID) (bool, error) { return false, nil },
		CountUnusedRecoveryCodesFn: func(ctx context.Context, userID uuid.UUID) (int, error) { return 5, nil },
	}
	enc, _ := auth.NewLocalEncryptor([]byte("test-secret-key-that-is-long-enough"))
	mfaSvc := NewMFAService(mfaRepo, userRepo, enc, nil, nil)

	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), mfaSvc)

	// The MFA path calls rdb.Set which will panic because rdb is nil.
	// We catch the panic to verify the MFA code path was entered.
	panicked := false
	func() {
		defer func() {
			if r := recover(); r != nil {
				panicked = true
			}
		}()
		svc.Login(context.Background(), &model.LoginRequest{
			Username: "alice",
			Password: "password123",
		})
	}()

	if !panicked {
		t.Error("expected panic from nil Redis client — MFA path should have been entered")
	}
}

// ---------------------------------------------------------------------------
// Refresh
// ---------------------------------------------------------------------------

func TestAuthService_Refresh_Success(t *testing.T) {
	jwt := testJWT()
	user := testUser()
	refreshToken := jwt.GenerateRefreshToken()
	hash := jwt.HashRefreshToken(refreshToken)

	userRepo := &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, model.ErrNotFound("user")
		},
	}
	tokenRepo := &mock.TokenRepo{
		GetByHashFn: func(ctx context.Context, h string) (*model.RefreshToken, error) {
			if h == hash {
				return &model.RefreshToken{
					ID:        uuid.New(),
					UserID:    user.ID,
					TokenHash: hash,
					ExpiresAt: time.Now().Add(7 * 24 * time.Hour),
				}, nil
			}
			return nil, model.ErrNotFound("token")
		},
		DeleteByHashFn: func(ctx context.Context, h string) error {
			return nil
		},
		CreateFn: func(ctx context.Context, tok *model.RefreshToken) error {
			return nil
		},
	}
	svc := NewAuthService(userRepo, tokenRepo, jwt, nil)

	resp, err := svc.Refresh(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected new access token")
	}
	if resp.RefreshToken == "" {
		t.Error("expected new refresh token")
	}
	// New refresh token should differ from old one (rotation)
	if resp.RefreshToken == refreshToken {
		t.Error("expected rotated refresh token")
	}
}

func TestAuthService_Refresh_InvalidToken(t *testing.T) {
	jwt := testJWT()
	tokenRepo := &mock.TokenRepo{
		GetByHashFn: func(ctx context.Context, hash string) (*model.RefreshToken, error) {
			return nil, model.ErrNotFound("token")
		},
	}
	svc := NewAuthService(&mock.UserRepo{}, tokenRepo, jwt, nil)

	_, err := svc.Refresh(context.Background(), "invalid-refresh-token")
	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %T: %v", err, err)
	}
}

func TestAuthService_Refresh_DisabledAccount(t *testing.T) {
	jwt := testJWT()
	user := testUser()
	user.IsActive = false
	refreshToken := jwt.GenerateRefreshToken()
	hash := jwt.HashRefreshToken(refreshToken)

	userRepo := &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return user, nil
		},
	}
	tokenRepo := &mock.TokenRepo{
		GetByHashFn: func(ctx context.Context, h string) (*model.RefreshToken, error) {
			return &model.RefreshToken{UserID: user.ID, TokenHash: hash}, nil
		},
		DeleteByHashFn: func(ctx context.Context, h string) error { return nil },
	}
	svc := NewAuthService(userRepo, tokenRepo, jwt, nil)

	_, err := svc.Refresh(context.Background(), refreshToken)
	var fe *model.ForbiddenError
	if !errors.As(err, &fe) {
		t.Fatalf("expected ForbiddenError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// Logout
// ---------------------------------------------------------------------------

func TestAuthService_Logout(t *testing.T) {
	jwt := testJWT()
	refreshToken := jwt.GenerateRefreshToken()
	expectedHash := jwt.HashRefreshToken(refreshToken)

	var deletedHash string
	tokenRepo := &mock.TokenRepo{
		DeleteByHashFn: func(ctx context.Context, hash string) error {
			deletedHash = hash
			return nil
		},
	}
	svc := NewAuthService(&mock.UserRepo{}, tokenRepo, jwt, nil)

	err := svc.Logout(context.Background(), refreshToken)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if deletedHash != expectedHash {
		t.Errorf("expected hash %s, got %s", expectedHash, deletedHash)
	}
}

// ---------------------------------------------------------------------------
// PasskeyLogin
// ---------------------------------------------------------------------------

func TestAuthService_PasskeyLogin_Success(t *testing.T) {
	user := testUser()
	userRepo := &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			if id == user.ID {
				return user, nil
			}
			return nil, model.ErrNotFound("user")
		},
	}
	tokenRepo := &mock.TokenRepo{
		CreateFn: func(ctx context.Context, tok *model.RefreshToken) error { return nil },
	}
	svc := NewAuthService(userRepo, tokenRepo, testJWT(), nil)

	resp, err := svc.PasskeyLogin(context.Background(), user.ID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if resp.AccessToken == "" {
		t.Error("expected access token")
	}
	if resp.User.Username != "alice" {
		t.Errorf("expected username alice, got %s", resp.User.Username)
	}
}

func TestAuthService_PasskeyLogin_DisabledAccount(t *testing.T) {
	user := testUser()
	user.IsActive = false
	userRepo := &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return user, nil
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	_, err := svc.PasskeyLogin(context.Background(), user.ID)
	var fe *model.ForbiddenError
	if !errors.As(err, &fe) {
		t.Fatalf("expected ForbiddenError, got %T: %v", err, err)
	}
}

func TestAuthService_PasskeyLogin_UserNotFound(t *testing.T) {
	userRepo := &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return nil, model.ErrNotFound("user")
		},
	}
	svc := NewAuthService(userRepo, &mock.TokenRepo{}, testJWT(), nil)

	_, err := svc.PasskeyLogin(context.Background(), uuid.New())
	var nf *model.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// CompleteMFALogin
// ---------------------------------------------------------------------------

func TestAuthService_CompleteMFALogin_DisabledAccount(t *testing.T) {
	user := testUser()
	user.IsActive = false

	// For this test we need an MFAService that can validate a token.
	// Since MFAService.ValidateMFAToken uses Redis, we'll test this path
	// by mocking the AuthService more directly. Instead, we verify that
	// CompleteMFALogin correctly returns ForbiddenError for a disabled user
	// by using a custom flow that bypasses Redis.
	// This is tested at the integration level; here we test BootstrapAdmin/Login/Refresh.
	t.Skip("requires Redis for MFA token validation — tested at integration level")
}
