package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/config"
)

func newTestJWT() *auth.JWTService {
	return auth.NewJWTService(config.JWTConfig{
		Secret:          "test-secret-at-least-32-bytes-long!!",
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 168 * time.Hour,
	})
}

func newTestAuth() *Auth {
	return NewAuth(newTestJWT(), nil)
}

func validToken(userID uuid.UUID, isAdmin bool) string {
	token, _ := newTestJWT().GenerateAccessToken(userID, isAdmin)
	return token
}

func expiredToken() string {
	svc := auth.NewJWTService(config.JWTConfig{
		Secret:          "test-secret-at-least-32-bytes-long!!",
		AccessTokenTTL:  -1 * time.Hour,
		RefreshTokenTTL: 168 * time.Hour,
	})
	token, _ := svc.GenerateAccessToken(uuid.New(), false)
	return token
}

// dummyHandler is a handler that records whether it was called.
func dummyHandler() (http.Handler, *bool) {
	called := false
	h := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	return h, &called
}

// ---------------------------------------------------------------------------
// Authenticate
// ---------------------------------------------------------------------------

func TestAuthenticate_ValidToken(t *testing.T) {
	authMW := newTestAuth()
	userID := uuid.New()
	next, called := dummyHandler()

	handler := authMW.Authenticate(next)

	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(userID, true))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !*called {
		t.Error("next handler was not called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthenticate_MissingHeader(t *testing.T) {
	authMW := newTestAuth()
	next, called := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if *called {
		t.Error("next handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_InvalidFormat(t *testing.T) {
	authMW := newTestAuth()
	next, _ := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Token abc123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_ExpiredToken(t *testing.T) {
	authMW := newTestAuth()
	next, called := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer "+expiredToken())
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if *called {
		t.Error("next handler should not be called for expired token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_MalformedToken(t *testing.T) {
	authMW := newTestAuth()
	next, _ := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer invalid.jwt.token")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticate_InjectsClaimsInContext(t *testing.T) {
	authMW := newTestAuth()
	userID := uuid.New()

	var capturedClaims *auth.Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if ok {
			capturedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(userID, true))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if capturedClaims == nil {
		t.Fatal("claims not found in context")
	}
	if capturedClaims.UserID != userID {
		t.Errorf("claims.UserID = %v, want %v", capturedClaims.UserID, userID)
	}
	if !capturedClaims.IsAdmin {
		t.Error("claims.IsAdmin should be true")
	}
}

// ---------------------------------------------------------------------------
// RequireAdmin
// ---------------------------------------------------------------------------

func TestRequireAdmin_AdminAllowed(t *testing.T) {
	authMW := newTestAuth()
	next, called := dummyHandler()

	handler := authMW.RequireAdmin(next)
	r := httptest.NewRequest("GET", "/admin", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), true))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !*called {
		t.Error("next handler should be called for admin")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestRequireAdmin_NonAdminRejected(t *testing.T) {
	authMW := newTestAuth()
	next, called := dummyHandler()

	handler := authMW.RequireAdmin(next)
	r := httptest.NewRequest("GET", "/admin", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), false))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if *called {
		t.Error("next handler should not be called for non-admin")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAdmin_NoToken(t *testing.T) {
	authMW := newTestAuth()
	next, _ := dummyHandler()

	handler := authMW.RequireAdmin(next)
	r := httptest.NewRequest("GET", "/admin", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ---------------------------------------------------------------------------
// AuthenticateWithQueryToken
// ---------------------------------------------------------------------------

func TestAuthenticateWithQueryToken_HeaderPreferred(t *testing.T) {
	authMW := newTestAuth()
	userID := uuid.New()

	var capturedClaims *auth.Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, _ := ClaimsFromContext(r.Context())
		capturedClaims = claims
		w.WriteHeader(http.StatusOK)
	})

	handler := authMW.AuthenticateWithQueryToken(next)
	r := httptest.NewRequest("GET", "/test?token=invalid", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(userID, false))
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedClaims == nil || capturedClaims.UserID != userID {
		t.Error("should use header token when both are present")
	}
}

func TestAuthenticateWithQueryToken_FallbackToQuery(t *testing.T) {
	authMW := newTestAuth()
	userID := uuid.New()
	next, called := dummyHandler()

	handler := authMW.AuthenticateWithQueryToken(next)
	r := httptest.NewRequest("GET", "/test?token="+validToken(userID, false), nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !*called {
		t.Error("next handler should be called with valid query token")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthenticateWithQueryToken_NoToken(t *testing.T) {
	authMW := newTestAuth()
	next, _ := dummyHandler()

	handler := authMW.AuthenticateWithQueryToken(next)
	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ---------------------------------------------------------------------------
// RequireMFASetup
// ---------------------------------------------------------------------------

type mockMFAChecker struct {
	required bool
}

func (m *mockMFAChecker) IsMFARequired(_ context.Context) bool {
	return m.required
}

func TestRequireMFASetup_NotRequired(t *testing.T) {
	authMW := newTestAuth()
	checker := &mockMFAChecker{required: false}
	getUserMFA := func(_ context.Context, _ uuid.UUID) (bool, error) {
		return false, nil
	}

	mw := authMW.RequireMFASetup(checker, getUserMFA)
	next, called := dummyHandler()

	handler := mw(next)
	r := httptest.NewRequest("GET", "/api/v1/some-endpoint", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), false))
	w := httptest.NewRecorder()

	// Need to run through Authenticate first to set claims
	authMW.Authenticate(handler).ServeHTTP(w, r)

	if !*called {
		t.Error("should be called when MFA not required")
	}
}

func TestRequireMFASetup_RequiredUserHasMFA(t *testing.T) {
	authMW := newTestAuth()
	checker := &mockMFAChecker{required: true}
	getUserMFA := func(_ context.Context, _ uuid.UUID) (bool, error) {
		return true, nil
	}

	mw := authMW.RequireMFASetup(checker, getUserMFA)
	next, called := dummyHandler()

	handler := mw(next)
	r := httptest.NewRequest("GET", "/api/v1/some-endpoint", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), false))
	w := httptest.NewRecorder()

	authMW.Authenticate(handler).ServeHTTP(w, r)

	if !*called {
		t.Error("should be called when user has MFA configured")
	}
}

func TestRequireMFASetup_RequiredUserLacksMFA(t *testing.T) {
	authMW := newTestAuth()
	checker := &mockMFAChecker{required: true}
	getUserMFA := func(_ context.Context, _ uuid.UUID) (bool, error) {
		return false, nil
	}

	mw := authMW.RequireMFASetup(checker, getUserMFA)
	next, called := dummyHandler()

	handler := mw(next)
	r := httptest.NewRequest("GET", "/api/v1/some-endpoint", nil)
	r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), false))
	w := httptest.NewRecorder()

	authMW.Authenticate(handler).ServeHTTP(w, r)

	if *called {
		t.Error("should NOT be called when user lacks MFA")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}

	var body map[string]map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"]["code"] != "mfa_setup_required" {
		t.Errorf("error code = %q", body["error"]["code"])
	}
}

func TestRequireMFASetup_BypassMFASetupEndpoints(t *testing.T) {
	authMW := newTestAuth()
	checker := &mockMFAChecker{required: true}
	getUserMFA := func(_ context.Context, _ uuid.UUID) (bool, error) {
		return false, nil // user lacks MFA
	}

	mw := authMW.RequireMFASetup(checker, getUserMFA)
	next, called := dummyHandler()

	handler := mw(next)

	// MFA setup endpoints should be allowed through
	paths := []string{
		"/api/v1/users/me/mfa/totp/setup",
		"/api/v1/users/me/mfa/methods",
		"/api/v1/users/me",
		"/api/v1/server/settings",
	}

	for _, path := range paths {
		*called = false
		r := httptest.NewRequest("GET", path, nil)
		r.Header.Set("Authorization", "Bearer "+validToken(uuid.New(), false))
		w := httptest.NewRecorder()

		authMW.Authenticate(handler).ServeHTTP(w, r)

		if !*called {
			t.Errorf("path %q should bypass MFA enforcement", path)
		}
	}
}

// ---------------------------------------------------------------------------
// API Token authentication
// ---------------------------------------------------------------------------

// mockTokenValidator implements TokenValidator for testing.
type mockTokenValidator struct {
	validateFn func(ctx context.Context, raw string) (*auth.Claims, error)
}

func (m *mockTokenValidator) ValidateToken(ctx context.Context, raw string) (*auth.Claims, error) {
	return m.validateFn(ctx, raw)
}

func TestAuthenticate_APIToken_Success(t *testing.T) {
	userID := uuid.New()
	tv := &mockTokenValidator{
		validateFn: func(_ context.Context, raw string) (*auth.Claims, error) {
			if raw != "sat_testtoken123" {
				t.Errorf("unexpected token: %q", raw)
			}
			return &auth.Claims{UserID: userID, IsAdmin: true}, nil
		},
	}
	authMW := NewAuth(newTestJWT(), tv)

	var capturedClaims *auth.Claims
	next := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		claims, ok := ClaimsFromContext(r.Context())
		if ok {
			capturedClaims = claims
		}
		w.WriteHeader(http.StatusOK)
	})

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer sat_testtoken123")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if capturedClaims == nil {
		t.Fatal("claims not found in context")
	}
	if capturedClaims.UserID != userID {
		t.Errorf("UserID = %v, want %v", capturedClaims.UserID, userID)
	}
	if !capturedClaims.IsAdmin {
		t.Error("IsAdmin should be true")
	}
}

func TestAuthenticate_APIToken_Invalid(t *testing.T) {
	tv := &mockTokenValidator{
		validateFn: func(_ context.Context, _ string) (*auth.Claims, error) {
			return nil, errors.New("invalid api token")
		},
	}
	authMW := NewAuth(newTestJWT(), tv)
	next, called := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer sat_invalidtoken")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if *called {
		t.Error("next handler should not be called for invalid API token")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestAuthenticateWithQueryToken_APIToken(t *testing.T) {
	userID := uuid.New()
	tv := &mockTokenValidator{
		validateFn: func(_ context.Context, raw string) (*auth.Claims, error) {
			return &auth.Claims{UserID: userID, IsAdmin: false}, nil
		},
	}
	authMW := NewAuth(newTestJWT(), tv)
	next, called := dummyHandler()

	handler := authMW.AuthenticateWithQueryToken(next)
	r := httptest.NewRequest("GET", "/test?token=sat_querytoken", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if !*called {
		t.Error("next handler should be called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestAuthenticate_NilTokenValidator_SatPrefixFallsToJWT(t *testing.T) {
	// When tokenVal is nil, sat_ tokens fall through to JWT validation
	// and fail (since they're not valid JWTs).
	authMW := NewAuth(newTestJWT(), nil)
	next, called := dummyHandler()

	handler := authMW.Authenticate(next)
	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Authorization", "Bearer sat_novalidator")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if *called {
		t.Error("next handler should not be called")
	}
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// ---------------------------------------------------------------------------
// ClaimsFromContext
// ---------------------------------------------------------------------------

func TestClaimsFromContext_NoClaimsReturnsNotOK(t *testing.T) {
	ctx := context.Background()
	_, ok := ClaimsFromContext(ctx)
	if ok {
		t.Error("should return ok=false for empty context")
	}
}

// ---------------------------------------------------------------------------
// writeError
// ---------------------------------------------------------------------------

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusUnauthorized, "unauthorized", "invalid token")

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	if w.Header().Get("Content-Type") != "application/json" {
		t.Errorf("Content-Type = %q", w.Header().Get("Content-Type"))
	}

	var body map[string]map[string]string
	json.Unmarshal(w.Body.Bytes(), &body)
	if body["error"]["code"] != "unauthorized" {
		t.Errorf("error.code = %q", body["error"]["code"])
	}
}
