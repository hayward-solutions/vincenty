package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestDevices_Create(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devuser1", "dev1@test.local", "Password123!", false)
	userTokens := e.Login(t, "devuser1", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "My Phone",
		"type": "atak",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var device struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&device); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if device.Name != "My Phone" {
		t.Errorf("name = %q, want %q", device.Name, "My Phone")
	}
}

func TestDevices_List(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devlistuser", "devlist@test.local", "Password123!", false)
	userTokens := e.Login(t, "devlistuser", "Password123!")

	// Create a device
	createResp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "Device List Test",
		"type": "atak",
	}, userTokens.AccessToken)
	createResp.Body.Close()

	// List
	resp := e.DoJSON(t, "GET", "/api/v1/users/me/devices", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var devices []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&devices); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(devices) < 1 {
		t.Error("expected at least 1 device")
	}
}

func TestDevices_Update(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devupdateuser", "devupdate@test.local", "Password123!", false)
	userTokens := e.Login(t, "devupdateuser", "Password123!")

	// Create a device
	createResp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "Old Name",
		"type": "atak",
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Update
	resp := e.DoJSON(t, "PUT", "/api/v1/devices/"+created.ID, map[string]any{
		"name": "New Name",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestDevices_Delete(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devdeluser", "devdel@test.local", "Password123!", false)
	userTokens := e.Login(t, "devdeluser", "Password123!")

	// Create a device
	createResp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "Delete Me",
		"type": "atak",
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Delete
	resp := e.DoJSON(t, "DELETE", "/api/v1/devices/"+created.ID, nil, userTokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete device status = %d", resp.StatusCode)
	}
}

func TestDevices_SetPrimary(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devprimaryuser", "devprimary@test.local", "Password123!", false)
	userTokens := e.Login(t, "devprimaryuser", "Password123!")

	// Create a device
	createResp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "Primary Device",
		"type": "atak",
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Set primary
	resp := e.DoJSON(t, "PUT", "/api/v1/users/me/devices/"+created.ID+"/primary", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestDevices_Resolve(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "devresolveuser", "devresolve@test.local", "Password123!", false)
	userTokens := e.Login(t, "devresolveuser", "Password123!")

	// Create a device with a UID
	createResp := e.DoJSON(t, "POST", "/api/v1/users/me/devices", map[string]any{
		"name": "Resolve Device",
		"type": "atak",
		"uid":  "ANDROID-resolve-test-123",
	}, userTokens.AccessToken)
	createResp.Body.Close()

	// Resolve by UID
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/devices/resolve", map[string]any{
		"uid": "ANDROID-resolve-test-123",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}
