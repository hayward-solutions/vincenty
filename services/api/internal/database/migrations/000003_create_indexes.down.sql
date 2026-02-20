-- Audit Logs
DROP INDEX IF EXISTS idx_audit_logs_action;
DROP INDEX IF EXISTS idx_audit_logs_created_at;
DROP INDEX IF EXISTS idx_audit_logs_group_id;
DROP INDEX IF EXISTS idx_audit_logs_user_id;

-- Location History
DROP INDEX IF EXISTS idx_location_history_user_time;
DROP INDEX IF EXISTS idx_location_history_location;
DROP INDEX IF EXISTS idx_location_history_recorded_at;
DROP INDEX IF EXISTS idx_location_history_user_device;

-- Attachments
DROP INDEX IF EXISTS idx_attachments_message_id;

-- Messages
DROP INDEX IF EXISTS idx_messages_created_at;
DROP INDEX IF EXISTS idx_messages_sender_id;
DROP INDEX IF EXISTS idx_messages_recipient_id;
DROP INDEX IF EXISTS idx_messages_group_id;

-- Group Members
DROP INDEX IF EXISTS idx_group_members_user_id;
DROP INDEX IF EXISTS idx_group_members_group_id;

-- Devices
DROP INDEX IF EXISTS idx_devices_user_id;
