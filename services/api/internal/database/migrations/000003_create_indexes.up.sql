-- =========================================================================
-- Devices
-- =========================================================================
CREATE INDEX idx_devices_user_id ON devices(user_id);

-- =========================================================================
-- Group Members
-- =========================================================================
CREATE INDEX idx_group_members_group_id ON group_members(group_id);
CREATE INDEX idx_group_members_user_id ON group_members(user_id);

-- =========================================================================
-- Messages
-- =========================================================================
CREATE INDEX idx_messages_group_id ON messages(group_id);
CREATE INDEX idx_messages_recipient_id ON messages(recipient_id);
CREATE INDEX idx_messages_sender_id ON messages(sender_id);
CREATE INDEX idx_messages_created_at ON messages(created_at DESC);

-- =========================================================================
-- Attachments
-- =========================================================================
CREATE INDEX idx_attachments_message_id ON attachments(message_id);

-- =========================================================================
-- Location History (critical for replay feature)
-- =========================================================================
CREATE INDEX idx_location_history_user_device ON location_history(user_id, device_id);
CREATE INDEX idx_location_history_recorded_at ON location_history(recorded_at DESC);
CREATE INDEX idx_location_history_location ON location_history USING GIST(location);
CREATE INDEX idx_location_history_user_time ON location_history(user_id, recorded_at DESC);

-- =========================================================================
-- Audit Logs
-- =========================================================================
CREATE INDEX idx_audit_logs_user_id ON audit_logs(user_id);
CREATE INDEX idx_audit_logs_group_id ON audit_logs(group_id);
CREATE INDEX idx_audit_logs_created_at ON audit_logs(created_at DESC);
CREATE INDEX idx_audit_logs_action ON audit_logs(action);
