package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestServerSettings_AdminGet(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/server/settings", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var settings struct {
		MFARequired       bool   `json:"mfa_required"`
		MapboxAccessToken string `json:"mapbox_access_token"`
		GoogleMapsApiKey  string `json:"google_maps_api_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatalf("decode: %v", err)
	}
}

func TestServerSettings_AdminUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	mfaRequired := true
	resp := e.DoJSON(t, "PUT", "/api/v1/server/settings", map[string]any{
		"mfa_required": mfaRequired,
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	// Reset
	mfaRequired = false
	resetResp := e.DoJSON(t, "PUT", "/api/v1/server/settings", map[string]any{
		"mfa_required": mfaRequired,
	}, tokens.AccessToken)
	resetResp.Body.Close()
}

func TestServerSettings_NonAdminCannotGet(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "settingsreguser", "settingsreg@test.local", "Password123!", false)
	userTokens := e.Login(t, "settingsreguser", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/server/settings", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestServerSettings_NonAdminCannotUpdate(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "settingsreguser2", "settingsreg2@test.local", "Password123!", false)
	userTokens := e.Login(t, "settingsreguser2", "Password123!")

	mfaRequired := true
	resp := e.DoJSON(t, "PUT", "/api/v1/server/settings", map[string]any{
		"mfa_required": mfaRequired,
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestServerSettings_GetAfterUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Update MFA to true
	mfaRequired := true
	updateResp := e.DoJSON(t, "PUT", "/api/v1/server/settings", map[string]any{
		"mfa_required": mfaRequired,
	}, tokens.AccessToken)
	testutil.RequireStatus(t, updateResp, http.StatusOK)
	updateResp.Body.Close()

	// Get and verify the response is a single object
	resp := e.DoJSON(t, "GET", "/api/v1/server/settings", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var settings struct {
		MFARequired       bool   `json:"mfa_required"`
		MapboxAccessToken string `json:"mapbox_access_token"`
		GoogleMapsApiKey  string `json:"google_maps_api_key"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&settings); err != nil {
		t.Fatalf("decode: %v", err)
	}

	if !settings.MFARequired {
		t.Error("expected mfa_required to be true after update")
	}

	// Reset it back to avoid affecting other tests
	mfaRequired = false
	resetResp := e.DoJSON(t, "PUT", "/api/v1/server/settings", map[string]any{
		"mfa_required": mfaRequired,
	}, tokens.AccessToken)
	resetResp.Body.Close()
}
