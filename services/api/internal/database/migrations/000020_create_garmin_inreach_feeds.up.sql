-- Garmin InReach feed configuration.
-- Each row maps a Garmin MapShare identifier to a Vincenty user and device,
-- enabling the background poller to fetch KML tracking data and bridge it
-- into the internal location system.
CREATE TABLE IF NOT EXISTS garmin_inreach_feeds (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id         UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id       UUID        NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    mapshare_id     TEXT        NOT NULL,
    feed_password   TEXT,                          -- optional MapShare password
    poll_interval   INTERVAL    NOT NULL DEFAULT '120 seconds',
    enabled         BOOLEAN     NOT NULL DEFAULT true,
    last_polled_at  TIMESTAMPTZ,
    last_point_at   TIMESTAMPTZ,                   -- timestamp of the newest point seen
    error_count     INT         NOT NULL DEFAULT 0,
    last_error      TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_garmin_inreach_feeds_mapshare UNIQUE (mapshare_id)
);

CREATE INDEX idx_garmin_inreach_feeds_user    ON garmin_inreach_feeds (user_id);
CREATE INDEX idx_garmin_inreach_feeds_enabled ON garmin_inreach_feeds (enabled) WHERE enabled = true;
