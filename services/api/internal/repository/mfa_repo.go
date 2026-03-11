package repository

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/vincenty/api/internal/model"
)

// MFARepository handles database operations for MFA methods, WebAuthn
// credentials, and recovery codes.
type MFARepository struct {
	pool *pgxpool.Pool
}

// NewMFARepository creates a new MFARepository.
func NewMFARepository(pool *pgxpool.Pool) *MFARepository {
	return &MFARepository{pool: pool}
}

// ---------------------------------------------------------------------------
// TOTP methods
// ---------------------------------------------------------------------------

// CreateTOTP inserts a new (unverified) TOTP method.
func (r *MFARepository) CreateTOTP(ctx context.Context, m *model.TOTPMethod) error {
	query := `
		INSERT INTO user_totp_methods (id, user_id, name, secret_encrypted, verified)
		VALUES ($1, $2, $3, $4, $5)
		RETURNING created_at, updated_at`

	if m.ID == uuid.Nil {
		m.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		m.ID, m.UserID, m.Name, m.SecretEncrypted, m.Verified,
	).Scan(&m.CreatedAt, &m.UpdatedAt)
}

// GetTOTPByID retrieves a TOTP method by ID.
func (r *MFARepository) GetTOTPByID(ctx context.Context, id uuid.UUID) (*model.TOTPMethod, error) {
	query := `
		SELECT id, user_id, name, secret_encrypted, verified, last_used_at, created_at, updated_at
		FROM user_totp_methods WHERE id = $1`

	m := &model.TOTPMethod{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&m.ID, &m.UserID, &m.Name, &m.SecretEncrypted, &m.Verified,
		&m.LastUsedAt, &m.CreatedAt, &m.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("totp method")
		}
		return nil, err
	}
	return m, nil
}

// ListTOTPByUser returns all TOTP methods for a user.
func (r *MFARepository) ListTOTPByUser(ctx context.Context, userID uuid.UUID) ([]model.TOTPMethod, error) {
	query := `
		SELECT id, user_id, name, secret_encrypted, verified, last_used_at, created_at, updated_at
		FROM user_totp_methods WHERE user_id = $1 ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var methods []model.TOTPMethod
	for rows.Next() {
		var m model.TOTPMethod
		if err := rows.Scan(
			&m.ID, &m.UserID, &m.Name, &m.SecretEncrypted, &m.Verified,
			&m.LastUsedAt, &m.CreatedAt, &m.UpdatedAt,
		); err != nil {
			return nil, err
		}
		methods = append(methods, m)
	}
	return methods, rows.Err()
}

// VerifyTOTP marks a TOTP method as verified.
func (r *MFARepository) VerifyTOTP(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_totp_methods SET verified = true, updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("totp method")
	}
	return nil
}

// TouchTOTP updates the last_used_at timestamp.
func (r *MFARepository) TouchTOTP(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE user_totp_methods SET last_used_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}

// DeleteTOTP removes a TOTP method by ID.
func (r *MFARepository) DeleteTOTP(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM user_totp_methods WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("totp method")
	}
	return nil
}

// DeleteAllTOTPForUser removes all TOTP methods for a user.
func (r *MFARepository) DeleteAllTOTPForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM user_totp_methods WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// ---------------------------------------------------------------------------
// WebAuthn credentials
// ---------------------------------------------------------------------------

// CreateWebAuthn inserts a new WebAuthn credential.
func (r *MFARepository) CreateWebAuthn(ctx context.Context, c *model.WebAuthnCredential) error {
	query := `
		INSERT INTO webauthn_credentials (id, user_id, name, credential_id, public_key, aaguid, sign_count, transports, backup_eligible, backup_state, passwordless_enabled)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
		RETURNING created_at, updated_at`

	if c.ID == uuid.Nil {
		c.ID = uuid.New()
	}

	return r.pool.QueryRow(ctx, query,
		c.ID, c.UserID, c.Name, c.CredentialID, c.PublicKey,
		c.AAGUID, c.SignCount, c.Transports, c.BackupEligible, c.BackupState, c.PasswordlessEnabled,
	).Scan(&c.CreatedAt, &c.UpdatedAt)
}

// GetWebAuthnByID retrieves a WebAuthn credential by ID.
func (r *MFARepository) GetWebAuthnByID(ctx context.Context, id uuid.UUID) (*model.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, name, credential_id, public_key, aaguid, sign_count,
		       transports, backup_eligible, backup_state, passwordless_enabled, last_used_at, created_at, updated_at
		FROM webauthn_credentials WHERE id = $1`

	c := &model.WebAuthnCredential{}
	err := r.pool.QueryRow(ctx, query, id).Scan(
		&c.ID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey,
		&c.AAGUID, &c.SignCount, &c.Transports, &c.BackupEligible, &c.BackupState, &c.PasswordlessEnabled,
		&c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("webauthn credential")
		}
		return nil, err
	}
	return c, nil
}

// GetWebAuthnByCredentialID looks up a credential by its raw credential ID.
func (r *MFARepository) GetWebAuthnByCredentialID(ctx context.Context, credentialID []byte) (*model.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, name, credential_id, public_key, aaguid, sign_count,
		       transports, backup_eligible, backup_state, passwordless_enabled, last_used_at, created_at, updated_at
		FROM webauthn_credentials WHERE credential_id = $1`

	c := &model.WebAuthnCredential{}
	err := r.pool.QueryRow(ctx, query, credentialID).Scan(
		&c.ID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey,
		&c.AAGUID, &c.SignCount, &c.Transports, &c.BackupEligible, &c.BackupState, &c.PasswordlessEnabled,
		&c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, model.ErrNotFound("webauthn credential")
		}
		return nil, err
	}
	return c, nil
}

// ListWebAuthnByUser returns all WebAuthn credentials for a user.
func (r *MFARepository) ListWebAuthnByUser(ctx context.Context, userID uuid.UUID) ([]model.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, name, credential_id, public_key, aaguid, sign_count,
		       transports, backup_eligible, backup_state, passwordless_enabled, last_used_at, created_at, updated_at
		FROM webauthn_credentials WHERE user_id = $1 ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []model.WebAuthnCredential
	for rows.Next() {
		var c model.WebAuthnCredential
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey,
			&c.AAGUID, &c.SignCount, &c.Transports, &c.BackupEligible, &c.BackupState, &c.PasswordlessEnabled,
			&c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// ListPasswordlessCredentials returns all WebAuthn credentials with passwordless enabled.
func (r *MFARepository) ListPasswordlessCredentials(ctx context.Context) ([]model.WebAuthnCredential, error) {
	query := `
		SELECT id, user_id, name, credential_id, public_key, aaguid, sign_count,
		       transports, backup_eligible, backup_state, passwordless_enabled, last_used_at, created_at, updated_at
		FROM webauthn_credentials WHERE passwordless_enabled = true`

	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var creds []model.WebAuthnCredential
	for rows.Next() {
		var c model.WebAuthnCredential
		if err := rows.Scan(
			&c.ID, &c.UserID, &c.Name, &c.CredentialID, &c.PublicKey,
			&c.AAGUID, &c.SignCount, &c.Transports, &c.BackupEligible, &c.BackupState, &c.PasswordlessEnabled,
			&c.LastUsedAt, &c.CreatedAt, &c.UpdatedAt,
		); err != nil {
			return nil, err
		}
		creds = append(creds, c)
	}
	return creds, rows.Err()
}

// UpdateWebAuthnSignCount updates the sign count, backup state, and last_used_at after a successful assertion.
func (r *MFARepository) UpdateWebAuthnSignCount(ctx context.Context, id uuid.UUID, signCount int64, backupState bool) error {
	query := `UPDATE webauthn_credentials SET sign_count = $2, backup_state = $3, last_used_at = NOW(), updated_at = NOW() WHERE id = $1`
	_, err := r.pool.Exec(ctx, query, id, signCount, backupState)
	return err
}

// UpdateWebAuthnPasswordless toggles the passwordless_enabled flag.
func (r *MFARepository) UpdateWebAuthnPasswordless(ctx context.Context, id uuid.UUID, enabled bool) error {
	query := `UPDATE webauthn_credentials SET passwordless_enabled = $2, updated_at = NOW() WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id, enabled)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("webauthn credential")
	}
	return nil
}

// DeleteWebAuthn removes a WebAuthn credential by ID.
func (r *MFARepository) DeleteWebAuthn(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM webauthn_credentials WHERE id = $1`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("webauthn credential")
	}
	return nil
}

// DeleteAllWebAuthnForUser removes all WebAuthn credentials for a user.
func (r *MFARepository) DeleteAllWebAuthnForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM webauthn_credentials WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// ---------------------------------------------------------------------------
// Recovery codes
// ---------------------------------------------------------------------------

// CreateRecoveryCodes inserts a batch of recovery codes for a user,
// replacing any existing codes.
func (r *MFARepository) CreateRecoveryCodes(ctx context.Context, userID uuid.UUID, hashes []string) error {
	// Delete existing codes first
	if _, err := r.pool.Exec(ctx, `DELETE FROM recovery_codes WHERE user_id = $1`, userID); err != nil {
		return err
	}

	for _, h := range hashes {
		code := &model.RecoveryCode{
			ID:       uuid.New(),
			UserID:   userID,
			CodeHash: h,
		}
		_, err := r.pool.Exec(ctx,
			`INSERT INTO recovery_codes (id, user_id, code_hash) VALUES ($1, $2, $3)`,
			code.ID, code.UserID, code.CodeHash,
		)
		if err != nil {
			return err
		}
	}
	return nil
}

// ListUnusedRecoveryCodes returns all unused recovery codes for a user.
func (r *MFARepository) ListUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) ([]model.RecoveryCode, error) {
	query := `
		SELECT id, user_id, code_hash, used_at, created_at
		FROM recovery_codes
		WHERE user_id = $1 AND used_at IS NULL
		ORDER BY created_at`

	rows, err := r.pool.Query(ctx, query, userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var codes []model.RecoveryCode
	for rows.Next() {
		var c model.RecoveryCode
		if err := rows.Scan(&c.ID, &c.UserID, &c.CodeHash, &c.UsedAt, &c.CreatedAt); err != nil {
			return nil, err
		}
		codes = append(codes, c)
	}
	return codes, rows.Err()
}

// MarkRecoveryCodeUsed marks a recovery code as used.
func (r *MFARepository) MarkRecoveryCodeUsed(ctx context.Context, id uuid.UUID) error {
	query := `UPDATE recovery_codes SET used_at = NOW() WHERE id = $1 AND used_at IS NULL`
	tag, err := r.pool.Exec(ctx, query, id)
	if err != nil {
		return err
	}
	if tag.RowsAffected() == 0 {
		return model.ErrNotFound("recovery code")
	}
	return nil
}

// DeleteAllRecoveryCodesForUser removes all recovery codes for a user.
func (r *MFARepository) DeleteAllRecoveryCodesForUser(ctx context.Context, userID uuid.UUID) error {
	query := `DELETE FROM recovery_codes WHERE user_id = $1`
	_, err := r.pool.Exec(ctx, query, userID)
	return err
}

// CountUnusedRecoveryCodes returns the number of unused recovery codes.
func (r *MFARepository) CountUnusedRecoveryCodes(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `SELECT COUNT(*) FROM recovery_codes WHERE user_id = $1 AND used_at IS NULL`
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// ---------------------------------------------------------------------------
// Aggregate helpers
// ---------------------------------------------------------------------------

// CountVerifiedMethods returns the total number of verified MFA methods
// (TOTP + WebAuthn) for a user.
func (r *MFARepository) CountVerifiedMethods(ctx context.Context, userID uuid.UUID) (int, error) {
	var count int
	query := `
		SELECT (
			SELECT COUNT(*) FROM user_totp_methods WHERE user_id = $1 AND verified = true
		) + (
			SELECT COUNT(*) FROM webauthn_credentials WHERE user_id = $1
		)`
	err := r.pool.QueryRow(ctx, query, userID).Scan(&count)
	return count, err
}

// HasVerifiedTOTP checks if the user has at least one verified TOTP method.
func (r *MFARepository) HasVerifiedTOTP(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM user_totp_methods WHERE user_id = $1 AND verified = true)`
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	return exists, err
}

// HasWebAuthn checks if the user has at least one WebAuthn credential.
func (r *MFARepository) HasWebAuthn(ctx context.Context, userID uuid.UUID) (bool, error) {
	var exists bool
	query := `SELECT EXISTS(SELECT 1 FROM webauthn_credentials WHERE user_id = $1)`
	err := r.pool.QueryRow(ctx, query, userID).Scan(&exists)
	return exists, err
}
