package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"
	"time"

	"github.com/go-webauthn/webauthn/protocol"
	"github.com/go-webauthn/webauthn/webauthn"
	"github.com/google/uuid"
	"github.com/pquerna/otp/totp"
	"github.com/redis/go-redis/v9"
	"golang.org/x/crypto/bcrypt"

	"github.com/sitaware/api/internal/auth"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/repository"
)

const (
	// mfaTokenPrefix is the Redis key prefix for MFA challenge tokens.
	mfaTokenPrefix = "mfa:"
	// mfaTokenTTL is how long an MFA challenge token is valid.
	mfaTokenTTL = 5 * time.Minute
	// webauthnSessionPrefix is the Redis key prefix for WebAuthn session data.
	webauthnSessionPrefix = "webauthn:"
	// webauthnSessionTTL is how long a WebAuthn session is valid.
	webauthnSessionTTL = 5 * time.Minute
	// recoveryCodeCount is the number of recovery codes generated.
	recoveryCodeCount = 8
	// totpIssuer is the issuer shown in authenticator apps.
	totpIssuer = "SitAware"
)

// MFAService handles MFA business logic including TOTP, WebAuthn, and recovery codes.
type MFAService struct {
	mfaRepo    repository.MFARepo
	userRepo   repository.UserRepo
	encryptor  auth.SecretEncryptor
	rdb        *redis.Client
	webAuthn   *webauthn.WebAuthn
}

// NewMFAService creates a new MFAService.
func NewMFAService(
	mfaRepo repository.MFARepo,
	userRepo repository.UserRepo,
	encryptor auth.SecretEncryptor,
	rdb *redis.Client,
	wa *webauthn.WebAuthn,
) *MFAService {
	return &MFAService{
		mfaRepo:   mfaRepo,
		userRepo:  userRepo,
		encryptor: encryptor,
		rdb:       rdb,
		webAuthn:  wa,
	}
}

// ---------------------------------------------------------------------------
// TOTP
// ---------------------------------------------------------------------------

// SetupTOTP generates a new TOTP secret, encrypts it, and stores it as unverified.
func (s *MFAService) SetupTOTP(ctx context.Context, userID uuid.UUID, name string) (*model.TOTPSetupResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      totpIssuer,
		AccountName: user.Username,
	})
	if err != nil {
		return nil, fmt.Errorf("generate totp key: %w", err)
	}

	encrypted, err := s.encryptor.Encrypt(ctx, []byte(key.Secret()))
	if err != nil {
		return nil, fmt.Errorf("encrypt totp secret: %w", err)
	}

	method := &model.TOTPMethod{
		UserID:          userID,
		Name:            name,
		SecretEncrypted: encrypted,
		Verified:        false,
	}
	if err := s.mfaRepo.CreateTOTP(ctx, method); err != nil {
		return nil, err
	}

	return &model.TOTPSetupResponse{
		MethodID: method.ID,
		Secret:   key.Secret(),
		URI:      key.URL(),
		Issuer:   totpIssuer,
		Account:  user.Username,
	}, nil
}

// VerifyTOTPSetup validates a TOTP code against an unverified method and activates it.
// On the first verified MFA method, recovery codes are generated and returned.
func (s *MFAService) VerifyTOTPSetup(ctx context.Context, userID uuid.UUID, methodID uuid.UUID, code string) (*model.RecoveryCodesResponse, error) {
	method, err := s.mfaRepo.GetTOTPByID(ctx, methodID)
	if err != nil {
		return nil, err
	}
	if method.UserID != userID {
		return nil, model.ErrNotFound("totp method")
	}
	if method.Verified {
		return nil, model.ErrValidation("method is already verified")
	}

	secret, err := s.encryptor.Decrypt(ctx, method.SecretEncrypted)
	if err != nil {
		return nil, fmt.Errorf("decrypt totp secret: %w", err)
	}

	if !totp.Validate(code, string(secret)) {
		return nil, model.ErrValidation("invalid TOTP code")
	}

	if err := s.mfaRepo.VerifyTOTP(ctx, methodID); err != nil {
		return nil, err
	}

	// Enable MFA on the user if not already enabled
	return s.enableMFAAndGenerateCodes(ctx, userID)
}

// ValidateTOTP checks a TOTP code against all verified TOTP methods for a user.
// Returns true if any method matches.
func (s *MFAService) ValidateTOTP(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	methods, err := s.mfaRepo.ListTOTPByUser(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, m := range methods {
		if !m.Verified {
			continue
		}
		secret, err := s.encryptor.Decrypt(ctx, m.SecretEncrypted)
		if err != nil {
			slog.Error("failed to decrypt TOTP secret", "method_id", m.ID, "error", err)
			continue
		}
		if totp.Validate(code, string(secret)) {
			_ = s.mfaRepo.TouchTOTP(ctx, m.ID)
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// WebAuthn
// ---------------------------------------------------------------------------

// webAuthnUser adapts a model.User + credentials for the go-webauthn library.
type webAuthnUser struct {
	user  *model.User
	creds []model.WebAuthnCredential
}

func (u *webAuthnUser) WebAuthnID() []byte {
	id := u.user.ID
	return id[:]
}

func (u *webAuthnUser) WebAuthnName() string        { return u.user.Username }
func (u *webAuthnUser) WebAuthnDisplayName() string {
	if u.user.DisplayName != nil {
		return *u.user.DisplayName
	}
	return u.user.Username
}
func (u *webAuthnUser) WebAuthnCredentials() []webauthn.Credential {
	creds := make([]webauthn.Credential, 0, len(u.creds))
	for _, c := range u.creds {
		transports := make([]protocol.AuthenticatorTransport, 0, len(c.Transports))
		for _, t := range c.Transports {
			transports = append(transports, protocol.AuthenticatorTransport(t))
		}
		creds = append(creds, webauthn.Credential{
			ID:              c.CredentialID,
			PublicKey:       c.PublicKey,
			AttestationType: "",
			Transport:       transports,
			Flags: webauthn.CredentialFlags{
				BackupEligible: c.BackupEligible,
				BackupState:    c.BackupState,
			},
			Authenticator: webauthn.Authenticator{
				AAGUID:    c.AAGUID,
				SignCount: uint32(c.SignCount),
			},
		})
	}
	return creds
}
func (u *webAuthnUser) WebAuthnIcon() string { return "" }

func (s *MFAService) getWebAuthnUser(ctx context.Context, userID uuid.UUID) (*webAuthnUser, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	creds, err := s.mfaRepo.ListWebAuthnByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &webAuthnUser{user: user, creds: creds}, nil
}

// BeginWebAuthnRegistration starts a WebAuthn registration ceremony.
func (s *MFAService) BeginWebAuthnRegistration(ctx context.Context, userID uuid.UUID, name string) (any, error) {
	waUser, err := s.getWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	options, session, err := s.webAuthn.BeginRegistration(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin webauthn registration: %w", err)
	}

	// Store session in Redis
	sessionData, err := json.Marshal(struct {
		Session *webauthn.SessionData `json:"session"`
		Name    string                `json:"name"`
	}{Session: session, Name: name})
	if err != nil {
		return nil, fmt.Errorf("marshal webauthn session: %w", err)
	}

	sessionKey := webauthnSessionPrefix + "reg:" + userID.String()
	if err := s.rdb.Set(ctx, sessionKey, sessionData, webauthnSessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("store webauthn session: %w", err)
	}

	return options, nil
}

// FinishWebAuthnRegistration completes a WebAuthn registration ceremony.
func (s *MFAService) FinishWebAuthnRegistration(ctx context.Context, userID uuid.UUID, credentialCreationData *protocol.ParsedCredentialCreationData) (*model.RecoveryCodesResponse, error) {
	waUser, err := s.getWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	// Retrieve session from Redis
	sessionKey := webauthnSessionPrefix + "reg:" + userID.String()
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Bytes()
	if err != nil {
		return nil, model.ErrValidation("webauthn registration session expired or not found")
	}
	s.rdb.Del(ctx, sessionKey)

	var stored struct {
		Session *webauthn.SessionData `json:"session"`
		Name    string                `json:"name"`
	}
	if err := json.Unmarshal(sessionJSON, &stored); err != nil {
		return nil, fmt.Errorf("unmarshal webauthn session: %w", err)
	}

	credential, err := s.webAuthn.CreateCredential(waUser, *stored.Session, credentialCreationData)
	if err != nil {
		return nil, fmt.Errorf("create webauthn credential: %w", err)
	}

	transports := make([]string, 0, len(credential.Transport))
	for _, t := range credential.Transport {
		transports = append(transports, string(t))
	}

	// Auto-enable passwordless for backup-eligible credentials (platform
	// authenticators / synced passkeys) since these are always discoverable
	// and the user almost certainly intends to use them as passkeys.
	passwordless := credential.Flags.BackupEligible

	cred := &model.WebAuthnCredential{
		UserID:              userID,
		Name:                stored.Name,
		CredentialID:        credential.ID,
		PublicKey:           credential.PublicKey,
		AAGUID:              credential.Authenticator.AAGUID,
		SignCount:           int64(credential.Authenticator.SignCount),
		Transports:          transports,
		BackupEligible:      credential.Flags.BackupEligible,
		BackupState:         credential.Flags.BackupState,
		PasswordlessEnabled: passwordless,
	}
	if err := s.mfaRepo.CreateWebAuthn(ctx, cred); err != nil {
		return nil, err
	}

	return s.enableMFAAndGenerateCodes(ctx, userID)
}

// BeginWebAuthnLogin starts a WebAuthn assertion ceremony for MFA login.
func (s *MFAService) BeginWebAuthnLogin(ctx context.Context, userID uuid.UUID) (any, error) {
	waUser, err := s.getWebAuthnUser(ctx, userID)
	if err != nil {
		return nil, err
	}

	options, session, err := s.webAuthn.BeginLogin(waUser)
	if err != nil {
		return nil, fmt.Errorf("begin webauthn login: %w", err)
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("marshal webauthn session: %w", err)
	}

	sessionKey := webauthnSessionPrefix + "login:" + userID.String()
	if err := s.rdb.Set(ctx, sessionKey, sessionData, webauthnSessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("store webauthn session: %w", err)
	}

	return options, nil
}

// FinishWebAuthnLogin completes a WebAuthn assertion ceremony for MFA login.
func (s *MFAService) FinishWebAuthnLogin(ctx context.Context, userID uuid.UUID, assertionData *protocol.ParsedCredentialAssertionData) error {
	waUser, err := s.getWebAuthnUser(ctx, userID)
	if err != nil {
		return err
	}

	sessionKey := webauthnSessionPrefix + "login:" + userID.String()
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Bytes()
	if err != nil {
		return model.ErrValidation("webauthn login session expired or not found")
	}
	s.rdb.Del(ctx, sessionKey)

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionJSON, &session); err != nil {
		return fmt.Errorf("unmarshal webauthn session: %w", err)
	}

	credential, err := s.webAuthn.ValidateLogin(waUser, session, assertionData)
	if err != nil {
		slog.Error("webauthn MFA login validation failed", "user_id", userID, "error", err)
		return model.ErrValidation("webauthn assertion failed")
	}

	// Update sign count and backup state in DB
	dbCred, err := s.mfaRepo.GetWebAuthnByCredentialID(ctx, credential.ID)
	if err == nil {
		_ = s.mfaRepo.UpdateWebAuthnSignCount(ctx, dbCred.ID, int64(credential.Authenticator.SignCount), credential.Flags.BackupState)
	}

	return nil
}

// BeginPasskeyLogin starts a discoverable credential assertion (passwordless).
func (s *MFAService) BeginPasskeyLogin(ctx context.Context) (any, error) {
	options, session, err := s.webAuthn.BeginDiscoverableLogin()
	if err != nil {
		return nil, fmt.Errorf("begin passkey login: %w", err)
	}

	sessionData, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("marshal passkey session: %w", err)
	}

	// Use a random key since we don't know the user yet
	sessionID := uuid.New().String()
	sessionKey := webauthnSessionPrefix + "passkey:" + sessionID
	if err := s.rdb.Set(ctx, sessionKey, sessionData, webauthnSessionTTL).Err(); err != nil {
		return nil, fmt.Errorf("store passkey session: %w", err)
	}

	// Return both the options and the session ID so the client can send it back
	return map[string]any{
		"options":    options,
		"session_id": sessionID,
	}, nil
}

// FinishPasskeyLogin completes a discoverable credential assertion and returns the user ID.
func (s *MFAService) FinishPasskeyLogin(ctx context.Context, sessionID string, assertionData *protocol.ParsedCredentialAssertionData) (uuid.UUID, error) {
	sessionKey := webauthnSessionPrefix + "passkey:" + sessionID
	sessionJSON, err := s.rdb.Get(ctx, sessionKey).Bytes()
	if err != nil {
		return uuid.Nil, model.ErrValidation("passkey login session expired or not found")
	}
	s.rdb.Del(ctx, sessionKey)

	var session webauthn.SessionData
	if err := json.Unmarshal(sessionJSON, &session); err != nil {
		return uuid.Nil, fmt.Errorf("unmarshal passkey session: %w", err)
	}

	// Handler for discoverable credentials: look up the user by credential ID
	handler := func(rawID, userHandle []byte) (webauthn.User, error) {
		cred, err := s.mfaRepo.GetWebAuthnByCredentialID(ctx, rawID)
		if err != nil {
			return nil, fmt.Errorf("credential not found")
		}
		if !cred.PasswordlessEnabled {
			return nil, fmt.Errorf("credential is not enabled for passwordless login")
		}
		waUser, err := s.getWebAuthnUser(ctx, cred.UserID)
		if err != nil {
			return nil, err
		}
		return waUser, nil
	}

	credential, err := s.webAuthn.ValidateDiscoverableLogin(handler, session, assertionData)
	if err != nil {
		slog.Error("passkey login validation failed", "error", err)
		return uuid.Nil, model.ErrValidation("passkey assertion failed")
	}

	// Look up the credential to get the user ID and update sign count + backup state
	dbCred, err := s.mfaRepo.GetWebAuthnByCredentialID(ctx, credential.ID)
	if err != nil {
		return uuid.Nil, err
	}
	_ = s.mfaRepo.UpdateWebAuthnSignCount(ctx, dbCred.ID, int64(credential.Authenticator.SignCount), credential.Flags.BackupState)

	return dbCred.UserID, nil
}

// TogglePasswordless enables or disables passwordless login for a WebAuthn credential.
func (s *MFAService) TogglePasswordless(ctx context.Context, userID uuid.UUID, credID uuid.UUID, enabled bool) error {
	cred, err := s.mfaRepo.GetWebAuthnByID(ctx, credID)
	if err != nil {
		return err
	}
	if cred.UserID != userID {
		return model.ErrNotFound("webauthn credential")
	}
	return s.mfaRepo.UpdateWebAuthnPasswordless(ctx, credID, enabled)
}

// ---------------------------------------------------------------------------
// Recovery codes
// ---------------------------------------------------------------------------

// GenerateRecoveryCodes creates a new set of recovery codes for the user,
// replacing any existing ones. Returns the plaintext codes.
func (s *MFAService) GenerateRecoveryCodes(ctx context.Context, userID uuid.UUID) (*model.RecoveryCodesResponse, error) {
	codes := make([]string, recoveryCodeCount)
	hashes := make([]string, recoveryCodeCount)

	for i := range recoveryCodeCount {
		code := generateRecoveryCode()
		codes[i] = code
		// Normalize before hashing: strip dashes and lowercase, matching
		// the normalization in ValidateRecoveryCode.
		normalized := strings.ReplaceAll(strings.ToLower(code), "-", "")
		hash, err := bcrypt.GenerateFromPassword([]byte(normalized), bcrypt.DefaultCost)
		if err != nil {
			return nil, fmt.Errorf("hash recovery code: %w", err)
		}
		hashes[i] = string(hash)
	}

	if err := s.mfaRepo.CreateRecoveryCodes(ctx, userID, hashes); err != nil {
		return nil, err
	}

	return &model.RecoveryCodesResponse{Codes: codes}, nil
}

// ValidateRecoveryCode checks a recovery code against unused codes for the user.
// If valid, marks the code as used.
func (s *MFAService) ValidateRecoveryCode(ctx context.Context, userID uuid.UUID, code string) (bool, error) {
	// Normalize: remove dashes and convert to lowercase
	normalized := strings.ReplaceAll(strings.ToLower(strings.TrimSpace(code)), "-", "")

	unused, err := s.mfaRepo.ListUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return false, err
	}

	for _, rc := range unused {
		if err := bcrypt.CompareHashAndPassword([]byte(rc.CodeHash), []byte(normalized)); err == nil {
			if err := s.mfaRepo.MarkRecoveryCodeUsed(ctx, rc.ID); err != nil {
				return false, err
			}
			return true, nil
		}
	}
	return false, nil
}

// ---------------------------------------------------------------------------
// MFA method listing and deletion
// ---------------------------------------------------------------------------

// ListMethods returns all MFA methods for a user.
func (s *MFAService) ListMethods(ctx context.Context, userID uuid.UUID) ([]model.MFAMethodResponse, error) {
	var methods []model.MFAMethodResponse

	totpMethods, err := s.mfaRepo.ListTOTPByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, m := range totpMethods {
		if !m.Verified {
			continue // Don't show unverified methods in the list
		}
		methods = append(methods, model.MFAMethodResponse{
			ID:         m.ID,
			Type:       "totp",
			Name:       m.Name,
			Verified:   m.Verified,
			LastUsedAt: m.LastUsedAt,
			CreatedAt:  m.CreatedAt,
		})
	}

	webAuthnCreds, err := s.mfaRepo.ListWebAuthnByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	for _, c := range webAuthnCreds {
		methods = append(methods, model.MFAMethodResponse{
			ID:                  c.ID,
			Type:                "webauthn",
			Name:                c.Name,
			Verified:            true, // WebAuthn credentials are verified upon creation
			PasswordlessEnabled: c.PasswordlessEnabled,
			LastUsedAt:          c.LastUsedAt,
			CreatedAt:           c.CreatedAt,
		})
	}

	return methods, nil
}

// DeleteMethod removes an MFA method. If no verified methods remain, MFA is disabled.
func (s *MFAService) DeleteMethod(ctx context.Context, userID uuid.UUID, methodID uuid.UUID) error {
	// Try TOTP first
	totpMethod, err := s.mfaRepo.GetTOTPByID(ctx, methodID)
	if err == nil && totpMethod.UserID == userID {
		if err := s.mfaRepo.DeleteTOTP(ctx, methodID); err != nil {
			return err
		}
		return s.maybeDisableMFA(ctx, userID)
	}

	// Try WebAuthn
	webAuthnCred, err := s.mfaRepo.GetWebAuthnByID(ctx, methodID)
	if err == nil && webAuthnCred.UserID == userID {
		if err := s.mfaRepo.DeleteWebAuthn(ctx, methodID); err != nil {
			return err
		}
		return s.maybeDisableMFA(ctx, userID)
	}

	return model.ErrNotFound("mfa method")
}

// AdminResetMFA removes all MFA methods and recovery codes for a user.
func (s *MFAService) AdminResetMFA(ctx context.Context, userID uuid.UUID) error {
	if err := s.mfaRepo.DeleteAllTOTPForUser(ctx, userID); err != nil {
		return err
	}
	if err := s.mfaRepo.DeleteAllWebAuthnForUser(ctx, userID); err != nil {
		return err
	}
	if err := s.mfaRepo.DeleteAllRecoveryCodesForUser(ctx, userID); err != nil {
		return err
	}
	return s.userRepo.SetMFAEnabled(ctx, userID, false)
}

// ---------------------------------------------------------------------------
// MFA challenge tokens (Redis)
// ---------------------------------------------------------------------------

// CreateMFAToken generates an MFA challenge token and stores it in Redis.
func (s *MFAService) CreateMFAToken(ctx context.Context, userID uuid.UUID) (*model.MFAChallengeResponse, error) {
	// Determine available methods
	var methods []string

	hasTOTP, err := s.mfaRepo.HasVerifiedTOTP(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasTOTP {
		methods = append(methods, "totp")
	}

	hasWebAuthn, err := s.mfaRepo.HasWebAuthn(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasWebAuthn {
		methods = append(methods, "webauthn")
	}

	unusedCodes, err := s.mfaRepo.CountUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	if unusedCodes > 0 {
		methods = append(methods, "recovery")
	}

	token := uuid.New().String()
	session := model.MFASession{
		UserID:  userID,
		Methods: methods,
	}

	data, err := json.Marshal(session)
	if err != nil {
		return nil, fmt.Errorf("marshal mfa session: %w", err)
	}

	key := mfaTokenPrefix + token
	if err := s.rdb.Set(ctx, key, data, mfaTokenTTL).Err(); err != nil {
		return nil, fmt.Errorf("store mfa token: %w", err)
	}

	return &model.MFAChallengeResponse{
		MFARequired: true,
		MFAToken:    token,
		Methods:     methods,
	}, nil
}

// ValidateMFAToken retrieves and deletes an MFA token from Redis.
func (s *MFAService) ValidateMFAToken(ctx context.Context, token string) (*model.MFASession, error) {
	key := mfaTokenPrefix + token
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, model.ErrValidation("invalid or expired MFA token")
	}
	s.rdb.Del(ctx, key)

	var session model.MFASession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal mfa session: %w", err)
	}
	return &session, nil
}

// PeekMFAToken retrieves an MFA token from Redis without consuming it.
// Used by WebAuthn begin flow where we need the user ID but can't consume
// the token yet (it's consumed in the finish step).
func (s *MFAService) PeekMFAToken(ctx context.Context, token string) (*model.MFASession, error) {
	key := mfaTokenPrefix + token
	data, err := s.rdb.Get(ctx, key).Bytes()
	if err != nil {
		return nil, model.ErrValidation("invalid or expired MFA token")
	}

	var session model.MFASession
	if err := json.Unmarshal(data, &session); err != nil {
		return nil, fmt.Errorf("unmarshal mfa session: %w", err)
	}
	return &session, nil
}

// GetAvailableMethods returns available MFA methods for a user (used by login flow).
func (s *MFAService) GetAvailableMethods(ctx context.Context, userID uuid.UUID) ([]string, error) {
	var methods []string

	hasTOTP, err := s.mfaRepo.HasVerifiedTOTP(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasTOTP {
		methods = append(methods, "totp")
	}

	hasWebAuthn, err := s.mfaRepo.HasWebAuthn(ctx, userID)
	if err != nil {
		return nil, err
	}
	if hasWebAuthn {
		methods = append(methods, "webauthn")
	}

	unusedCodes, err := s.mfaRepo.CountUnusedRecoveryCodes(ctx, userID)
	if err != nil {
		return nil, err
	}
	if unusedCodes > 0 {
		methods = append(methods, "recovery")
	}

	return methods, nil
}

// ---------------------------------------------------------------------------
// Internal helpers
// ---------------------------------------------------------------------------

// enableMFAAndGenerateCodes enables MFA on the user and generates recovery codes
// if this is the first verified method.
func (s *MFAService) enableMFAAndGenerateCodes(ctx context.Context, userID uuid.UUID) (*model.RecoveryCodesResponse, error) {
	user, err := s.userRepo.GetByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if !user.MFAEnabled {
		if err := s.userRepo.SetMFAEnabled(ctx, userID, true); err != nil {
			return nil, err
		}
		// Generate recovery codes on first MFA setup
		return s.GenerateRecoveryCodes(ctx, userID)
	}

	return nil, nil
}

// maybeDisableMFA disables MFA if the user has no more verified methods.
func (s *MFAService) maybeDisableMFA(ctx context.Context, userID uuid.UUID) error {
	count, err := s.mfaRepo.CountVerifiedMethods(ctx, userID)
	if err != nil {
		return err
	}
	if count == 0 {
		if err := s.userRepo.SetMFAEnabled(ctx, userID, false); err != nil {
			return err
		}
		// Also clean up recovery codes since MFA is now off
		return s.mfaRepo.DeleteAllRecoveryCodesForUser(ctx, userID)
	}
	return nil
}

// generateRecoveryCode generates a random 8-character alphanumeric code
// formatted as xxxx-xxxx for readability.
func generateRecoveryCode() string {
	b := make([]byte, 6) // 6 bytes = enough entropy for 8 base32 chars
	if _, err := rand.Read(b); err != nil {
		panic("crypto/rand failed: " + err.Error())
	}
	raw := strings.ToLower(base64.RawStdEncoding.EncodeToString(b))
	// Take first 8 chars, insert dash in the middle
	if len(raw) > 8 {
		raw = raw[:8]
	}
	return raw[:4] + "-" + raw[4:]
}
