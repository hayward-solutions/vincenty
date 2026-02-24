-- Stream keys for hardware device authentication (drones, CCTV, body cams).
-- Each key maps to a set of default groups for automatic sharing.

CREATE TABLE stream_keys (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    label       TEXT        NOT NULL,
    key_hash    TEXT        NOT NULL,
    created_by  UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    is_active   BOOLEAN     NOT NULL DEFAULT true,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_keys_key_hash ON stream_keys(key_hash);
CREATE INDEX idx_stream_keys_created_by ON stream_keys(created_by);

-- Default groups a stream key auto-shares to when a device starts streaming.
CREATE TABLE stream_key_groups (
    stream_key_id UUID NOT NULL REFERENCES stream_keys(id) ON DELETE CASCADE,
    group_id      UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    PRIMARY KEY (stream_key_id, group_id)
);

-- A live or recorded video stream.
CREATE TABLE streams (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    title           TEXT        NOT NULL DEFAULT '',
    broadcaster_id  UUID        REFERENCES users(id) ON DELETE SET NULL,
    stream_key_id   UUID        REFERENCES stream_keys(id) ON DELETE SET NULL,
    source_type     TEXT        NOT NULL CHECK (source_type IN ('browser', 'rtsp', 'rtmp')),
    status          TEXT        NOT NULL DEFAULT 'live' CHECK (status IN ('live', 'ended')),
    media_path      TEXT        NOT NULL,
    recording_url   TEXT,
    started_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    ended_at        TIMESTAMPTZ,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_streams_broadcaster_id ON streams(broadcaster_id);
CREATE INDEX idx_streams_stream_key_id ON streams(stream_key_id);
CREATE INDEX idx_streams_status ON streams(status);

-- Many-to-many: which groups can view this stream.
CREATE TABLE stream_groups (
    stream_id UUID NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    group_id  UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    shared_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (stream_id, group_id)
);

-- GPS telemetry timestamped to the stream (for map-synced playback).
CREATE TABLE stream_locations (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    stream_id   UUID           NOT NULL REFERENCES streams(id) ON DELETE CASCADE,
    location    GEOMETRY(Point, 4326) NOT NULL,
    altitude    DOUBLE PRECISION,
    heading     DOUBLE PRECISION,
    speed       DOUBLE PRECISION,
    recorded_at TIMESTAMPTZ    NOT NULL,
    created_at  TIMESTAMPTZ    NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_stream_locations_stream_time
    ON stream_locations(stream_id, recorded_at);
