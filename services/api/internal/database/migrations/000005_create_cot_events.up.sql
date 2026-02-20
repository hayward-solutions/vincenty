-- =========================================================================
-- CoT Events (Cursor on Target)
-- =========================================================================
CREATE TABLE cot_events (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    event_uid   TEXT NOT NULL,
    event_type  TEXT NOT NULL,
    how         TEXT NOT NULL DEFAULT '',
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    device_id   UUID REFERENCES devices(id) ON DELETE SET NULL,
    callsign    TEXT,
    location    GEOMETRY(Point, 4326) NOT NULL,
    hae         DOUBLE PRECISION,
    ce          DOUBLE PRECISION,
    le          DOUBLE PRECISION,
    speed       DOUBLE PRECISION,
    course      DOUBLE PRECISION,
    detail_xml  TEXT,
    raw_xml     TEXT,
    event_time  TIMESTAMPTZ NOT NULL,
    start_time  TIMESTAMPTZ NOT NULL,
    stale_time  TIMESTAMPTZ NOT NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- CoT Events Indexes
-- =========================================================================

-- Query by CoT UID (e.g., get latest event for a TAK device)
CREATE INDEX idx_cot_events_event_uid ON cot_events(event_uid);

-- Query by event type prefix (e.g., all atom events, all geochat events)
CREATE INDEX idx_cot_events_event_type ON cot_events(event_type);

-- Time-range queries (event_time is the CoT event's own timestamp)
CREATE INDEX idx_cot_events_event_time ON cot_events(event_time DESC);

-- Stale time for cleanup / expiry queries
CREATE INDEX idx_cot_events_stale_time ON cot_events(stale_time);

-- Spatial queries (find events in a bounding box / radius)
CREATE INDEX idx_cot_events_location ON cot_events USING GIST(location);

-- Composite: latest event per UID (common query pattern)
CREATE INDEX idx_cot_events_uid_time ON cot_events(event_uid, event_time DESC);

-- FK lookups
CREATE INDEX idx_cot_events_user_id ON cot_events(user_id);
CREATE INDEX idx_cot_events_device_id ON cot_events(device_id);
