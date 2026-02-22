CREATE TABLE terrain_configs (
    id               UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
    name             VARCHAR(255) NOT NULL,
    terrain_url      VARCHAR(512) NOT NULL,
    terrain_encoding VARCHAR(20) NOT NULL DEFAULT 'terrarium',
    is_default       BOOLEAN NOT NULL DEFAULT FALSE,
    created_by       UUID REFERENCES users(id) ON DELETE SET NULL,
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Migrate existing terrain data from map_configs (only rows that have a terrain_url).
INSERT INTO terrain_configs (name, terrain_url, terrain_encoding, is_default, created_by, created_at, updated_at)
SELECT name || ' Terrain', terrain_url, terrain_encoding, is_default, created_by, created_at, updated_at
FROM map_configs
WHERE terrain_url IS NOT NULL;

-- Drop terrain columns from map_configs.
ALTER TABLE map_configs DROP COLUMN IF EXISTS terrain_encoding;
ALTER TABLE map_configs DROP COLUMN IF EXISTS terrain_url;
