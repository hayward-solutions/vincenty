-- Add credential flags to webauthn_credentials (required by go-webauthn v0.15+ login validation).
ALTER TABLE webauthn_credentials ADD COLUMN backup_eligible BOOLEAN NOT NULL DEFAULT FALSE;
ALTER TABLE webauthn_credentials ADD COLUMN backup_state BOOLEAN NOT NULL DEFAULT FALSE;
