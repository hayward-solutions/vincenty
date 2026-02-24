package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestMapConfigs_GetSettings(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mapsettingsuser", "mapsettings@test.local", "Password123!", false)
	userTokens := e.Login(t, "mapsettingsuser", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/map/settings", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestMapConfigs_AdminList(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/map-configs", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestMapConfigs_AdminCreate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	tileURL := "https://tile.openstreetmap.org/{z}/{x}/{y}.png"
	resp := e.DoJSON(t, "POST", "/api/v1/map-configs", map[string]any{
		"name":        "Custom Map",
		"source_type": "remote",
		"tile_url":    tileURL,
		"min_zoom":    0,
		"max_zoom":    18,
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var cfg struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&cfg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if cfg.Name != "Custom Map" {
		t.Errorf("name = %q, want %q", cfg.Name, "Custom Map")
	}
}

func TestMapConfigs_AdminUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create
	tileURL := "https://tile.example.com/{z}/{x}/{y}.png"
	createResp := e.DoJSON(t, "POST", "/api/v1/map-configs", map[string]any{
		"name":        "Update Map",
		"source_type": "remote",
		"tile_url":    tileURL,
		"min_zoom":    0,
		"max_zoom":    18,
	}, tokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Update
	resp := e.DoJSON(t, "PUT", "/api/v1/map-configs/"+created.ID, map[string]any{
		"name": "Updated Map Name",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestMapConfigs_AdminDelete(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create
	tileURL := "https://tile.example.com/{z}/{x}/{y}.png"
	createResp := e.DoJSON(t, "POST", "/api/v1/map-configs", map[string]any{
		"name":        "Delete Map",
		"source_type": "remote",
		"tile_url":    tileURL,
		"min_zoom":    0,
		"max_zoom":    18,
	}, tokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Delete
	resp := e.DoJSON(t, "DELETE", "/api/v1/map-configs/"+created.ID, nil, tokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete map config status = %d", resp.StatusCode)
	}
}

func TestMapConfigs_NonAdminCannotCreate(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mapreguser", "mapreg@test.local", "Password123!", false)
	userTokens := e.Login(t, "mapreguser", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/map-configs", map[string]any{
		"name":        "Unauthorized Map",
		"source_type": "remote",
		"tile_url":    "https://example.com/{z}/{x}/{y}.png",
		"min_zoom":    0,
		"max_zoom":    18,
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}
