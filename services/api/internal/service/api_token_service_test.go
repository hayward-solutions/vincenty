package service

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository/mock"
)

func testAPITokenRepo() *mock.APITokenRepo {
	return &mock.APITokenRepo{
		TouchLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAPITokenService_Create_Success(t *testing.T) {
	repo := testAPITokenRepo()

	var storedHash string
	repo.CreateFn = func(_ context.Context, token *model.APIToken) error {
		storedHash = token.TokenHash
		token.CreatedAt = time.Now()
		return nil
	}

	svc := NewAPITokenService(repo)
	req := &model.CreateAPITokenRequest{Name: "test-token"}
	resp, err := svc.Create(context.Background(), uuid.New(), req)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.HasPrefix(resp.Token, model.APITokenPrefix) {
		t.Errorf("token should start with %q, got %q", model.APITokenPrefix, resp.Token[:10])
	}
	if len(resp.Token) != len(model.APITokenPrefix)+64 {
		t.Errorf("token length = %d, want %d", len(resp.Token), len(model.APITokenPrefix)+64)
	}
	if resp.Name != "test-token" {
		t.Errorf("name = %q", resp.Name)
	}
	if storedHash == "" {
		t.Error("token hash was not stored")
	}
	if storedHash == resp.Token {
		t.Error("raw token should not be stored as hash")
	}
}

func TestAPITokenService_Create_ValidationError(t *testing.T) {
	repo := testAPITokenRepo()
	svc := NewAPITokenService(repo)

	req := &model.CreateAPITokenRequest{Name: ""}
	_, err := svc.Create(context.Background(), uuid.New(), req)
	var ve *model.ValidationError
	if !errors.As(err, &ve) {
		t.Errorf("expected ValidationError, got %v", err)
	}
}

func TestAPITokenService_Create_RepoError(t *testing.T) {
	repo := testAPITokenRepo()
	repo.CreateFn = func(_ context.Context, _ *model.APIToken) error {
		return errors.New("db error")
	}

	svc := NewAPITokenService(repo)
	req := &model.CreateAPITokenRequest{Name: "test"}
	_, err := svc.Create(context.Background(), uuid.New(), req)
	if err == nil {
		t.Error("expected error from repo")
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestAPITokenService_List_Success(t *testing.T) {
	repo := testAPITokenRepo()
	userID := uuid.New()
	repo.ListByUserIDFn = func(_ context.Context, uid uuid.UUID) ([]model.APIToken, error) {
		if uid != userID {
			t.Errorf("userID = %v, want %v", uid, userID)
		}
		return []model.APIToken{
			{ID: uuid.New(), Name: "token-1", CreatedAt: time.Now()},
			{ID: uuid.New(), Name: "token-2", CreatedAt: time.Now()},
		}, nil
	}

	svc := NewAPITokenService(repo)
	tokens, err := svc.List(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 2 {
		t.Errorf("len = %d, want 2", len(tokens))
	}
	if tokens[0].Name != "token-1" {
		t.Errorf("tokens[0].Name = %q", tokens[0].Name)
	}
}

func TestAPITokenService_List_Empty(t *testing.T) {
	repo := testAPITokenRepo()
	repo.ListByUserIDFn = func(_ context.Context, _ uuid.UUID) ([]model.APIToken, error) {
		return nil, nil
	}

	svc := NewAPITokenService(repo)
	tokens, err := svc.List(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(tokens) != 0 {
		t.Errorf("len = %d, want 0", len(tokens))
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAPITokenService_Delete_Success(t *testing.T) {
	repo := testAPITokenRepo()
	userID := uuid.New()
	tokenID := uuid.New()
	repo.DeleteFn = func(_ context.Context, uid, tid uuid.UUID) error {
		if uid != userID || tid != tokenID {
			t.Errorf("unexpected IDs: user=%v token=%v", uid, tid)
		}
		return nil
	}

	svc := NewAPITokenService(repo)
	if err := svc.Delete(context.Background(), userID, tokenID); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

func TestAPITokenService_Delete_NotFound(t *testing.T) {
	repo := testAPITokenRepo()
	repo.DeleteFn = func(_ context.Context, _, _ uuid.UUID) error {
		return model.ErrNotFound("api token")
	}

	svc := NewAPITokenService(repo)
	err := svc.Delete(context.Background(), uuid.New(), uuid.New())
	var nf *model.NotFoundError
	if !errors.As(err, &nf) {
		t.Errorf("expected NotFoundError, got %v", err)
	}
}

// ---------------------------------------------------------------------------
// ValidateToken
// ---------------------------------------------------------------------------

func TestAPITokenService_ValidateToken_Success(t *testing.T) {
	repo := testAPITokenRepo()
	userID := uuid.New()
	tokenID := uuid.New()

	repo.GetByTokenHashFn = func(_ context.Context, _ string) (*model.APIToken, *model.User, error) {
		return &model.APIToken{ID: tokenID, UserID: userID},
			&model.User{ID: userID, IsAdmin: true, IsActive: true},
			nil
	}

	svc := NewAPITokenService(repo)
	claims, err := svc.ValidateToken(context.Background(), "sat_abc123")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if !claims.IsAdmin {
		t.Error("IsAdmin should be true")
	}
}

func TestAPITokenService_ValidateToken_NotAnAPIToken(t *testing.T) {
	repo := testAPITokenRepo()
	svc := NewAPITokenService(repo)

	_, err := svc.ValidateToken(context.Background(), "eyJhbGciOiJIUzI1NiJ9.xxx")
	if err == nil {
		t.Error("expected error for non-API token")
	}
}

func TestAPITokenService_ValidateToken_InvalidToken(t *testing.T) {
	repo := testAPITokenRepo()
	repo.GetByTokenHashFn = func(_ context.Context, _ string) (*model.APIToken, *model.User, error) {
		return nil, nil, model.ErrNotFound("api token")
	}

	svc := NewAPITokenService(repo)
	_, err := svc.ValidateToken(context.Background(), "sat_invalid")
	if err == nil {
		t.Error("expected error for invalid token")
	}
}

func TestAPITokenService_ValidateToken_InactiveUser(t *testing.T) {
	repo := testAPITokenRepo()
	repo.GetByTokenHashFn = func(_ context.Context, _ string) (*model.APIToken, *model.User, error) {
		return &model.APIToken{ID: uuid.New()},
			&model.User{ID: uuid.New(), IsActive: false},
			nil
	}

	svc := NewAPITokenService(repo)
	_, err := svc.ValidateToken(context.Background(), "sat_abc123")
	if err == nil {
		t.Error("expected error for inactive user")
	}
	if !strings.Contains(err.Error(), "disabled") {
		t.Errorf("error = %q, should mention disabled", err.Error())
	}
}
