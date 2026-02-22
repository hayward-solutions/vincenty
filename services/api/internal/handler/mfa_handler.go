package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/google/uuid"
	"github.com/sitaware/api/internal/middleware"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/service"
)

// MFAHandler handles MFA configuration and login challenge endpoints.
type MFAHandler struct {
	mfaService  *service.MFAService
	authService *service.AuthService
}

// NewMFAHandler creates a new MFAHandler.
func NewMFAHandler(mfaService *service.MFAService, authService *service.AuthService) *MFAHandler {
	return &MFAHandler{mfaService: mfaService, authService: authService}
}

// ---------------------------------------------------------------------------
// TOTP setup (authenticated)
// ---------------------------------------------------------------------------

// SetupTOTP handles POST /api/v1/users/me/mfa/totp/setup
func (h *MFAHandler) SetupTOTP(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.TOTPSetupRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.mfaService.SetupTOTP(r.Context(), claims.UserID, req.Name)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// VerifyTOTPSetup handles POST /api/v1/users/me/mfa/totp/verify
func (h *MFAHandler) VerifyTOTPSetup(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.TOTPVerifyRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	codes, err := h.mfaService.VerifyTOTPSetup(r.Context(), claims.UserID, req.MethodID, req.Code)
	if err != nil {
		HandleError(w, err)
		return
	}

	// If this was the first MFA method, recovery codes are returned
	if codes != nil {
		JSON(w, http.StatusOK, map[string]any{
			"verified":       true,
			"recovery_codes": codes.Codes,
		})
		return
	}

	JSON(w, http.StatusOK, map[string]any{"verified": true})
}

// ---------------------------------------------------------------------------
// WebAuthn registration (authenticated)
// ---------------------------------------------------------------------------

// BeginWebAuthnRegister handles POST /api/v1/users/me/mfa/webauthn/register/begin
func (h *MFAHandler) BeginWebAuthnRegister(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	req, err := Decode[model.WebAuthnRegisterRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	options, err := h.mfaService.BeginWebAuthnRegistration(r.Context(), claims.UserID, req.Name)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, options)
}

// FinishWebAuthnRegister handles POST /api/v1/users/me/mfa/webauthn/register/finish
func (h *MFAHandler) FinishWebAuthnRegister(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "failed to read request body")
		return
	}

	parsed, err := protocol.ParseCredentialCreationResponseBody(
		bytesReadCloser(body),
	)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid webauthn attestation: "+err.Error())
		return
	}

	codes, err := h.mfaService.FinishWebAuthnRegistration(r.Context(), claims.UserID, parsed)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp := map[string]any{"registered": true}
	if codes != nil {
		resp["recovery_codes"] = codes.Codes
	}
	JSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Method management (authenticated)
// ---------------------------------------------------------------------------

// ListMethods handles GET /api/v1/users/me/mfa/methods
func (h *MFAHandler) ListMethods(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	methods, err := h.mfaService.ListMethods(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	if methods == nil {
		methods = []model.MFAMethodResponse{}
	}

	JSON(w, http.StatusOK, methods)
}

// DeleteMethod handles DELETE /api/v1/users/me/mfa/methods/{id}
func (h *MFAHandler) DeleteMethod(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	methodID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid method id")
		return
	}

	if err := h.mfaService.DeleteMethod(r.Context(), claims.UserID, methodID); err != nil {
		HandleError(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// TogglePasswordless handles PUT /api/v1/users/me/mfa/webauthn/{id}/passwordless
func (h *MFAHandler) TogglePasswordless(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	credID, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid credential id")
		return
	}

	req, err := Decode[model.PasskeyToggleRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}

	if err := h.mfaService.TogglePasswordless(r.Context(), claims.UserID, credID, req.PasswordlessEnabled); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{"passwordless_enabled": req.PasswordlessEnabled})
}

// RegenerateRecoveryCodes handles POST /api/v1/users/me/mfa/recovery-codes
func (h *MFAHandler) RegenerateRecoveryCodes(w http.ResponseWriter, r *http.Request) {
	claims, ok := middleware.ClaimsFromContext(r.Context())
	if !ok {
		Error(w, http.StatusUnauthorized, "unauthorized", "missing auth context")
		return
	}

	codes, err := h.mfaService.GenerateRecoveryCodes(r.Context(), claims.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, codes)
}

// ---------------------------------------------------------------------------
// MFA login challenge endpoints (public, requires mfa_token)
// ---------------------------------------------------------------------------

// MFAVerifyTOTP handles POST /api/v1/auth/mfa/totp
func (h *MFAHandler) MFAVerifyTOTP(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.MFAVerifyTOTPRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	// Consume the MFA token to get the user ID
	session, err := h.mfaService.ValidateMFAToken(r.Context(), req.MFAToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	valid, err := h.mfaService.ValidateTOTP(r.Context(), session.UserID, req.Code)
	if err != nil {
		HandleError(w, err)
		return
	}
	if !valid {
		Error(w, http.StatusUnauthorized, "unauthorized", "invalid TOTP code")
		return
	}

	// MFA passed — issue tokens
	resp, err := h.authService.PasskeyLogin(r.Context(), session.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// MFARecovery handles POST /api/v1/auth/mfa/recovery
func (h *MFAHandler) MFARecovery(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.MFARecoveryRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	session, err := h.mfaService.ValidateMFAToken(r.Context(), req.MFAToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	valid, err := h.mfaService.ValidateRecoveryCode(r.Context(), session.UserID, req.Code)
	if err != nil {
		HandleError(w, err)
		return
	}
	if !valid {
		Error(w, http.StatusUnauthorized, "unauthorized", "invalid recovery code")
		return
	}

	resp, err := h.authService.PasskeyLogin(r.Context(), session.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// MFAWebAuthnBegin handles POST /api/v1/auth/mfa/webauthn/begin
func (h *MFAHandler) MFAWebAuthnBegin(w http.ResponseWriter, r *http.Request) {
	req, err := Decode[model.MFAWebAuthnBeginRequest](r)
	if err != nil {
		HandleError(w, err)
		return
	}
	if err := req.Validate(); err != nil {
		HandleError(w, err)
		return
	}

	// Peek at MFA token without consuming it (finish step will consume it)
	session, err := h.mfaService.PeekMFAToken(r.Context(), req.MFAToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	options, err := h.mfaService.BeginWebAuthnLogin(r.Context(), session.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]any{
		"options":   options,
		"mfa_token": req.MFAToken,
	})
}

// MFAWebAuthnFinish handles POST /api/v1/auth/mfa/webauthn/finish
func (h *MFAHandler) MFAWebAuthnFinish(w http.ResponseWriter, r *http.Request) {
	// Read raw body to extract both mfa_token and the assertion data
	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "failed to read request body")
		return
	}

	// Extract mfa_token from the body
	var envelope struct {
		MFAToken string `json:"mfa_token"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.MFAToken == "" {
		Error(w, http.StatusBadRequest, "validation_error", "mfa_token is required")
		return
	}

	session, err := h.mfaService.ValidateMFAToken(r.Context(), envelope.MFAToken)
	if err != nil {
		HandleError(w, err)
		return
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(
		bytesReadCloser(body),
	)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid webauthn assertion: "+err.Error())
		return
	}

	if err := h.mfaService.FinishWebAuthnLogin(r.Context(), session.UserID, parsed); err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.authService.PasskeyLogin(r.Context(), session.UserID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Passkey (passwordless) login endpoints
// ---------------------------------------------------------------------------

// PasskeyBegin handles POST /api/v1/auth/passkey/begin
func (h *MFAHandler) PasskeyBegin(w http.ResponseWriter, r *http.Request) {
	options, err := h.mfaService.BeginPasskeyLogin(r.Context())
	if err != nil {
		HandleError(w, err)
		return
	}
	JSON(w, http.StatusOK, options)
}

// PasskeyFinish handles POST /api/v1/auth/passkey/finish
func (h *MFAHandler) PasskeyFinish(w http.ResponseWriter, r *http.Request) {
	body, err := io.ReadAll(r.Body)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "failed to read request body")
		return
	}

	// Extract session_id from the body
	var envelope struct {
		SessionID string `json:"session_id"`
	}
	if err := json.Unmarshal(body, &envelope); err != nil || envelope.SessionID == "" {
		Error(w, http.StatusBadRequest, "validation_error", "session_id is required")
		return
	}

	parsed, err := protocol.ParseCredentialRequestResponseBody(
		bytesReadCloser(body),
	)
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid passkey assertion: "+err.Error())
		return
	}

	userID, err := h.mfaService.FinishPasskeyLogin(r.Context(), envelope.SessionID, parsed)
	if err != nil {
		HandleError(w, err)
		return
	}

	resp, err := h.authService.PasskeyLogin(r.Context(), userID)
	if err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, resp)
}

// ---------------------------------------------------------------------------
// Admin: reset user MFA
// ---------------------------------------------------------------------------

// AdminResetMFA handles DELETE /api/v1/users/{id}/mfa
func (h *MFAHandler) AdminResetMFA(w http.ResponseWriter, r *http.Request) {
	id, err := uuid.Parse(r.PathValue("id"))
	if err != nil {
		Error(w, http.StatusBadRequest, "validation_error", "invalid user id")
		return
	}

	if err := h.mfaService.AdminResetMFA(r.Context(), id); err != nil {
		HandleError(w, err)
		return
	}

	JSON(w, http.StatusOK, map[string]string{"status": "mfa_reset"})
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// bytesReadCloser wraps a byte slice as an io.ReadCloser for protocol parsing.
func bytesReadCloser(data []byte) io.ReadCloser {
	return io.NopCloser(bytes.NewReader(data))
}
