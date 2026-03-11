package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/vincenty/api/internal/testutil"
)

func TestAuth_Login_Success(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	if tokens.AccessToken == "" {
		t.Error("access_token is empty")
	}
	if tokens.RefreshToken == "" {
		t.Error("refresh_token is empty")
	}
}

func TestAuth_Login_InvalidPassword(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "admin",
		"password": "wrong-password",
	}, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}

func TestAuth_Login_NonExistentUser(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "nobody",
		"password": "password123",
	}, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}

func TestAuth_Login_EmptyBody(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "",
		"password": "",
	}, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}

func TestAuth_Refresh_Success(t *testing.T) {
	e := getEnv(t)
	tokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var newTokens testutil.AuthTokens
	if err := json.NewDecoder(resp.Body).Decode(&newTokens); err != nil {
		t.Fatalf("decode refresh response: %v", err)
	}
	if newTokens.AccessToken == "" {
		t.Error("new access_token is empty")
	}
	if newTokens.RefreshToken == "" {
		t.Error("new refresh_token is empty")
	}
	// Refresh token should be rotated
	if newTokens.RefreshToken == tokens.RefreshToken {
		t.Error("refresh_token was not rotated")
	}
}

func TestAuth_Refresh_InvalidToken(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": "not-a-valid-refresh-token",
	}, "")
	defer resp.Body.Close()
	// Should be 401 for invalid refresh token
	if resp.StatusCode != http.StatusUnauthorized && resp.StatusCode != http.StatusBadRequest {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("expected 401 or 400, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestAuth_Refresh_UsedTokenIsRevoked(t *testing.T) {
	e := getEnv(t)
	// Create a user specifically for this test to avoid interference
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "refresh-test-user", "refresh@test.local", "Password123!", false)
	tokens := e.Login(t, "refresh-test-user", "Password123!")

	// First refresh should succeed
	resp1 := e.DoJSON(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, "")
	resp1.Body.Close()
	testutil.RequireStatus(t, resp1, http.StatusOK)

	// Second refresh with the SAME (now-consumed) token should fail
	resp2 := e.DoJSON(t, "POST", "/api/v1/auth/refresh", map[string]string{
		"refresh_token": tokens.RefreshToken,
	}, "")
	defer resp2.Body.Close()
	if resp2.StatusCode == http.StatusOK {
		t.Error("reuse of consumed refresh token should fail, got 200")
	}
}

func TestAuth_Logout_Success(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	resp := e.DoJSON(t, "POST", "/api/v1/auth/logout", map[string]string{
		"refresh_token": adminTokens.RefreshToken,
	}, adminTokens.AccessToken)
	defer resp.Body.Close()
	// 200 or 204 are both acceptable
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		b, _ := io.ReadAll(resp.Body)
		t.Errorf("logout expected 200/204, got %d: %s", resp.StatusCode, string(b))
	}
}

func TestAuth_Logout_NoToken(t *testing.T) {
	e := getEnv(t)
	resp := e.DoJSON(t, "POST", "/api/v1/auth/logout", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusUnauthorized)
}

func TestAuth_DisabledAccount(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	// Create a user, then disable them
	userID := e.CreateUser(t, adminTokens.AccessToken, "disabled-user", "disabled@test.local", "Password123!", false)

	// Disable the user via admin API
	resp := e.DoJSON(t, "PUT", "/api/v1/users/"+userID.String(), map[string]any{
		"is_active": false,
	}, adminTokens.AccessToken)
	resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	// Attempt login should fail
	loginResp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "disabled-user",
		"password": "Password123!",
	}, "")
	defer loginResp.Body.Close()
	// Should be 403 or 400
	if loginResp.StatusCode == http.StatusOK {
		t.Error("login with disabled account should fail, got 200")
	}
}
