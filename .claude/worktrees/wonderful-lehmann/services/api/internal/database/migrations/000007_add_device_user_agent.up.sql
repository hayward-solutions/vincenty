-- Add user_agent column to devices table for heuristic device recognition.
ALTER TABLE devices ADD COLUMN user_agent TEXT;

-- Index for the heuristic lookup: find a user's web device by user-agent.
CREATE INDEX idx_devices_user_agent_lookup
    ON devices (user_id, device_type, user_agent)
    WHERE user_agent IS NOT NULL;
