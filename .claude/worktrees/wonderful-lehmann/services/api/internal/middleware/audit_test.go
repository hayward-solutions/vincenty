package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
)

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name   string
		header string
		want   string
	}{
		{"valid bearer", "Bearer abc123", "abc123"},
		{"lowercase bearer", "bearer abc123", "abc123"},
		{"BEARER uppercase", "BEARER abc123", "abc123"},
		{"missing header", "", ""},
		{"no bearer prefix", "Token abc123", ""},
		{"bearer no space", "Bearerabc123", ""},
		{"only bearer", "Bearer", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			if tt.header != "" {
				req.Header.Set("Authorization", tt.header)
			}
			got := extractBearerToken(req)
			if got != tt.want {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractIDFromBody(t *testing.T) {
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"id": id.String()})

	got := extractIDFromBody(body)
	if got != id {
		t.Errorf("extractIDFromBody() = %v, want %v", got, id)
	}
}

func TestExtractIDFromBody_NoID(t *testing.T) {
	body := []byte(`{"name":"test"}`)
	got := extractIDFromBody(body)
	if got != uuid.Nil {
		t.Errorf("extractIDFromBody() = %v, want Nil", got)
	}
}

func TestExtractIDFromBody_InvalidJSON(t *testing.T) {
	body := []byte(`not json`)
	got := extractIDFromBody(body)
	if got != uuid.Nil {
		t.Errorf("extractIDFromBody() = %v, want Nil", got)
	}
}

func TestExtractUserIDFromBody_Nested(t *testing.T) {
	id := uuid.New()
	body, _ := json.Marshal(map[string]any{
		"user": map[string]string{"id": id.String()},
	})

	got := extractUserIDFromBody(body)
	if got != id {
		t.Errorf("extractUserIDFromBody() = %v, want %v", got, id)
	}
}

func TestExtractUserIDFromBody_Flat(t *testing.T) {
	id := uuid.New()
	body, _ := json.Marshal(map[string]string{"user_id": id.String()})

	got := extractUserIDFromBody(body)
	if got != id {
		t.Errorf("extractUserIDFromBody() = %v, want %v", got, id)
	}
}

func TestExtractUserIDFromBody_NoUserID(t *testing.T) {
	body := []byte(`{"status":"ok"}`)
	got := extractUserIDFromBody(body)
	if got != uuid.Nil {
		t.Errorf("extractUserIDFromBody() = %v, want Nil", got)
	}
}

func TestExtractUserIDFromBody_InvalidJSON(t *testing.T) {
	body := []byte(`invalid`)
	got := extractUserIDFromBody(body)
	if got != uuid.Nil {
		t.Errorf("extractUserIDFromBody() = %v, want Nil", got)
	}
}

func TestExtractIP(t *testing.T) {
	tests := []struct {
		name       string
		xff        string
		xri        string
		remoteAddr string
		want       string
	}{
		{"XFF single", "1.2.3.4", "", "5.6.7.8:1234", "1.2.3.4"},
		{"XFF multiple", "1.2.3.4, 10.0.0.1, 192.168.1.1", "", "5.6.7.8:1234", "1.2.3.4"},
		{"X-Real-IP", "", "9.8.7.6", "5.6.7.8:1234", "9.8.7.6"},
		{"RemoteAddr with port", "", "", "5.6.7.8:1234", "5.6.7.8"},
		{"RemoteAddr no port", "", "", "5.6.7.8", "5.6.7.8"},
		{"XFF takes priority over XRI", "1.2.3.4", "9.8.7.6", "5.6.7.8:1234", "1.2.3.4"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodGet, "/", nil)
			req.RemoteAddr = tt.remoteAddr
			if tt.xff != "" {
				req.Header.Set("X-Forwarded-For", tt.xff)
			}
			if tt.xri != "" {
				req.Header.Set("X-Real-IP", tt.xri)
			}
			got := ExtractIP(req)
			if got != tt.want {
				t.Errorf("ExtractIP() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestExtractGroupID_PathParam(t *testing.T) {
	// Create a request using a mux that sets path values
	mux := http.NewServeMux()
	var result *uuid.UUID
	groupID := uuid.New()

	mux.HandleFunc("GET /test/{id}", func(w http.ResponseWriter, r *http.Request) {
		result = extractGroupID(r, "path:id")
	})

	req := httptest.NewRequest(http.MethodGet, "/test/"+groupID.String(), nil)
	rec := httptest.NewRecorder()
	mux.ServeHTTP(rec, req)

	if result == nil || *result != groupID {
		t.Errorf("extractGroupID(path:id) = %v, want %v", result, groupID)
	}
}

func TestExtractGroupID_EmptySource(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result := extractGroupID(req, "")
	if result != nil {
		t.Errorf("expected nil for empty source, got %v", result)
	}
}

func TestExtractGroupID_InvalidSource(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/", nil)
	result := extractGroupID(req, "invalid")
	if result != nil {
		t.Errorf("expected nil for invalid source, got %v", result)
	}
}

func TestExtractGroupID_FormValue(t *testing.T) {
	groupID := uuid.New()
	body := "group_id=" + groupID.String()
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result := extractGroupID(req, "form:group_id")
	if result == nil || *result != groupID {
		t.Errorf("extractGroupID(form:group_id) = %v, want %v", result, groupID)
	}
}

func TestExtractGroupID_InvalidUUID(t *testing.T) {
	body := "group_id=not-a-uuid"
	req := httptest.NewRequest(http.MethodPost, "/", bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	result := extractGroupID(req, "form:group_id")
	if result != nil {
		t.Errorf("expected nil for invalid UUID, got %v", result)
	}
}

func TestResponseCapture_WriteHeader(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec, status: http.StatusOK}

	rc.WriteHeader(http.StatusCreated)

	if rc.status != http.StatusCreated {
		t.Errorf("status = %d, want %d", rc.status, http.StatusCreated)
	}
}

func TestResponseCapture_Write_CapturesBody(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec, status: http.StatusOK, captureBody: true}

	rc.Write([]byte("hello"))
	rc.Write([]byte(" world"))

	if rc.body.String() != "hello world" {
		t.Errorf("captured body = %q, want %q", rc.body.String(), "hello world")
	}
	if rec.Body.String() != "hello world" {
		t.Errorf("actual body = %q, want %q", rec.Body.String(), "hello world")
	}
}

func TestResponseCapture_Write_NoCaptureBody(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec, status: http.StatusOK, captureBody: false}

	rc.Write([]byte("hello"))

	if rc.body.Len() != 0 {
		t.Errorf("captured body should be empty, got %q", rc.body.String())
	}
	if rec.Body.String() != "hello" {
		t.Errorf("actual body = %q, want %q", rec.Body.String(), "hello")
	}
}

func TestResponseCapture_Hijack_NotSupported(t *testing.T) {
	rec := httptest.NewRecorder()
	rc := &responseCapture{ResponseWriter: rec}

	_, _, err := rc.Hijack()
	if err == nil {
		t.Error("expected error from Hijack on non-hijackable ResponseWriter")
	}
}

func TestAuditRoutes_ContainExpectedRoutes(t *testing.T) {
	// Verify key audit routes are configured
	expected := []routeKey{
		{"POST", "/api/v1/auth/login"},
		{"POST", "/api/v1/auth/logout"},
		{"POST", "/api/v1/users"},
		{"PUT", "/api/v1/users/{id}"},
		{"DELETE", "/api/v1/users/{id}"},
		{"POST", "/api/v1/groups"},
		{"POST", "/api/v1/messages"},
		{"POST", "/api/v1/cot/events"},
	}

	for _, key := range expected {
		if _, ok := auditRoutes[key]; !ok {
			t.Errorf("missing audit route: %s %s", key.Method, key.Pattern)
		}
	}
}

func TestAuditRoutes_ActionsNonEmpty(t *testing.T) {
	for key, action := range auditRoutes {
		if action.Action == "" {
			t.Errorf("route %s %s has empty action", key.Method, key.Pattern)
		}
		if action.ResourceType == "" {
			t.Errorf("route %s %s has empty resource type", key.Method, key.Pattern)
		}
	}
}
