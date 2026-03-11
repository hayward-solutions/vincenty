-- =========================================================================
-- Media Rooms — active and historical video/voice/PTT rooms
-- =========================================================================
CREATE TABLE media_rooms (
    id               UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name             TEXT        NOT NULL,
    room_type        VARCHAR(50) NOT NULL CHECK (room_type IN ('call', 'ptt_channel', 'video_feed')),
    group_id         UUID        REFERENCES groups(id) ON DELETE SET NULL,
    created_by       UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    livekit_room     TEXT        NOT NULL UNIQUE,
    is_active        BOOLEAN     NOT NULL DEFAULT true,
    max_participants INT         NOT NULL DEFAULT 50,
    metadata         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at         TIMESTAMPTZ
);

CREATE INDEX idx_media_rooms_group_id ON media_rooms(group_id);
CREATE INDEX idx_media_rooms_active ON media_rooms(is_active) WHERE is_active = true;
CREATE INDEX idx_media_rooms_created_by ON media_rooms(created_by);

-- =========================================================================
-- Media Room Participants — who joined/left each room (audit trail)
-- =========================================================================
CREATE TABLE media_room_participants (
    id        UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id   UUID        NOT NULL REFERENCES media_rooms(id) ON DELETE CASCADE,
    user_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id UUID        REFERENCES devices(id) ON DELETE SET NULL,
    role      VARCHAR(50) NOT NULL DEFAULT 'participant' CHECK (role IN ('host', 'participant', 'viewer')),
    joined_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    left_at   TIMESTAMPTZ
);

CREATE INDEX idx_media_room_participants_room_id ON media_room_participants(room_id);
CREATE INDEX idx_media_room_participants_user_id ON media_room_participants(user_id);

-- =========================================================================
-- Video Feeds — external cameras (CCTV, drones, bodycams, dashcams)
-- =========================================================================
CREATE TABLE video_feeds (
    id                 UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    name               TEXT        NOT NULL,
    feed_type          VARCHAR(50) NOT NULL CHECK (feed_type IN ('rtsp', 'rtmp', 'whip', 'phone_cam')),
    source_url         TEXT,
    group_id           UUID        NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    created_by         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    livekit_ingress_id TEXT,
    livekit_room       TEXT        REFERENCES media_rooms(livekit_room),
    stream_key         TEXT,
    is_active          BOOLEAN     NOT NULL DEFAULT false,
    metadata           JSONB,
    created_at         TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at         TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_video_feeds_group_id ON video_feeds(group_id);
CREATE INDEX idx_video_feeds_created_by ON video_feeds(created_by);
CREATE INDEX idx_video_feeds_active ON video_feeds(is_active) WHERE is_active = true;

-- =========================================================================
-- Recordings — stored media from calls, feeds, and PTT channels
-- =========================================================================
CREATE TABLE recordings (
    id              UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    room_id         UUID        REFERENCES media_rooms(id) ON DELETE SET NULL,
    feed_id         UUID        REFERENCES video_feeds(id) ON DELETE SET NULL,
    egress_id       TEXT        NOT NULL,
    storage_path    TEXT,
    file_type       VARCHAR(20) NOT NULL DEFAULT 'mp4',
    duration_secs   INT,
    file_size_bytes BIGINT,
    status          VARCHAR(50) NOT NULL DEFAULT 'recording' CHECK (status IN ('recording', 'processing', 'complete', 'failed')),
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ
);

CREATE INDEX idx_recordings_room_id ON recordings(room_id);
CREATE INDEX idx_recordings_feed_id ON recordings(feed_id);
CREATE INDEX idx_recordings_status ON recordings(status);

-- =========================================================================
-- PTT Channels — persistent push-to-talk channels for groups
-- =========================================================================
CREATE TABLE ptt_channels (
    id         UUID        PRIMARY KEY DEFAULT gen_random_uuid(),
    group_id   UUID        NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    room_id    UUID        NOT NULL REFERENCES media_rooms(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL,
    is_default BOOLEAN     NOT NULL DEFAULT false,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, name)
);

CREATE INDEX idx_ptt_channels_group_id ON ptt_channels(group_id);
