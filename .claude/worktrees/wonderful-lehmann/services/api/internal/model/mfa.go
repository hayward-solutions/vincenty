package model

import (
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// Database models
// ---------------------------------------------------------------------------

// TOTPMethod represents a TOTP authenticator app registered by a user.
type TOTPMethod struct {
	ID              uuid.UUID  `json:"-"`
	UserID          uuid.UUID  `json:"-"`
	Name            string     `json:"-"`
	SecretEncrypted []byte     `json:"-"`
	Verified        bool       `json:"-"`
	LastUsedAt      *time.Time `json:"-"`
	CreatedAt       time.Time  `json:"-"`
	UpdatedAt       time.Time  `json:"-"`
}

// WebAuthnCredential represents a WebAuthn security key or passkey.
type WebAuthnCredential struct {
	ID                  uuid.UUID  `json:"-"`
	UserID              uuid.UUID  `json:"-"`
	Name                string     `json:"-"`
	CredentialID        []byte     `json:"-"`
	PublicKey           []byte     `json:"-"`
	AAGUID              []byte     `json:"-"`
	SignCount           int64      `json:"-"`
	Transports          []string   `json:"-"`
	BackupEligible      bool       `json:"-"`
	BackupState         bool       `json:"-"`
	PasswordlessEnabled bool       `json:"-"`
	LastUsedAt          *time.Time `json:"-"`
	CreatedAt           time.Time  `json:"-"`
	UpdatedAt           time.Time  `json:"-"`
}

// RecoveryCode represents a one-time-use recovery code.
type RecoveryCode struct {
	ID        uuid.UUID  `json:"-"`
	UserID    uuid.UUID  `json:"-"`
	CodeHash  string     `json:"-"`
	UsedAt    *time.Time `json:"-"`
	CreatedAt time.Time  `json:"-"`
}

// ServerSetting represents a key-value server setting.
type ServerSetting struct {
	Key       string    `json:"key"`
	Value     string    `json:"value"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ---------------------------------------------------------------------------
// API response types
// ---------------------------------------------------------------------------

// MFAMethodResponse is the JSON representation of an enrolled MFA method.
type MFAMethodResponse struct {
	ID                  uuid.UUID  `json:"id"`
	Type                string     `json:"type"` // "totp" or "webauthn"
	Name                string     `json:"name"`
	Verified            bool       `json:"verified"`
	PasswordlessEnabled bool       `json:"passwordless_enabled,omitempty"`
	LastUsedAt          *time.Time `json:"last_used_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
}

// TOTPSetupResponse is returned when beginning TOTP setup.
type TOTPSetupResponse struct {
	MethodID uuid.UUID `json:"method_id"`
	Secret   string    `json:"secret"`
	URI      string    `json:"uri"`
	Issuer   string    `json:"issuer"`
	Account  string    `json:"account"`
}

// RecoveryCodesResponse is returned when recovery codes are generated.
type RecoveryCodesResponse struct {
	Codes []string `json:"codes"`
}

// MFAChallengeResponse is returned during login when MFA verification is required.
type MFAChallengeResponse struct {
	MFARequired bool     `json:"mfa_required"`
	MFAToken    string   `json:"mfa_token"`
	Methods     []string `json:"methods"` // e.g. ["totp", "webauthn", "recovery"]
}

// ServerSettingsResponse is the JSON representation of server settings.
type ServerSettingsResponse struct {
	MFARequired       bool   `json:"mfa_required"`
	MapboxAccessToken string `json:"mapbox_access_token"`
	GoogleMapsApiKey  string `json:"google_maps_api_key"`
}

// ---------------------------------------------------------------------------
// Request types
// ---------------------------------------------------------------------------

// TOTPSetupRequest is the expected body for beginning TOTP setup.
type TOTPSetupRequest struct {
	Name string `json:"name"`
}

// Validate checks that required fields are present.
func (r *TOTPSetupRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	return nil
}

// TOTPVerifyRequest is the expected body for verifying a TOTP code during setup.
type TOTPVerifyRequest struct {
	MethodID uuid.UUID `json:"method_id"`
	Code     string    `json:"code"`
}

// Validate checks that required fields are present.
func (r *TOTPVerifyRequest) Validate() error {
	if r.MethodID == uuid.Nil {
		return ErrValidation("method_id is required")
	}
	if r.Code == "" {
		return ErrValidation("code is required")
	}
	if len(r.Code) != 6 {
		return ErrValidation("code must be 6 digits")
	}
	return nil
}

// MFAVerifyTOTPRequest is the expected body for verifying TOTP during login.
type MFAVerifyTOTPRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

// Validate checks that required fields are present.
func (r *MFAVerifyTOTPRequest) Validate() error {
	if r.MFAToken == "" {
		return ErrValidation("mfa_token is required")
	}
	if r.Code == "" {
		return ErrValidation("code is required")
	}
	return nil
}

// MFARecoveryRequest is the expected body for using a recovery code during login.
type MFARecoveryRequest struct {
	MFAToken string `json:"mfa_token"`
	Code     string `json:"code"`
}

// Validate checks that required fields are present.
func (r *MFARecoveryRequest) Validate() error {
	if r.MFAToken == "" {
		return ErrValidation("mfa_token is required")
	}
	if r.Code == "" {
		return ErrValidation("code is required")
	}
	return nil
}

// WebAuthnRegisterRequest is the expected body for beginning WebAuthn registration.
type WebAuthnRegisterRequest struct {
	Name string `json:"name"`
}

// Validate checks that required fields are present.
func (r *WebAuthnRegisterRequest) Validate() error {
	if r.Name == "" {
		return ErrValidation("name is required")
	}
	return nil
}

// MFAWebAuthnBeginRequest is the expected body for beginning WebAuthn assertion during login.
type MFAWebAuthnBeginRequest struct {
	MFAToken string `json:"mfa_token"`
}

// Validate checks that required fields are present.
func (r *MFAWebAuthnBeginRequest) Validate() error {
	if r.MFAToken == "" {
		return ErrValidation("mfa_token is required")
	}
	return nil
}

// MFAWebAuthnFinishRequest is the expected body for completing WebAuthn assertion during login.
type MFAWebAuthnFinishRequest struct {
	MFAToken string `json:"mfa_token"`
	// The assertion response is parsed from the raw request body by the handler.
}

// PasskeyToggleRequest is the expected body for toggling passwordless on a credential.
type PasskeyToggleRequest struct {
	PasswordlessEnabled bool `json:"passwordless_enabled"`
}

// UpdateServerSettingsRequest is the expected body for updating server settings.
type UpdateServerSettingsRequest struct {
	MFARequired       *bool   `json:"mfa_required"`
	MapboxAccessToken *string `json:"mapbox_access_token"`
	GoogleMapsApiKey  *string `json:"google_maps_api_key"`
}

// ---------------------------------------------------------------------------
// Redis MFA session data
// ---------------------------------------------------------------------------

// MFASession is stored in Redis during the MFA challenge flow.
type MFASession struct {
	UserID  uuid.UUID `json:"user_id"`
	Methods []string  `json:"methods"`
}
