package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestUsers_AdminList(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/users?page=1&page_size=50", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var body struct {
		Data  []json.RawMessage `json:"data"`
		Total int               `json:"total"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}
	// At least the bootstrap admin should exist
	if body.Total < 1 {
		t.Errorf("total = %d, want >= 1", body.Total)
	}
}

func TestUsers_AdminCreate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, tokens.AccessToken, "newuser1", "newuser1@test.local", "Password123!", false)
	if userID.String() == "" {
		t.Error("user ID is empty")
	}
}

func TestUsers_AdminCreate_DuplicateUsername(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	e.CreateUser(t, tokens.AccessToken, "dupuser", "dupuser@test.local", "Password123!", false)

	// Second create with same username should fail
	resp := e.DoJSON(t, "POST", "/api/v1/users", map[string]any{
		"username": "dupuser",
		"email":    "dupuser2@test.local",
		"password": "Password123!",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusConflict)
}

func TestUsers_AdminCreate_DuplicateEmail(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	e.CreateUser(t, tokens.AccessToken, "emaildup1", "same@test.local", "Password123!", false)

	resp := e.DoJSON(t, "POST", "/api/v1/users", map[string]any{
		"username": "emaildup2",
		"email":    "same@test.local",
		"password": "Password123!",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusConflict)
}

func TestUsers_AdminGetByID(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, tokens.AccessToken, "getuser", "getuser@test.local", "Password123!", false)

	resp := e.DoJSON(t, "GET", "/api/v1/users/"+userID.String(), nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var user struct {
		ID       string `json:"id"`
		Username string `json:"username"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.Username != "getuser" {
		t.Errorf("username = %q, want %q", user.Username, "getuser")
	}
}

func TestUsers_AdminUpdate(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, tokens.AccessToken, "updateuser", "update@test.local", "Password123!", false)

	resp := e.DoJSON(t, "PUT", "/api/v1/users/"+userID.String(), map[string]any{
		"display_name": "Updated Name",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestUsers_AdminDelete(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	userID := e.CreateUser(t, tokens.AccessToken, "deleteuser", "delete@test.local", "Password123!", false)

	resp := e.DoJSON(t, "DELETE", "/api/v1/users/"+userID.String(), nil, tokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete status = %d, want 200 or 204", resp.StatusCode)
	}

	// Verify deleted
	getResp := e.DoJSON(t, "GET", "/api/v1/users/"+userID.String(), nil, tokens.AccessToken)
	defer getResp.Body.Close()
	testutil.RequireStatus(t, getResp, http.StatusNotFound)
}

func TestUsers_AdminDelete_LastAdmin(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Get the admin user's ID by listing users
	resp := e.DoJSON(t, "GET", "/api/v1/users?page=1&page_size=100", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var body struct {
		Data []struct {
			ID       string `json:"id"`
			Username string `json:"username"`
			IsAdmin  bool   `json:"is_admin"`
		} `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&body); err != nil {
		t.Fatalf("decode: %v", err)
	}

	var adminID string
	for _, u := range body.Data {
		if u.Username == "admin" {
			adminID = u.ID
			break
		}
	}
	if adminID == "" {
		t.Fatal("could not find admin user")
	}

	// Attempt to delete the last admin should fail
	delResp := e.DoJSON(t, "DELETE", "/api/v1/users/"+adminID, nil, tokens.AccessToken)
	defer delResp.Body.Close()
	if delResp.StatusCode == http.StatusOK || delResp.StatusCode == http.StatusNoContent {
		t.Error("deleting last admin should fail, got success")
	}
}

func TestUsers_NonAdminCannotListUsers(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "regularuser1", "regular1@test.local", "Password123!", false)
	userTokens := e.Login(t, "regularuser1", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/users", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestUsers_NonAdminCannotCreate(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "regularuser2", "regular2@test.local", "Password123!", false)
	userTokens := e.Login(t, "regularuser2", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/users", map[string]any{
		"username": "hacker",
		"email":    "hacker@test.local",
		"password": "Password123!",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestUsers_GetNotFound(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", fmt.Sprintf("/api/v1/users/%s", "00000000-0000-0000-0000-000000000099"), nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusNotFound)
}
