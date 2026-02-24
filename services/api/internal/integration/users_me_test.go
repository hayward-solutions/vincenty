package integration

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/sitaware/api/internal/testutil"
)

func TestUsersMe_GetProfile(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "GET", "/api/v1/users/me", nil, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var user struct {
		Username string `json:"username"`
		Email    string `json:"email"`
		IsAdmin  bool   `json:"is_admin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.Username != "admin" {
		t.Errorf("username = %q, want %q", user.Username, "admin")
	}
	if !user.IsAdmin {
		t.Error("expected is_admin=true")
	}
}

func TestUsersMe_GetProfile_Unauthenticated(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "GET", "/api/v1/users/me", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusUnauthorized)
}

func TestUsersMe_UpdateProfile(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "profileuser", "profile@test.local", "Password123!", false)
	tokens := e.Login(t, "profileuser", "Password123!")

	resp := e.DoJSON(t, "PUT", "/api/v1/users/me", map[string]any{
		"display_name": "Profile User Display",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	// Verify the update
	getResp := e.DoJSON(t, "GET", "/api/v1/users/me", nil, tokens.AccessToken)
	defer getResp.Body.Close()
	testutil.RequireStatus(t, getResp, http.StatusOK)

	var user struct {
		DisplayName *string `json:"display_name"`
	}
	if err := json.NewDecoder(getResp.Body).Decode(&user); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if user.DisplayName == nil || *user.DisplayName != "Profile User Display" {
		t.Errorf("display_name = %v, want %q", user.DisplayName, "Profile User Display")
	}
}

func TestUsersMe_ChangePassword(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "pwdchangeuser", "pwdchange@test.local", "OldPass123!", false)
	tokens := e.Login(t, "pwdchangeuser", "OldPass123!")

	resp := e.DoJSON(t, "PUT", "/api/v1/users/me/password", map[string]string{
		"current_password": "OldPass123!",
		"new_password":     "NewPass456!",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	// Login with new password should succeed
	newTokens := e.Login(t, "pwdchangeuser", "NewPass456!")
	if newTokens.AccessToken == "" {
		t.Error("login with new password failed")
	}
}

func TestUsersMe_ChangePassword_WrongCurrent(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "pwdwronguser", "pwdwrong@test.local", "Password123!", false)
	tokens := e.Login(t, "pwdwronguser", "Password123!")

	resp := e.DoJSON(t, "PUT", "/api/v1/users/me/password", map[string]string{
		"current_password": "WrongPassword!",
		"new_password":     "NewPass456!",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}

func TestUsersMe_ChangePassword_TooShort(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "pwdshortuser", "pwdshort@test.local", "Password123!", false)
	tokens := e.Login(t, "pwdshortuser", "Password123!")

	resp := e.DoJSON(t, "PUT", "/api/v1/users/me/password", map[string]string{
		"current_password": "Password123!",
		"new_password":     "short",
	}, tokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}
