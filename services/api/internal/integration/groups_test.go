package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestGroups_AdminCreate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, tokens.AccessToken, "Alpha Team")
	if groupID.String() == "" {
		t.Error("group ID is empty")
	}
}

func TestGroups_AdminList(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)
	e.CreateGroup(t, tokens.AccessToken, "List Team")

	resp := e.DoJSON(t, "GET", "/api/v1/groups?page=1&page_size=50", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var body struct {
		Data  []json.RawMessage `json:"data"`
		Total int               `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if body.Total < 1 {
		t.Errorf("total = %d, want >= 1", body.Total)
	}
}

func TestGroups_AdminGet(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)
	groupID := e.CreateGroup(t, tokens.AccessToken, "Get Team")

	resp := e.DoJSON(t, "GET", "/api/v1/groups/"+groupID.String(), nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var group struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&group); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if group.Name != "Get Team" {
		t.Errorf("name = %q, want %q", group.Name, "Get Team")
	}
}

func TestGroups_AdminUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)
	groupID := e.CreateGroup(t, tokens.AccessToken, "Update Team")

	resp := e.DoJSON(t, "PUT", "/api/v1/groups/"+groupID.String(), map[string]any{
		"name":        "Updated Team",
		"description": "A renamed team",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestGroups_AdminDelete(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)
	groupID := e.CreateGroup(t, tokens.AccessToken, "Delete Team")

	resp := e.DoJSON(t, "DELETE", "/api/v1/groups/"+groupID.String(), nil, tokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d", resp.StatusCode)
	}

	// Verify deleted
	getResp := e.DoJSON(t, "GET", "/api/v1/groups/"+groupID.String(), nil, tokens.AccessToken)
	defer getResp.Body.Close()
	testutil.RequireStatus(t, getResp, http.StatusNotFound)
}

func TestGroups_NonAdminCannotCreateGroup(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "groupreguser", "groupreg@test.local", "Password123!", false)
	userTokens := e.Login(t, "groupreguser", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/groups", map[string]string{
		"name": "Unauthorized Group",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestGroups_MyGroups(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, adminTokens.AccessToken, "mygroupsuser", "mygroups@test.local", "Password123!", false)
	groupID := e.CreateGroup(t, adminTokens.AccessToken, "My Groups Team")
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "mygroupsuser", "Password123!")
	resp := e.DoJSON(t, "GET", "/api/v1/users/me/groups", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var groups []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&groups); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, g := range groups {
		if g.ID == groupID.String() {
			found = true
			break
		}
	}
	if !found {
		t.Error("user's group not found in my groups")
	}
}

func TestGroups_UpdateMarker(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Marker Team")

	icon := "star"
	color := "#ff0000"
	resp := e.DoJSON(t, "PUT", "/api/v1/groups/"+groupID.String()+"/marker", map[string]any{
		"marker_icon":  icon,
		"marker_color": color,
	}, adminTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}
