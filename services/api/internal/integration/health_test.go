package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestHealthz(t *testing.T) {
	e := getEnv(t)
	resp := e.Do(t, "GET", "/healthz", "", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestReadyz(t *testing.T) {
	e := getEnv(t)
	resp := e.Do(t, "GET", "/readyz", "", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode readyz: %v", err)
	}
	if body["status"] != "ready" {
		t.Errorf("readyz status = %q, want %q", body["status"], "ready")
	}
}

func TestAPIInfo(t *testing.T) {
	e := getEnv(t)
	resp := e.Do(t, "GET", "/api/v1/", "", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var body map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode api info: %v", err)
	}
	if body["service"] != "sitaware-api" {
		t.Errorf("service = %q, want %q", body["service"], "sitaware-api")
	}
	if body["version"] == "" {
		t.Error("version field must not be empty")
	}
}
