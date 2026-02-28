package ws

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/config"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository/mock"
)

func testJWTService() *auth.JWTService {
	return auth.NewJWTService(config.JWTConfig{
		Secret:          "test-secret-key-for-ws-handler",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	})
}

func testHandler() (*Handler, *auth.JWTService, *mock.DeviceRepo, *mock.GroupRepo) {
	jwt := testJWTService()
	deviceRepo := &mock.DeviceRepo{}
	groupRepo := &mock.GroupRepo{}
	hub := newTestHub()
	hub.groupRepo = groupRepo
	hub.userRepo = &mock.UserRepo{}
	h := NewHandler(hub, jwt, nil, deviceRepo, groupRepo)
	return h, jwt, deviceRepo, groupRepo
}

// ---------------------------------------------------------------------------
// Pre-upgrade validation tests (no actual WebSocket upgrade)
// ---------------------------------------------------------------------------

// mockTokenValidator implements TokenValidator for testing.
type mockTokenValidator struct {
	validateFn func(ctx context.Context, raw string) (*auth.Claims, error)
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, raw string) (*auth.Claims, error) {
	return m.validateFn(ctx, raw)
}

// testHandlerWithTokenValidator creates a handler that uses an API token validator.
func testHandlerWithTokenValidator(tv TokenValidator) (*Handler, *auth.JWTService, *mock.DeviceRepo, *mock.GroupRepo) {
	jwt := testJWTService()
	deviceRepo := &mock.DeviceRepo{}
	groupRepo := &mock.GroupRepo{}
	hub := newTestHub()
	hub.groupRepo = groupRepo
	hub.userRepo = &mock.UserRepo{}
	h := NewHandler(hub, jwt, tv, deviceRepo, groupRepo)
	return h, jwt, deviceRepo, groupRepo
}

// ---------------------------------------------------------------------------
// API token authentication tests
// ---------------------------------------------------------------------------

func TestHandler_APIToken_ValidToken(t *testing.T) {
	userID := uuid.New()
	deviceID := uuid.New()

	tv := &mockTokenValidator{
		validateFn: func(_ context.Context, raw string) (*auth.Claims, error) {
			return &auth.Claims{UserID: userID, IsAdmin: false}, nil
		},
	}
	h, _, deviceRepo, groupRepo := testHandlerWithTokenValidator(tv)

	deviceRepo.GetByIDFn = func(_ context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, UserID: userID, Name: "CLI"}, nil
	}
	deviceRepo.TouchLastSeenFn = func(_ context.Context, _ uuid.UUID, _ *string, _ *string) error {
		return nil
	}
	groupRepo.ListByUserIDFn = func(_ context.Context, uid uuid.UUID) ([]model.Group, []int, error) {
		return nil, nil, nil
	}
	h.hub.userRepo = &mock.UserRepo{
		GetByIDFn: func(_ context.Context, id uuid.UUID) (*model.User, error) {
			return &model.User{ID: userID, Username: "testuser"}, nil
		},
	}

	req := httptest.NewRequest("GET", "/api/v1/ws?token=sat_validtoken&device_id="+deviceID.String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	// We can't complete the WebSocket upgrade in a test, but we should NOT get
	// an auth error (401/400/403). The failure we expect is the upgrade itself
	// since httptest.ResponseRecorder doesn't support hijacking.
	if rec.Code == http.StatusUnauthorized || rec.Code == http.StatusBadRequest || rec.Code == http.StatusForbidden {
		t.Errorf("got auth/validation error %d, API token should have been accepted", rec.Code)
	}
}

func TestHandler_APIToken_InvalidToken(t *testing.T) {
	tv := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (*auth.Claims, error) {
			return nil, errors.New("invalid api token")
		},
	}
	h, _, _, _ := testHandlerWithTokenValidator(tv)

	req := httptest.NewRequest("GET", "/api/v1/ws?token=sat_badtoken&device_id="+uuid.New().String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

// ---------------------------------------------------------------------------
// Pre-upgrade validation tests (no actual WebSocket upgrade)
// ---------------------------------------------------------------------------

func TestHandler_MissingToken(t *testing.T) {
	h, _, _, _ := testHandler()

	req := httptest.NewRequest("GET", "/api/v1/ws", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_InvalidToken(t *testing.T) {
	h, _, _, _ := testHandler()

	req := httptest.NewRequest("GET", "/api/v1/ws?token=invalid", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusUnauthorized {
		t.Errorf("expected %d, got %d", http.StatusUnauthorized, rec.Code)
	}
}

func TestHandler_MissingDeviceID(t *testing.T) {
	h, jwt, _, _ := testHandler()

	token, _ := jwt.GenerateAccessToken(uuid.New(), false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token, nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_InvalidDeviceID(t *testing.T) {
	h, jwt, _, _ := testHandler()

	token, _ := jwt.GenerateAccessToken(uuid.New(), false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token+"&device_id=not-a-uuid", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_DeviceNotFound(t *testing.T) {
	h, jwt, deviceRepo, _ := testHandler()

	deviceRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return nil, model.ErrNotFound("device")
	}

	deviceID := uuid.New()
	token, _ := jwt.GenerateAccessToken(uuid.New(), false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token+"&device_id="+deviceID.String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusBadRequest {
		t.Errorf("expected %d, got %d", http.StatusBadRequest, rec.Code)
	}
}

func TestHandler_DeviceBelongsToDifferentUser(t *testing.T) {
	h, jwt, deviceRepo, _ := testHandler()

	otherUserID := uuid.New()
	deviceID := uuid.New()
	deviceRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, UserID: otherUserID, Name: "Phone"}, nil
	}

	userID := uuid.New()
	token, _ := jwt.GenerateAccessToken(userID, false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token+"&device_id="+deviceID.String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusForbidden {
		t.Errorf("expected %d, got %d", http.StatusForbidden, rec.Code)
	}
}

func TestHandler_GroupLoadFailure(t *testing.T) {
	h, jwt, deviceRepo, groupRepo := testHandler()

	userID := uuid.New()
	deviceID := uuid.New()
	deviceRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, UserID: userID, Name: "Phone"}, nil
	}
	deviceRepo.TouchLastSeenFn = func(_ context.Context, _ uuid.UUID, _ *string, _ *string) error {
		return nil
	}
	groupRepo.ListByUserIDFn = func(ctx context.Context, uid uuid.UUID) ([]model.Group, []int, error) {
		return nil, nil, model.ErrNotFound("internal error")
	}

	token, _ := jwt.GenerateAccessToken(userID, false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token+"&device_id="+deviceID.String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}

func TestHandler_UserLoadFailure(t *testing.T) {
	h, jwt, deviceRepo, groupRepo := testHandler()

	userID := uuid.New()
	deviceID := uuid.New()
	deviceRepo.GetByIDFn = func(ctx context.Context, id uuid.UUID) (*model.Device, error) {
		return &model.Device{ID: deviceID, UserID: userID, Name: "Phone"}, nil
	}
	deviceRepo.TouchLastSeenFn = func(_ context.Context, _ uuid.UUID, _ *string, _ *string) error {
		return nil
	}
	groupRepo.ListByUserIDFn = func(ctx context.Context, uid uuid.UUID) ([]model.Group, []int, error) {
		return []model.Group{{ID: uuid.New(), Name: "G1"}}, []int{3}, nil
	}
	// Override the hub's userRepo to fail
	h.hub.userRepo = &mock.UserRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.User, error) {
			return nil, model.ErrNotFound("user")
		},
	}

	token, _ := jwt.GenerateAccessToken(userID, false)
	req := httptest.NewRequest("GET", "/api/v1/ws?token="+token+"&device_id="+deviceID.String(), nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	if rec.Code != http.StatusInternalServerError {
		t.Errorf("expected %d, got %d", http.StatusInternalServerError, rec.Code)
	}
}
