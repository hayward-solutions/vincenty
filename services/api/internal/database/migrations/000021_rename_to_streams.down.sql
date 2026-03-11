-- =========================================================================
-- Migration 21 DOWN: Reverse streams → video_feeds rename
-- =========================================================================

-- 1. Reverse recordings.stream_id → feed_id
ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_stream_id_fkey;
ALTER TABLE recordings ADD CONSTRAINT recordings_feed_id_fkey
    FOREIGN KEY (stream_id) REFERENCES streams(id) ON DELETE SET NULL;
ALTER INDEX idx_recordings_stream_id RENAME TO idx_recordings_feed_id;
ALTER TABLE recordings RENAME COLUMN stream_id TO feed_id;

-- 2. Reverse media_rooms room_type constraint
UPDATE media_rooms SET room_type = 'video_feed' WHERE room_type = 'stream';
ALTER TABLE media_rooms DROP CONSTRAINT media_rooms_room_type_check;
ALTER TABLE media_rooms ADD CONSTRAINT media_rooms_room_type_check
    CHECK (room_type IN ('call', 'ptt_channel', 'video_feed'));

-- 3. Reverse index renames
ALTER INDEX idx_streams_group_id   RENAME TO idx_video_feeds_group_id;
ALTER INDEX idx_streams_created_by RENAME TO idx_video_feeds_created_by;
ALTER INDEX idx_streams_active     RENAME TO idx_video_feeds_active;

-- 4. Reverse source_type data migration
UPDATE streams SET source_type = 'phone_cam' WHERE source_type = 'device_camera';

-- 5. Reverse source_type → feed_type constraint
ALTER TABLE streams DROP CONSTRAINT streams_source_type_check;
ALTER TABLE streams ADD CONSTRAINT video_feeds_feed_type_check
    CHECK (source_type IN ('rtsp', 'rtmp', 'whip', 'phone_cam'));
ALTER TABLE streams RENAME COLUMN source_type TO feed_type;

-- 6. Reverse table rename
ALTER TABLE streams RENAME TO video_feeds;
