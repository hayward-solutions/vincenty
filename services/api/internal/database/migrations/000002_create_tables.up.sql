-- =========================================================================
-- Users
-- =========================================================================
CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    username      VARCHAR(255) UNIQUE NOT NULL,
    email         VARCHAR(255) UNIQUE NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    display_name  VARCHAR(255),
    is_admin      BOOLEAN NOT NULL DEFAULT FALSE,
    is_active     BOOLEAN NOT NULL DEFAULT TRUE,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Devices
-- =========================================================================
CREATE TABLE devices (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name          VARCHAR(255) NOT NULL,
    device_type   VARCHAR(50) NOT NULL DEFAULT 'web',
    device_uid    VARCHAR(255) UNIQUE,
    last_seen_at  TIMESTAMPTZ,
    last_location GEOMETRY(POINT, 4326),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Groups
-- =========================================================================
CREATE TABLE groups (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Group Members
-- =========================================================================
CREATE TABLE group_members (
    id             UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    group_id       UUID NOT NULL REFERENCES groups(id) ON DELETE CASCADE,
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    can_read       BOOLEAN NOT NULL DEFAULT TRUE,
    can_write      BOOLEAN NOT NULL DEFAULT FALSE,
    is_group_admin BOOLEAN NOT NULL DEFAULT FALSE,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE(group_id, user_id)
);

-- =========================================================================
-- Messages
-- =========================================================================
CREATE TABLE messages (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    sender_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    sender_device_id UUID REFERENCES devices(id) ON DELETE SET NULL,
    group_id         UUID REFERENCES groups(id) ON DELETE CASCADE,
    recipient_id     UUID REFERENCES users(id) ON DELETE CASCADE,
    content          TEXT,
    message_type     VARCHAR(50) NOT NULL DEFAULT 'text',
    location         GEOMETRY(POINT, 4326),
    metadata         JSONB,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT messages_target_check
        CHECK (group_id IS NOT NULL OR recipient_id IS NOT NULL)
);

-- =========================================================================
-- Attachments (files stored in object storage)
-- =========================================================================
CREATE TABLE attachments (
    id           UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    message_id   UUID NOT NULL REFERENCES messages(id) ON DELETE CASCADE,
    filename     VARCHAR(255) NOT NULL,
    content_type VARCHAR(255) NOT NULL,
    size_bytes   BIGINT NOT NULL,
    object_key   VARCHAR(512) NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Location History
-- =========================================================================
CREATE TABLE location_history (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    device_id   UUID NOT NULL REFERENCES devices(id) ON DELETE CASCADE,
    location    GEOMETRY(POINT, 4326) NOT NULL,
    altitude    DOUBLE PRECISION,
    heading     DOUBLE PRECISION,
    speed       DOUBLE PRECISION,
    accuracy    DOUBLE PRECISION,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Map Configurations
-- =========================================================================
CREATE TABLE map_configs (
    id          UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name        VARCHAR(255) NOT NULL,
    source_type VARCHAR(50) NOT NULL DEFAULT 'remote',
    tile_url    VARCHAR(512),
    style_json  JSONB,
    min_zoom    INTEGER NOT NULL DEFAULT 0,
    max_zoom    INTEGER NOT NULL DEFAULT 18,
    is_default  BOOLEAN NOT NULL DEFAULT FALSE,
    created_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- =========================================================================
-- Audit Logs
-- =========================================================================
CREATE TABLE audit_logs (
    id            UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    user_id       UUID REFERENCES users(id) ON DELETE SET NULL,
    device_id     UUID REFERENCES devices(id) ON DELETE SET NULL,
    action        VARCHAR(100) NOT NULL,
    resource_type VARCHAR(100),
    resource_id   UUID,
    group_id      UUID REFERENCES groups(id) ON DELETE SET NULL,
    metadata      JSONB,
    location      GEOMETRY(POINT, 4326),
    ip_address    INET,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
