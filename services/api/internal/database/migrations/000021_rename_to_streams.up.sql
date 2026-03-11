-- =========================================================================
-- Migration 21: Rename video_feeds → streams, remove "call" room type
-- =========================================================================

-- 1. Rename video_feeds table → streams
ALTER TABLE video_feeds RENAME TO streams;

-- 2. Rename column: feed_type → source_type
ALTER TABLE streams RENAME COLUMN feed_type TO source_type;

-- 3. Update the source_type CHECK constraint (was on video_feeds)
ALTER TABLE streams DROP CONSTRAINT video_feeds_feed_type_check;
ALTER TABLE streams ADD CONSTRAINT streams_source_type_check
    CHECK (source_type IN ('rtsp', 'rtmp', 'whip', 'device_camera', 'screen_share'));

-- 4. Migrate existing data: phone_cam → device_camera
UPDATE streams SET source_type = 'device_camera' WHERE source_type = 'phone_cam';

-- 5. Rename indexes
ALTER INDEX idx_video_feeds_group_id   RENAME TO idx_streams_group_id;
ALTER INDEX idx_video_feeds_created_by RENAME TO idx_streams_created_by;
ALTER INDEX idx_video_feeds_active     RENAME TO idx_streams_active;

-- 6. Update media_rooms room_type CHECK constraint: remove 'call', rename 'video_feed' → 'stream'
ALTER TABLE media_rooms DROP CONSTRAINT media_rooms_room_type_check;
ALTER TABLE media_rooms ADD CONSTRAINT media_rooms_room_type_check
    CHECK (room_type IN ('stream', 'ptt_channel'));

-- 7. Migrate existing room_type data
UPDATE media_rooms SET room_type = 'stream' WHERE room_type = 'video_feed';
-- End any active call rooms (calls are no longer a concept)
UPDATE media_rooms
    SET room_type = 'stream', is_active = false, ended_at = NOW()
    WHERE room_type = 'call' AND is_active = true;
UPDATE media_rooms SET room_type = 'stream' WHERE room_type = 'call';

-- 8. Rename recordings.feed_id → stream_id (column + index)
ALTER TABLE recordings RENAME COLUMN feed_id TO stream_id;
ALTER INDEX idx_recordings_feed_id RENAME TO idx_recordings_stream_id;

-- 9. Update FK on recordings.stream_id to point to streams(id)
ALTER TABLE recordings DROP CONSTRAINT IF EXISTS recordings_feed_id_fkey;
ALTER TABLE recordings ADD CONSTRAINT recordings_stream_id_fkey
    FOREIGN KEY (stream_id) REFERENCES streams(id) ON DELETE SET NULL;
