-- Add is_primary column to devices table (default false).
ALTER TABLE devices ADD COLUMN is_primary BOOLEAN NOT NULL DEFAULT false;

-- Backfill: set the oldest device per user as primary.
UPDATE devices d SET is_primary = true
FROM (
    SELECT DISTINCT ON (user_id) id
    FROM devices
    ORDER BY user_id, created_at ASC
) first_devices
WHERE d.id = first_devices.id;

-- Enforce at most one primary device per user.
CREATE UNIQUE INDEX idx_devices_user_primary ON devices(user_id) WHERE is_primary = true;
