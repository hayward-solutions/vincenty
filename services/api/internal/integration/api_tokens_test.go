package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/testutil"
)

// ---------------------------------------------------------------------------
// API Token CRUD + Authentication
// ---------------------------------------------------------------------------

func TestAPIToken_CreateListDelete(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create an API token
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": "integration-test"}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var created struct {
		Token string `json:"token"`
		ID    string `json:"id"`
		Name  string `json:"name"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&created); err != nil {
		t.Fatalf("decode create response: %v", err)
	}
	if created.Token == "" {
		t.Fatal("raw token should be returned on creation")
	}
	if created.Name != "integration-test" {
		t.Errorf("name = %q, want %q", created.Name, "integration-test")
	}

	// List tokens — should contain the new one
	resp2 := e.DoJSON(t, "GET", "/api/v1/users/me/api-tokens", nil, tokens.AccessToken)
	defer resp2.Body.Close()
	testutil.RequireStatus(t, resp2, http.StatusOK)

	var listed []struct {
		ID   string `json:"id"`
		Name string `json:"name"`
	}
	if err := json.NewDecoder(resp2.Body).Decode(&listed); err != nil {
		t.Fatalf("decode list response: %v", err)
	}

	found := false
	for _, tok := range listed {
		if tok.ID == created.ID {
			found = true
		}
	}
	if !found {
		t.Errorf("created token %s not found in list", created.ID)
	}

	// Delete the token
	resp3 := e.DoJSON(t, "DELETE", "/api/v1/users/me/api-tokens/"+created.ID, nil, tokens.AccessToken)
	defer resp3.Body.Close()
	testutil.RequireStatus(t, resp3, http.StatusNoContent)

	// List again — should be gone
	resp4 := e.DoJSON(t, "GET", "/api/v1/users/me/api-tokens", nil, tokens.AccessToken)
	defer resp4.Body.Close()
	testutil.RequireStatus(t, resp4, http.StatusOK)

	var listedAfter []struct {
		ID string `json:"id"`
	}
	json.NewDecoder(resp4.Body).Decode(&listedAfter)
	for _, tok := range listedAfter {
		if tok.ID == created.ID {
			t.Error("deleted token should not appear in list")
		}
	}
}

func TestAPIToken_AuthenticateWithAPIToken(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create an API token
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": "auth-test"}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var created struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&created)

	// Use the API token to call a protected endpoint
	resp2 := e.DoJSON(t, "GET", "/api/v1/users/me", nil, created.Token)
	defer resp2.Body.Close()
	testutil.RequireStatus(t, resp2, http.StatusOK)

	var me struct {
		Username string `json:"username"`
	}
	json.NewDecoder(resp2.Body).Decode(&me)
	if me.Username == "" {
		t.Error("expected user data in response")
	}

	// Clean up
	e.DoJSON(t, "DELETE", "/api/v1/users/me/api-tokens/"+created.ID, nil, tokens.AccessToken)
}

func TestAPIToken_DeletedTokenRejected(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Create and immediately delete a token
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": "delete-test"}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var created struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&created)

	e.DoJSON(t, "DELETE", "/api/v1/users/me/api-tokens/"+created.ID, nil, tokens.AccessToken)

	// Try to use the deleted token
	resp2 := e.DoJSON(t, "GET", "/api/v1/users/me", nil, created.Token)
	defer resp2.Body.Close()
	testutil.RequireStatus(t, resp2, http.StatusUnauthorized)
}

func TestAPIToken_ValidationErrors(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	// Empty name
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": ""}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)

	// Delete non-existent token
	resp2 := e.DoJSON(t, "DELETE", "/api/v1/users/me/api-tokens/"+uuid.New().String(), nil, tokens.AccessToken)
	defer resp2.Body.Close()
	testutil.RequireStatus(t, resp2, http.StatusNotFound)
}

func TestAPIToken_NonAdminCanCreateAndUse(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	// Create a regular user
	e.CreateUser(t, adminTokens.AccessToken, "tokenuser", "tokenuser@test.com", "password123", false)
	userTokens := e.Login(t, "tokenuser", "password123")

	// Regular user creates an API token
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/api-tokens",
		map[string]string{"name": "user-token"}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var created struct {
		Token string `json:"token"`
		ID    string `json:"id"`
	}
	json.NewDecoder(resp.Body).Decode(&created)

	// Use the token — should work and return the user's data (not admin)
	resp2 := e.DoJSON(t, "GET", "/api/v1/users/me", nil, created.Token)
	defer resp2.Body.Close()
	testutil.RequireStatus(t, resp2, http.StatusOK)

	var me struct {
		Username string `json:"username"`
		IsAdmin  bool   `json:"is_admin"`
	}
	json.NewDecoder(resp2.Body).Decode(&me)
	if me.Username != "tokenuser" {
		t.Errorf("username = %q, want %q", me.Username, "tokenuser")
	}
	if me.IsAdmin {
		t.Error("non-admin user token should not grant admin access")
	}

	// Non-admin token should be rejected for admin endpoint
	resp3 := e.DoJSON(t, "GET", "/api/v1/users", nil, created.Token)
	defer resp3.Body.Close()
	testutil.RequireStatus(t, resp3, http.StatusForbidden)
}
