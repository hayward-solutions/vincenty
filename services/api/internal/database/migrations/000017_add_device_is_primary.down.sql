DROP INDEX IF EXISTS idx_devices_user_primary;
ALTER TABLE devices DROP COLUMN IF EXISTS is_primary;
