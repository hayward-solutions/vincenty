package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/sitaware/api/internal/config"
)

func TestRateLimit_AllowsWithinBurst(t *testing.T) {
	cfg := config.RateLimitConfig{RPS: 100, Burst: 10}
	handler := RateLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// All requests within burst should succeed
	for i := 0; i < 10; i++ {
		r := httptest.NewRequest("GET", "/test", nil)
		r.RemoteAddr = "192.168.1.1:12345"
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		if w.Code != http.StatusOK {
			t.Errorf("request %d: status = %d, want %d", i, w.Code, http.StatusOK)
		}
	}
}

func TestRateLimit_BlocksExcessRequests(t *testing.T) {
	// 1 RPS with burst of 1 - second request should be blocked
	cfg := config.RateLimitConfig{RPS: 1, Burst: 1}
	handler := RateLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// First request uses the burst token
	r := httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("first request: status = %d, want %d", w.Code, http.StatusOK)
	}

	// Second request should be rate limited
	r = httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusTooManyRequests {
		t.Errorf("second request: status = %d, want %d", w.Code, http.StatusTooManyRequests)
	}

	if w.Header().Get("Retry-After") != "1" {
		t.Errorf("Retry-After = %q, want %q", w.Header().Get("Retry-After"), "1")
	}
}

func TestRateLimit_DifferentIPsIndependent(t *testing.T) {
	cfg := config.RateLimitConfig{RPS: 1, Burst: 1}
	handler := RateLimit(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	// Exhaust IP 1
	r := httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.1:12345"
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	// IP 2 should still work
	r = httptest.NewRequest("GET", "/test", nil)
	r.RemoteAddr = "10.0.0.2:12345"
	w = httptest.NewRecorder()
	handler.ServeHTTP(w, r)
	if w.Code != http.StatusOK {
		t.Errorf("different IP should not be rate limited, got %d", w.Code)
	}
}

// ---------------------------------------------------------------------------
// MaxBodySize
// ---------------------------------------------------------------------------

func TestMaxBodySize_SmallBody(t *testing.T) {
	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		buf := make([]byte, 100)
		_, err := r.Body.Read(buf)
		if err != nil && err.Error() != "EOF" {
			t.Errorf("unexpected read error: %v", err)
		}
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestMaxBodySize_NilBody(t *testing.T) {
	handler := MaxBodySize(1024)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	r := httptest.NewRequest("GET", "/test", nil)
	r.Body = nil
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}
