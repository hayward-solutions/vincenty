package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/vincenty/api/internal/testutil"
)

func TestTerrainConfigs_AdminList(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/terrain-configs", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestTerrainConfigs_AdminCreate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "POST", "/api/v1/terrain-configs", map[string]any{
		"name":        "Custom Terrain",
		"source_type": "remote",
		"terrain_url": "https://terrain.example.com/tiles/{z}/{x}/{y}.terrain",
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
	if cfg.Name != "Custom Terrain" {
		t.Errorf("name = %q, want %q", cfg.Name, "Custom Terrain")
	}
}

func TestTerrainConfigs_AdminUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create
	createResp := e.DoJSON(t, "POST", "/api/v1/terrain-configs", map[string]any{
		"name":        "Update Terrain",
		"source_type": "remote",
		"terrain_url": "https://terrain.example.com/tiles/{z}/{x}/{y}.terrain",
	}, tokens.AccessToken)
	testutil.RequireStatus(t, createResp, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	if created.ID == "" {
		t.Fatal("created terrain config has empty ID")
	}

	// Update
	resp := e.DoJSON(t, "PUT", "/api/v1/terrain-configs/"+created.ID, map[string]any{
		"name": "Updated Terrain Name",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestTerrainConfigs_AdminDelete(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create
	createResp := e.DoJSON(t, "POST", "/api/v1/terrain-configs", map[string]any{
		"name":        "Delete Terrain",
		"source_type": "remote",
		"terrain_url": "https://terrain.example.com/tiles/{z}/{x}/{y}.terrain",
	}, tokens.AccessToken)
	testutil.RequireStatus(t, createResp, http.StatusCreated)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	if created.ID == "" {
		t.Fatal("created terrain config has empty ID")
	}

	// Delete
	resp := e.DoJSON(t, "DELETE", "/api/v1/terrain-configs/"+created.ID, nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusNoContent)
}

func TestTerrainConfigs_NonAdminCannotCreate(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "terrainreguser", "terrainreg@test.local", "Password123!", false)
	userTokens := e.Login(t, "terrainreguser", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/terrain-configs", map[string]any{
		"name":        "Unauthorized Terrain",
		"source_type": "remote",
		"terrain_url": "https://example.com/terrain",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}
