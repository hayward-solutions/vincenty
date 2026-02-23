-- Create the drawings table for map annotations (lines, circles, rectangles).
-- GeoJSON is stored as a FeatureCollection with per-feature styling in properties.

CREATE TABLE drawings (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    owner_id   UUID        NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name       TEXT        NOT NULL DEFAULT '',
    geojson    JSONB       NOT NULL DEFAULT '{"type":"FeatureCollection","features":[]}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_drawings_owner_id ON drawings(owner_id);
