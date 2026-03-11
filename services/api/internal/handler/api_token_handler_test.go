package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/middleware"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository/mock"
	"github.com/vincenty/api/internal/service"
)

// testAPITokenHandler returns a handler backed by a mock repo.
func testAPITokenHandler() (*APITokenHandler, *mock.APITokenRepo) {
	repo := &mock.APITokenRepo{
		TouchLastUsedFn: func(_ context.Context, _ uuid.UUID) error { return nil },
	}
	svc := service.NewAPITokenService(repo)
	h := NewAPITokenHandler(svc)
	return h, repo
}

// apiTokenRequest creates an authenticated request with claims injected.
func apiTokenRequest(method, path string, body any, userID uuid.UUID) *http.Request {
	var buf bytes.Buffer
	if body != nil {
		json.NewEncoder(&buf).Encode(body)
	}
	r := httptest.NewRequest(method, path, &buf)
	r.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: userID, IsAdmin: false}
	ctx := middleware.ContextWithClaims(r.Context(), claims)
	return r.WithContext(ctx)
}

// ---------------------------------------------------------------------------
// Create
// ---------------------------------------------------------------------------

func TestAPITokenHandler_Create_Success(t *testing.T) {
	h, repo := testAPITokenHandler()
	repo.CreateFn = func(_ context.Context, token *model.APIToken) error {
		token.ID = uuid.New()
		token.CreatedAt = time.Now()
		return nil
	}

	r := apiTokenRequest("POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": "my-token"}, uuid.New())
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}

	var resp model.CreateAPITokenResponse
	json.NewDecoder(w.Body).Decode(&resp)
	if resp.Token == "" {
		t.Error("token should be present in response")
	}
	if resp.Name != "my-token" {
		t.Errorf("name = %q", resp.Name)
	}
}

func TestAPITokenHandler_Create_BadJSON(t *testing.T) {
	h, _ := testAPITokenHandler()

	r := httptest.NewRequest("POST", "/api/v1/users/me/api-tokens",
		bytes.NewBufferString("{invalid"))
	r.Header.Set("Content-Type", "application/json")

	claims := &auth.Claims{UserID: uuid.New()}
	ctx := middleware.ContextWithClaims(r.Context(), claims)
	r = r.WithContext(ctx)

	w := httptest.NewRecorder()
	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPITokenHandler_Create_MissingClaims(t *testing.T) {
	h, _ := testAPITokenHandler()

	r := httptest.NewRequest("POST", "/api/v1/users/me/api-tokens",
		bytes.NewBufferString(`{"name":"test"}`))
	r.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAPITokenHandler_Create_ValidationError(t *testing.T) {
	h, _ := testAPITokenHandler()

	r := apiTokenRequest("POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": ""}, uuid.New())
	w := httptest.NewRecorder()

	h.Create(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// ---------------------------------------------------------------------------
// List
// ---------------------------------------------------------------------------

func TestAPITokenHandler_List_Success(t *testing.T) {
	h, repo := testAPITokenHandler()
	repo.ListByUserIDFn = func(_ context.Context, _ uuid.UUID) ([]model.APIToken, error) {
		return []model.APIToken{
			{ID: uuid.New(), Name: "t1", CreatedAt: time.Now()},
		}, nil
	}

	r := apiTokenRequest("GET", "/api/v1/users/me/api-tokens", nil, uuid.New())
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}

	var tokens []model.APITokenResponse
	json.NewDecoder(w.Body).Decode(&tokens)
	if len(tokens) != 1 {
		t.Errorf("len = %d, want 1", len(tokens))
	}
}

func TestAPITokenHandler_List_Error(t *testing.T) {
	h, repo := testAPITokenHandler()
	repo.ListByUserIDFn = func(_ context.Context, _ uuid.UUID) ([]model.APIToken, error) {
		return nil, errors.New("db error")
	}

	r := apiTokenRequest("GET", "/api/v1/users/me/api-tokens", nil, uuid.New())
	w := httptest.NewRecorder()

	h.List(w, r)

	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// ---------------------------------------------------------------------------
// Delete
// ---------------------------------------------------------------------------

func TestAPITokenHandler_Delete_Success(t *testing.T) {
	h, repo := testAPITokenHandler()
	repo.DeleteFn = func(_ context.Context, _, _ uuid.UUID) error {
		return nil
	}

	tokenID := uuid.New()
	r := apiTokenRequest("DELETE", "/api/v1/users/me/api-tokens/"+tokenID.String(), nil, uuid.New())
	r.SetPathValue("id", tokenID.String())
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNoContent)
	}
}

func TestAPITokenHandler_Delete_InvalidID(t *testing.T) {
	h, _ := testAPITokenHandler()

	r := apiTokenRequest("DELETE", "/api/v1/users/me/api-tokens/not-a-uuid", nil, uuid.New())
	r.SetPathValue("id", "not-a-uuid")
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestAPITokenHandler_Delete_NotFound(t *testing.T) {
	h, repo := testAPITokenHandler()
	repo.DeleteFn = func(_ context.Context, _, _ uuid.UUID) error {
		return model.ErrNotFound("api token")
	}

	tokenID := uuid.New()
	r := apiTokenRequest("DELETE", "/api/v1/users/me/api-tokens/"+tokenID.String(), nil, uuid.New())
	r.SetPathValue("id", tokenID.String())
	w := httptest.NewRecorder()

	h.Delete(w, r)

	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}
