package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestDrawings_Create(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "drawuser1", "draw1@test.local", "Password123!", false)
	userTokens := e.Login(t, "drawuser1", "Password123!")

	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[151.2093,-33.8688]},"properties":{}}`)
	resp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "Test Drawing",
		"geojson": geojson,
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var drawing struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&drawing); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if drawing.Name != "Test Drawing" {
		t.Errorf("name = %q, want %q", drawing.Name, "Test Drawing")
	}
}

func TestDrawings_ListOwn(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "drawlistuser", "drawlist@test.local", "Password123!", false)
	userTokens := e.Login(t, "drawlistuser", "Password123!")

	// Create a drawing
	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{}}`)
	createResp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "My Drawing",
		"geojson": geojson,
	}, userTokens.AccessToken)
	createResp.Body.Close()

	// List own drawings
	resp := e.DoJSON(t, "GET", "/api/v1/drawings", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestDrawings_Update(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "drawupdateuser", "drawupdate@test.local", "Password123!", false)
	userTokens := e.Login(t, "drawupdateuser", "Password123!")

	// Create
	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{}}`)
	createResp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "Original Name",
		"geojson": geojson,
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Update
	resp := e.DoJSON(t, "PUT", "/api/v1/drawings/"+created.ID, map[string]any{
		"name": "Updated Name",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestDrawings_Delete(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "drawdeluser", "drawdel@test.local", "Password123!", false)
	userTokens := e.Login(t, "drawdeluser", "Password123!")

	// Create
	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{}}`)
	createResp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "Delete Me",
		"geojson": geojson,
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Delete
	resp := e.DoJSON(t, "DELETE", "/api/v1/drawings/"+created.ID, nil, userTokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete drawing status = %d", resp.StatusCode)
	}
}

func TestDrawings_ShareToGroup(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, adminTokens.AccessToken, "drawshareuser", "drawshare@test.local", "Password123!", false)
	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Drawing Share Group")
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "drawshareuser", "Password123!")

	// Create
	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{}}`)
	createResp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "Share Me",
		"geojson": geojson,
	}, userTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Share
	resp := e.DoJSON(t, "POST", "/api/v1/drawings/"+created.ID+"/share", map[string]any{
		"group_id": groupID.String(),
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)
}

func TestDrawings_OtherUserCannotUpdate(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "drawowner", "drawowner@test.local", "Password123!", false)
	e.CreateUser(t, adminTokens.AccessToken, "drawintruder", "drawintruder@test.local", "Password123!", false)

	ownerTokens := e.Login(t, "drawowner", "Password123!")
	intruderTokens := e.Login(t, "drawintruder", "Password123!")

	// Owner creates a drawing
	geojson := json.RawMessage(`{"type":"Feature","geometry":{"type":"Point","coordinates":[0,0]},"properties":{}}`)
	createResp := e.DoJSON(t, "POST", "/api/v1/drawings", map[string]any{
		"name":    "My Drawing",
		"geojson": geojson,
	}, ownerTokens.AccessToken)
	var created struct {
		ID string `json:"id"`
	}
	json.NewDecoder(createResp.Body).Decode(&created)
	createResp.Body.Close()

	// Intruder tries to update
	resp := e.DoJSON(t, "PUT", "/api/v1/drawings/"+created.ID, map[string]any{
		"name": "Hacked Name",
	}, intruderTokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		t.Error("other user should not be able to update drawing")
	}
}
