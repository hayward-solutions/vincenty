package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/vincenty/api/internal/config"
)

func TestCORS_WildcardOrigin(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"*"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Errorf("ACAO = %q, want %q", w.Header().Get("Access-Control-Allow-Origin"), "*")
	}
}

func TestCORS_AllowedOrigin(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "https://app.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("ACAO = %q, want %q", w.Header().Get("Access-Control-Allow-Origin"), "https://app.example.com")
	}
	if w.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Errorf("ACAC = %q, want %q", w.Header().Get("Access-Control-Allow-Credentials"), "true")
	}
	if w.Header().Get("Vary") != "Origin" {
		t.Errorf("Vary = %q, want %q", w.Header().Get("Vary"), "Origin")
	}
}

func TestCORS_DisallowedOrigin(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest("GET", "/test", nil)
	r.Header.Set("Origin", "https://evil.example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("ACAO should be empty for disallowed origin, got %q", w.Header().Get("Access-Control-Allow-Origin"))
	}
	// Request should still be served (200)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestCORS_OptionsPreflight(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"*"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("should not see this"))
		}),
	)

	r := httptest.NewRequest("OPTIONS", "/test", nil)
	r.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Code != http.StatusNoContent {
		t.Errorf("status = %d, want %d for OPTIONS preflight", w.Code, http.StatusNoContent)
	}
	if w.Body.Len() > 0 {
		t.Error("OPTIONS response should have no body")
	}
}

func TestCORS_CommonHeaders(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"*"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("Access-Control-Allow-Methods should be set")
	}

	headers := w.Header().Get("Access-Control-Allow-Headers")
	if headers == "" {
		t.Error("Access-Control-Allow-Headers should be set")
	}

	maxAge := w.Header().Get("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("Access-Control-Max-Age = %q, want %q", maxAge, "3600")
	}
}

func TestCORS_NoOriginHeader(t *testing.T) {
	handler := CORS(config.CORSConfig{AllowedOrigins: []string{"https://app.example.com"}})(
		http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}),
	)

	r := httptest.NewRequest("GET", "/test", nil)
	// No Origin header
	w := httptest.NewRecorder()

	handler.ServeHTTP(w, r)

	if w.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("ACAO should be empty when no Origin header is present")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, should still serve request", w.Code)
	}
}

func TestCORS_MultipleAllowedOrigins(t *testing.T) {
	cfg := config.CORSConfig{AllowedOrigins: []string{"https://a.example.com", "https://b.example.com"}}
	handler := CORS(cfg)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))

	tests := []struct {
		origin   string
		expected string
	}{
		{"https://a.example.com", "https://a.example.com"},
		{"https://b.example.com", "https://b.example.com"},
		{"https://c.example.com", ""},
	}

	for _, tt := range tests {
		r := httptest.NewRequest("GET", "/test", nil)
		r.Header.Set("Origin", tt.origin)
		w := httptest.NewRecorder()
		handler.ServeHTTP(w, r)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got != tt.expected {
			t.Errorf("Origin %q: ACAO = %q, want %q", tt.origin, got, tt.expected)
		}
	}
}
