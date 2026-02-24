package integration

import (
	"encoding/json"
	"io"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/sitaware/api/internal/testutil"
)

// ---------------------------------------------------------------------------
// TOTP setup and verify
// ---------------------------------------------------------------------------

func TestMFA_TOTPSetup(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-setup-user", "mfasetup@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-setup-user", "Password123!")

	resp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "My Authenticator",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
		URI      string    `json:"uri"`
		Issuer   string    `json:"issuer"`
		Account  string    `json:"account"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&setup); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if setup.MethodID == uuid.Nil {
		t.Error("method_id is empty")
	}
	if setup.Secret == "" {
		t.Error("secret is empty")
	}
	if setup.URI == "" {
		t.Error("uri is empty")
	}
}

func TestMFA_TOTPVerify(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-verify-user", "mfaverify@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-verify-user", "Password123!")

	// Setup TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Verify Test",
	}, userTokens.AccessToken)
	testutil.RequireStatus(t, setupResp, http.StatusOK)

	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	// Generate a valid TOTP code from the secret
	code, err := totp.GenerateCode(setup.Secret, time.Now())
	if err != nil {
		t.Fatalf("generate totp code: %v", err)
	}

	// Verify the TOTP setup
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      code,
	}, userTokens.AccessToken)
	defer verifyResp.Body.Close()
	testutil.RequireStatus(t, verifyResp, http.StatusOK)

	var result struct {
		Verified      bool     `json:"verified"`
		RecoveryCodes []string `json:"recovery_codes"`
	}
	if err := json.NewDecoder(verifyResp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if !result.Verified {
		t.Error("expected verified=true")
	}
	// First MFA method should return recovery codes
	if len(result.RecoveryCodes) == 0 {
		t.Error("expected recovery codes to be returned for first MFA method")
	}
}

func TestMFA_TOTPVerifyInvalidCode(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-badcode-user", "mfabadcode@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-badcode-user", "Password123!")

	// Setup TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Bad Code Test",
	}, userTokens.AccessToken)
	testutil.RequireStatus(t, setupResp, http.StatusOK)

	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	// Try with an invalid code
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      "000000",
	}, userTokens.AccessToken)
	defer verifyResp.Body.Close()
	testutil.RequireStatus(t, verifyResp, http.StatusBadRequest)
}

// ---------------------------------------------------------------------------
// List MFA methods
// ---------------------------------------------------------------------------

func TestMFA_ListMethodsEmpty(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-list-user", "mfalist@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-list-user", "Password123!")

	resp := e.DoJSON(t, "GET", "/api/v1/users/me/mfa/methods", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var methods []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&methods); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(methods) != 0 {
		t.Errorf("expected 0 methods, got %d", len(methods))
	}
}

func TestMFA_ListMethodsAfterSetup(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-listsetup-user", "mfalistsetup@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-listsetup-user", "Password123!")

	// Setup and verify TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "List Test Auth",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	code, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      code,
	}, userTokens.AccessToken)
	verifyResp.Body.Close()

	// List methods
	resp := e.DoJSON(t, "GET", "/api/v1/users/me/mfa/methods", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var methods []struct {
		ID       uuid.UUID `json:"id"`
		Type     string    `json:"type"`
		Name     string    `json:"name"`
		Verified bool      `json:"verified"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&methods); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(methods) != 1 {
		t.Fatalf("expected 1 method, got %d", len(methods))
	}
	if methods[0].Type != "totp" {
		t.Errorf("type = %q, want %q", methods[0].Type, "totp")
	}
	if !methods[0].Verified {
		t.Error("expected method to be verified")
	}
}

// ---------------------------------------------------------------------------
// MFA login challenge flow
// ---------------------------------------------------------------------------

func TestMFA_LoginChallenge(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-challenge-user", "mfachallenge@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-challenge-user", "Password123!")

	// Setup and verify TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Challenge Auth",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	setupCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      setupCode,
	}, userTokens.AccessToken)
	verifyResp.Body.Close()

	// Now login should trigger MFA challenge
	loginResp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "mfa-challenge-user",
		"password": "Password123!",
	}, "")
	defer loginResp.Body.Close()

	// Login with MFA enabled returns 403 with MFA challenge
	// OR it might return 200 with mfa_required flag — check response
	b, _ := io.ReadAll(loginResp.Body)

	var challenge struct {
		MFARequired bool     `json:"mfa_required"`
		MFAToken    string   `json:"mfa_token"`
		Methods     []string `json:"methods"`
	}
	if err := json.Unmarshal(b, &challenge); err != nil {
		t.Fatalf("decode challenge: %v err; body=%s", err, string(b))
	}

	if !challenge.MFARequired {
		t.Fatal("expected mfa_required=true in login response")
	}
	if challenge.MFAToken == "" {
		t.Fatal("expected mfa_token to be non-empty")
	}

	// Now complete the MFA challenge with a valid TOTP code
	mfaCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	totpResp := e.DoJSON(t, "POST", "/api/v1/auth/mfa/totp", map[string]string{
		"mfa_token": challenge.MFAToken,
		"code":      mfaCode,
	}, "")
	defer totpResp.Body.Close()
	testutil.RequireStatus(t, totpResp, http.StatusOK)

	var tokens testutil.AuthTokens
	if err := json.NewDecoder(totpResp.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access_token after MFA verification")
	}
}

// ---------------------------------------------------------------------------
// Recovery code flow
// ---------------------------------------------------------------------------

func TestMFA_RecoveryCodeLogin(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-recovery-user", "mfarecovery@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-recovery-user", "Password123!")

	// Setup and verify TOTP (which generates recovery codes)
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Recovery Auth",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	setupCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      setupCode,
	}, userTokens.AccessToken)

	var verifyResult struct {
		Verified      bool     `json:"verified"`
		RecoveryCodes []string `json:"recovery_codes"`
	}
	json.NewDecoder(verifyResp.Body).Decode(&verifyResult)
	verifyResp.Body.Close()

	if len(verifyResult.RecoveryCodes) == 0 {
		t.Fatal("no recovery codes returned")
	}

	// Login triggers MFA challenge
	loginResp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "mfa-recovery-user",
		"password": "Password123!",
	}, "")
	var challenge struct {
		MFAToken string `json:"mfa_token"`
	}
	json.NewDecoder(loginResp.Body).Decode(&challenge)
	loginResp.Body.Close()

	if challenge.MFAToken == "" {
		t.Fatal("expected mfa_token")
	}

	// Use a recovery code to complete the MFA challenge
	recoveryResp := e.DoJSON(t, "POST", "/api/v1/auth/mfa/recovery", map[string]string{
		"mfa_token": challenge.MFAToken,
		"code":      verifyResult.RecoveryCodes[0],
	}, "")
	defer recoveryResp.Body.Close()
	testutil.RequireStatus(t, recoveryResp, http.StatusOK)

	var tokens testutil.AuthTokens
	if err := json.NewDecoder(recoveryResp.Body).Decode(&tokens); err != nil {
		t.Fatalf("decode tokens: %v", err)
	}
	if tokens.AccessToken == "" {
		t.Error("expected access_token after recovery code login")
	}
}

// ---------------------------------------------------------------------------
// Regenerate recovery codes
// ---------------------------------------------------------------------------

func TestMFA_RegenerateRecoveryCodes(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-regen-user", "mfaregen@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-regen-user", "Password123!")

	// Setup and verify TOTP first
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Regen Auth",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	setupCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      setupCode,
	}, userTokens.AccessToken)
	verifyResp.Body.Close()

	// Regenerate recovery codes
	resp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/recovery-codes", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var codes struct {
		Codes []string `json:"codes"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&codes); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(codes.Codes) == 0 {
		t.Error("expected recovery codes")
	}
}

// ---------------------------------------------------------------------------
// Delete MFA method
// ---------------------------------------------------------------------------

func TestMFA_DeleteMethod(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-delete-user", "mfadelete@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-delete-user", "Password123!")

	// Setup and verify TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Delete Me",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	setupCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      setupCode,
	}, userTokens.AccessToken)
	verifyResp.Body.Close()

	// Delete the method
	resp := e.DoJSON(t, "DELETE", "/api/v1/users/me/mfa/methods/"+setup.MethodID.String(), nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusNoContent)

	// List should now be empty
	listResp := e.DoJSON(t, "GET", "/api/v1/users/me/mfa/methods", nil, userTokens.AccessToken)
	defer listResp.Body.Close()
	testutil.RequireStatus(t, listResp, http.StatusOK)

	var methods []json.RawMessage
	json.NewDecoder(listResp.Body).Decode(&methods)
	if len(methods) != 0 {
		t.Errorf("expected 0 methods after delete, got %d", len(methods))
	}
}

// ---------------------------------------------------------------------------
// Admin reset MFA
// ---------------------------------------------------------------------------

func TestMFA_AdminReset(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	userID := e.CreateUser(t, adminTokens.AccessToken, "mfa-adminreset-user", "mfaadminreset@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-adminreset-user", "Password123!")

	// Setup and verify TOTP
	setupResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "Admin Reset Auth",
	}, userTokens.AccessToken)
	var setup struct {
		MethodID uuid.UUID `json:"method_id"`
		Secret   string    `json:"secret"`
	}
	json.NewDecoder(setupResp.Body).Decode(&setup)
	setupResp.Body.Close()

	setupCode, _ := totp.GenerateCode(setup.Secret, time.Now())
	verifyResp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/verify", map[string]any{
		"method_id": setup.MethodID,
		"code":      setupCode,
	}, userTokens.AccessToken)
	verifyResp.Body.Close()

	// Admin resets user's MFA
	resp := e.DoJSON(t, "DELETE", "/api/v1/users/"+userID.String()+"/mfa", nil, adminTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var result struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if result.Status != "mfa_reset" {
		t.Errorf("status = %q, want %q", result.Status, "mfa_reset")
	}

	// User should now be able to login without MFA
	loginResp := e.DoJSON(t, "POST", "/api/v1/auth/login", map[string]string{
		"username": "mfa-adminreset-user",
		"password": "Password123!",
	}, "")
	defer loginResp.Body.Close()
	testutil.RequireStatus(t, loginResp, http.StatusOK)

	var tokens testutil.AuthTokens
	json.NewDecoder(loginResp.Body).Decode(&tokens)
	if tokens.AccessToken == "" {
		t.Error("expected direct login (no MFA challenge) after admin reset")
	}
}

// ---------------------------------------------------------------------------
// Unauthenticated access
// ---------------------------------------------------------------------------

func TestMFA_SetupRequiresAuth(t *testing.T) {
	e := getEnv(t)

	resp := e.DoJSON(t, "POST", "/api/v1/users/me/mfa/totp/setup", map[string]string{
		"name": "No Auth",
	}, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusUnauthorized)
}

func TestMFA_ListMethodsRequiresAuth(t *testing.T) {
	e := getEnv(t)

	resp := e.DoJSON(t, "GET", "/api/v1/users/me/mfa/methods", nil, "")
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusUnauthorized)
}

// ---------------------------------------------------------------------------
// Non-admin cannot reset another user's MFA
// ---------------------------------------------------------------------------

func TestMFA_NonAdminCannotReset(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)
	targetID := e.CreateUser(t, adminTokens.AccessToken, "mfa-target-user", "mfatarget@test.local", "Password123!", false)
	e.CreateUser(t, adminTokens.AccessToken, "mfa-nonadmin-user", "mfanonadmin@test.local", "Password123!", false)
	userTokens := e.Login(t, "mfa-nonadmin-user", "Password123!")

	resp := e.DoJSON(t, "DELETE", "/api/v1/users/"+targetID.String()+"/mfa", nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}
