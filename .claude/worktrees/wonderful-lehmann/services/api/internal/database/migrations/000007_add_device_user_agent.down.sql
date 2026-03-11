DROP INDEX IF EXISTS idx_devices_user_agent_lookup;
ALTER TABLE devices DROP COLUMN IF EXISTS user_agent;
