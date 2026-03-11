package service

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/auth"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/repository/mock"
)

// testMFAService creates an MFAService with mock repos and a local encryptor.
// Redis and WebAuthn are nil — only methods that don't touch them are testable.
func testMFAService(mfaRepo *mock.MFARepo, userRepo *mock.UserRepo) *MFAService {
	enc, _ := auth.NewLocalEncryptor([]byte("test-secret-key-that-is-long-enough"))
	return NewMFAService(mfaRepo, userRepo, enc, nil, nil)
}

// ---------------------------------------------------------------------------
// ListMethods
// ---------------------------------------------------------------------------

func TestMFAService_ListMethods(t *testing.T) {
	userID := uuid.New()
	totpID := uuid.New()
	waID := uuid.New()
	now := time.Now()

	mfaRepo := &mock.MFARepo{
		ListTOTPByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.TOTPMethod, error) {
			return []model.TOTPMethod{
				{ID: totpID, UserID: userID, Name: "My App", Verified: true, CreatedAt: now},
				{ID: uuid.New(), UserID: userID, Name: "Unverified", Verified: false, CreatedAt: now},
			}, nil
		},
		ListWebAuthnByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.WebAuthnCredential, error) {
			return []model.WebAuthnCredential{
				{ID: waID, UserID: userID, Name: "YubiKey", PasswordlessEnabled: true, CreatedAt: now},
			}, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	methods, err := svc.ListMethods(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should include 1 verified TOTP + 1 WebAuthn = 2 methods (unverified TOTP excluded)
	if len(methods) != 2 {
		t.Fatalf("expected 2 methods, got %d", len(methods))
	}
	if methods[0].Type != "totp" || methods[0].Name != "My App" {
		t.Errorf("expected first method to be TOTP 'My App', got %s %s", methods[0].Type, methods[0].Name)
	}
	if methods[1].Type != "webauthn" || methods[1].Name != "YubiKey" {
		t.Errorf("expected second method to be WebAuthn 'YubiKey', got %s %s", methods[1].Type, methods[1].Name)
	}
	if !methods[1].PasswordlessEnabled {
		t.Error("expected YubiKey to have passwordless enabled")
	}
}

func TestMFAService_ListMethods_Empty(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		ListTOTPByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.TOTPMethod, error) {
			return nil, nil
		},
		ListWebAuthnByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.WebAuthnCredential, error) {
			return nil, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	methods, err := svc.ListMethods(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 0 {
		t.Errorf("expected 0 methods, got %d", len(methods))
	}
}

// ---------------------------------------------------------------------------
// DeleteMethod
// ---------------------------------------------------------------------------

func TestMFAService_DeleteMethod_TOTP(t *testing.T) {
	userID := uuid.New()
	methodID := uuid.New()
	deleted := false

	mfaRepo := &mock.MFARepo{
		GetTOTPByIDFn: func(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
			if id == methodID {
				return &model.TOTPMethod{ID: methodID, UserID: userID, Verified: true}, nil
			}
			return nil, model.ErrNotFound("totp method")
		},
		DeleteTOTPFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
		CountVerifiedMethodsFn: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil // No more methods after deletion
		},
		DeleteAllRecoveryCodesForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			return nil
		},
	}
	userRepo := &mock.UserRepo{
		SetMFAEnabledFn: func(ctx context.Context, id uuid.UUID, enabled bool) error {
			if enabled {
				t.Error("expected MFA to be disabled")
			}
			return nil
		},
	}
	svc := testMFAService(mfaRepo, userRepo)

	err := svc.DeleteMethod(context.Background(), userID, methodID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected TOTP to be deleted")
	}
}

func TestMFAService_DeleteMethod_WebAuthn(t *testing.T) {
	userID := uuid.New()
	methodID := uuid.New()
	deleted := false

	mfaRepo := &mock.MFARepo{
		GetTOTPByIDFn: func(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
			return nil, model.ErrNotFound("totp method")
		},
		GetWebAuthnByIDFn: func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
			if id == methodID {
				return &model.WebAuthnCredential{ID: methodID, UserID: userID}, nil
			}
			return nil, model.ErrNotFound("webauthn credential")
		},
		DeleteWebAuthnFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
		CountVerifiedMethodsFn: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 1, nil // Still has another method
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.DeleteMethod(context.Background(), userID, methodID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !deleted {
		t.Error("expected WebAuthn credential to be deleted")
	}
}

func TestMFAService_DeleteMethod_NotFound(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		GetTOTPByIDFn: func(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
			return nil, model.ErrNotFound("totp method")
		},
		GetWebAuthnByIDFn: func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
			return nil, model.ErrNotFound("webauthn credential")
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.DeleteMethod(context.Background(), uuid.New(), uuid.New())
	var nf *model.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

func TestMFAService_DeleteMethod_WrongUser(t *testing.T) {
	otherUserID := uuid.New()
	methodID := uuid.New()

	mfaRepo := &mock.MFARepo{
		GetTOTPByIDFn: func(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
			return &model.TOTPMethod{ID: methodID, UserID: uuid.New()}, nil // Different user
		},
		GetWebAuthnByIDFn: func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
			return &model.WebAuthnCredential{ID: methodID, UserID: uuid.New()}, nil // Different user
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.DeleteMethod(context.Background(), otherUserID, methodID)
	var nf *model.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// AdminResetMFA
// ---------------------------------------------------------------------------

func TestMFAService_AdminResetMFA(t *testing.T) {
	userID := uuid.New()
	totpDeleted, waDeleted, codesDeleted, mfaDisabled := false, false, false, false

	mfaRepo := &mock.MFARepo{
		DeleteAllTOTPForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			totpDeleted = true
			return nil
		},
		DeleteAllWebAuthnForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			waDeleted = true
			return nil
		},
		DeleteAllRecoveryCodesForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			codesDeleted = true
			return nil
		},
	}
	userRepo := &mock.UserRepo{
		SetMFAEnabledFn: func(ctx context.Context, id uuid.UUID, enabled bool) error {
			if enabled {
				t.Error("expected MFA to be disabled")
			}
			mfaDisabled = true
			return nil
		},
	}
	svc := testMFAService(mfaRepo, userRepo)

	err := svc.AdminResetMFA(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !totpDeleted {
		t.Error("expected TOTP methods to be deleted")
	}
	if !waDeleted {
		t.Error("expected WebAuthn credentials to be deleted")
	}
	if !codesDeleted {
		t.Error("expected recovery codes to be deleted")
	}
	if !mfaDisabled {
		t.Error("expected MFA to be disabled")
	}
}

// ---------------------------------------------------------------------------
// TogglePasswordless
// ---------------------------------------------------------------------------

func TestMFAService_TogglePasswordless(t *testing.T) {
	userID := uuid.New()
	credID := uuid.New()
	var updatedEnabled bool

	mfaRepo := &mock.MFARepo{
		GetWebAuthnByIDFn: func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
			if id == credID {
				return &model.WebAuthnCredential{ID: credID, UserID: userID}, nil
			}
			return nil, model.ErrNotFound("webauthn credential")
		},
		UpdateWebAuthnPasswordlessFn: func(ctx context.Context, id uuid.UUID, enabled bool) error {
			updatedEnabled = enabled
			return nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.TogglePasswordless(context.Background(), userID, credID, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !updatedEnabled {
		t.Error("expected passwordless to be enabled")
	}
}

func TestMFAService_TogglePasswordless_WrongUser(t *testing.T) {
	credID := uuid.New()
	mfaRepo := &mock.MFARepo{
		GetWebAuthnByIDFn: func(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
			return &model.WebAuthnCredential{ID: credID, UserID: uuid.New()}, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.TogglePasswordless(context.Background(), uuid.New(), credID, true)
	var nf *model.NotFoundError
	if !errors.As(err, &nf) {
		t.Fatalf("expected NotFoundError, got %T: %v", err, err)
	}
}

// ---------------------------------------------------------------------------
// GetAvailableMethods
// ---------------------------------------------------------------------------

func TestMFAService_GetAvailableMethods(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		HasVerifiedTOTPFn:          func(ctx context.Context, uid uuid.UUID) (bool, error) { return true, nil },
		HasWebAuthnFn:              func(ctx context.Context, uid uuid.UUID) (bool, error) { return true, nil },
		CountUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) (int, error) { return 5, nil },
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	methods, err := svc.GetAvailableMethods(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 3 {
		t.Fatalf("expected 3 methods, got %d: %v", len(methods), methods)
	}
	expected := map[string]bool{"totp": true, "webauthn": true, "recovery": true}
	for _, m := range methods {
		if !expected[m] {
			t.Errorf("unexpected method: %s", m)
		}
	}
}

func TestMFAService_GetAvailableMethods_OnlyTOTP(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		HasVerifiedTOTPFn:          func(ctx context.Context, uid uuid.UUID) (bool, error) { return true, nil },
		HasWebAuthnFn:              func(ctx context.Context, uid uuid.UUID) (bool, error) { return false, nil },
		CountUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) (int, error) { return 0, nil },
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	methods, err := svc.GetAvailableMethods(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 1 || methods[0] != "totp" {
		t.Errorf("expected [totp], got %v", methods)
	}
}

func TestMFAService_GetAvailableMethods_None(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		HasVerifiedTOTPFn:          func(ctx context.Context, uid uuid.UUID) (bool, error) { return false, nil },
		HasWebAuthnFn:              func(ctx context.Context, uid uuid.UUID) (bool, error) { return false, nil },
		CountUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) (int, error) { return 0, nil },
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	methods, err := svc.GetAvailableMethods(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(methods) != 0 {
		t.Errorf("expected empty, got %v", methods)
	}
}

// ---------------------------------------------------------------------------
// ValidateRecoveryCode
// ---------------------------------------------------------------------------

func TestMFAService_ValidateRecoveryCode_Valid(t *testing.T) {
	userID := uuid.New()
	codeID := uuid.New()

	// Generate a known recovery code hash using bcrypt
	// The code "abcd-efgh" normalized is "abcdefgh"
	code := "abcd-efgh"
	normalized := "abcdefgh"
	hash, err := bcryptHash(normalized)
	if err != nil {
		t.Fatalf("failed to hash: %v", err)
	}

	markedUsed := false
	mfaRepo := &mock.MFARepo{
		ListUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) ([]model.RecoveryCode, error) {
			return []model.RecoveryCode{
				{ID: codeID, UserID: userID, CodeHash: hash},
			}, nil
		},
		MarkRecoveryCodeUsedFn: func(ctx context.Context, id uuid.UUID) error {
			if id == codeID {
				markedUsed = true
			}
			return nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	valid, err := svc.ValidateRecoveryCode(context.Background(), userID, code)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !valid {
		t.Error("expected code to be valid")
	}
	if !markedUsed {
		t.Error("expected code to be marked as used")
	}
}

func TestMFAService_ValidateRecoveryCode_Invalid(t *testing.T) {
	userID := uuid.New()
	hash, _ := bcryptHash("realcode1")

	mfaRepo := &mock.MFARepo{
		ListUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) ([]model.RecoveryCode, error) {
			return []model.RecoveryCode{
				{ID: uuid.New(), UserID: userID, CodeHash: hash},
			}, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	valid, err := svc.ValidateRecoveryCode(context.Background(), userID, "wrong-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected code to be invalid")
	}
}

func TestMFAService_ValidateRecoveryCode_NoCodes(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		ListUnusedRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID) ([]model.RecoveryCode, error) {
			return nil, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	valid, err := svc.ValidateRecoveryCode(context.Background(), uuid.New(), "some-code")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected code to be invalid with no codes")
	}
}

// ---------------------------------------------------------------------------
// ValidateTOTP
// ---------------------------------------------------------------------------

func TestMFAService_ValidateTOTP_NoVerifiedMethods(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		ListTOTPByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.TOTPMethod, error) {
			return []model.TOTPMethod{
				{ID: uuid.New(), Verified: false, SecretEncrypted: []byte("encrypted")},
			}, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	valid, err := svc.ValidateTOTP(context.Background(), uuid.New(), "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid: no verified methods")
	}
}

func TestMFAService_ValidateTOTP_NoMethods(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		ListTOTPByUserFn: func(ctx context.Context, uid uuid.UUID) ([]model.TOTPMethod, error) {
			return nil, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	valid, err := svc.ValidateTOTP(context.Background(), uuid.New(), "123456")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if valid {
		t.Error("expected invalid: no methods")
	}
}

// ---------------------------------------------------------------------------
// enableMFAAndGenerateCodes (tested indirectly through GenerateRecoveryCodes)
// ---------------------------------------------------------------------------

func TestMFAService_GenerateRecoveryCodes(t *testing.T) {
	userID := uuid.New()
	var storedHashes []string

	mfaRepo := &mock.MFARepo{
		CreateRecoveryCodesFn: func(ctx context.Context, uid uuid.UUID, hashes []string) error {
			storedHashes = hashes
			return nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	resp, err := svc.GenerateRecoveryCodes(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(resp.Codes) != 8 {
		t.Errorf("expected 8 recovery codes, got %d", len(resp.Codes))
	}
	if len(storedHashes) != 8 {
		t.Errorf("expected 8 hashes stored, got %d", len(storedHashes))
	}

	// Each code should be in xxxx-xxxx format
	for _, code := range resp.Codes {
		if len(code) != 9 { // 4 + dash + 4
			t.Errorf("expected code length 9 (xxxx-xxxx), got %d: %s", len(code), code)
		}
		if code[4] != '-' {
			t.Errorf("expected dash at position 4, got %c in %s", code[4], code)
		}
	}

	// Verify all codes are unique
	seen := make(map[string]bool)
	for _, code := range resp.Codes {
		if seen[code] {
			t.Errorf("duplicate recovery code: %s", code)
		}
		seen[code] = true
	}
}

// ---------------------------------------------------------------------------
// maybeDisableMFA
// ---------------------------------------------------------------------------

func TestMFAService_MaybeDisableMFA_DisablesWhenNoMethods(t *testing.T) {
	userID := uuid.New()
	mfaDisabled := false
	codesDeleted := false

	mfaRepo := &mock.MFARepo{
		CountVerifiedMethodsFn: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 0, nil
		},
		DeleteAllRecoveryCodesForUserFn: func(ctx context.Context, uid uuid.UUID) error {
			codesDeleted = true
			return nil
		},
	}
	userRepo := &mock.UserRepo{
		SetMFAEnabledFn: func(ctx context.Context, id uuid.UUID, enabled bool) error {
			if !enabled {
				mfaDisabled = true
			}
			return nil
		},
	}
	svc := testMFAService(mfaRepo, userRepo)

	err := svc.maybeDisableMFA(context.Background(), userID)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !mfaDisabled {
		t.Error("expected MFA to be disabled")
	}
	if !codesDeleted {
		t.Error("expected recovery codes to be deleted")
	}
}

func TestMFAService_MaybeDisableMFA_KeepsEnabledWithMethods(t *testing.T) {
	mfaRepo := &mock.MFARepo{
		CountVerifiedMethodsFn: func(ctx context.Context, uid uuid.UUID) (int, error) {
			return 2, nil
		},
	}
	svc := testMFAService(mfaRepo, &mock.UserRepo{})

	err := svc.maybeDisableMFA(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// No SetMFAEnabled call expected — if it panics, there's a bug
}

// ---------------------------------------------------------------------------
// webAuthnUser adapter
// ---------------------------------------------------------------------------

func TestWebAuthnUser_Methods(t *testing.T) {
	displayName := "Alice Smith"
	user := &model.User{
		ID:          uuid.New(),
		Username:    "alice",
		DisplayName: &displayName,
	}
	creds := []model.WebAuthnCredential{
		{
			CredentialID:   []byte("cred-1"),
			PublicKey:      []byte("pk-1"),
			Transports:     []string{"usb", "nfc"},
			BackupEligible: true,
			BackupState:    false,
			AAGUID:         []byte("aaguid-1234-5678"),
			SignCount:      42,
		},
	}

	waUser := &webAuthnUser{user: user, creds: creds}

	if string(waUser.WebAuthnID()) != string(user.ID[:]) {
		t.Error("WebAuthnID mismatch")
	}
	if waUser.WebAuthnName() != "alice" {
		t.Errorf("expected name alice, got %s", waUser.WebAuthnName())
	}
	if waUser.WebAuthnDisplayName() != "Alice Smith" {
		t.Errorf("expected display name 'Alice Smith', got %s", waUser.WebAuthnDisplayName())
	}
	if waUser.WebAuthnIcon() != "" {
		t.Error("expected empty icon")
	}

	waCreds := waUser.WebAuthnCredentials()
	if len(waCreds) != 1 {
		t.Fatalf("expected 1 credential, got %d", len(waCreds))
	}
	if string(waCreds[0].ID) != "cred-1" {
		t.Errorf("credential ID mismatch")
	}
	if len(waCreds[0].Transport) != 2 {
		t.Errorf("expected 2 transports, got %d", len(waCreds[0].Transport))
	}
	if !waCreds[0].Flags.BackupEligible {
		t.Error("expected backup eligible")
	}
	if waCreds[0].Authenticator.SignCount != 42 {
		t.Errorf("expected sign count 42, got %d", waCreds[0].Authenticator.SignCount)
	}
}

func TestWebAuthnUser_DisplayNameFallsBackToUsername(t *testing.T) {
	user := &model.User{
		Username:    "bob",
		DisplayName: nil,
	}
	waUser := &webAuthnUser{user: user, creds: nil}

	if waUser.WebAuthnDisplayName() != "bob" {
		t.Errorf("expected display name to fall back to username, got %s", waUser.WebAuthnDisplayName())
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func bcryptHash(plaintext string) (string, error) {
	// Import bcrypt via auth package — use the same cost
	hash, err := auth.HashPassword(plaintext)
	if err != nil {
		return "", err
	}
	return hash, nil
}
