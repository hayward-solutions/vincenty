package ws

import (
	"context"
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
	h := NewHandler(hub, jwt, deviceRepo, groupRepo)
	return h, jwt, deviceRepo, groupRepo
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
