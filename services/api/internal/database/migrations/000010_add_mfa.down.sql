DROP TABLE IF EXISTS server_settings;
DROP TABLE IF EXISTS recovery_codes;
DROP TABLE IF EXISTS webauthn_credentials;
DROP TABLE IF EXISTS user_totp_methods;
ALTER TABLE users DROP COLUMN IF EXISTS mfa_enabled;
