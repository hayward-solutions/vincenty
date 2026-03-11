ALTER TABLE terrain_configs
    DROP COLUMN IF EXISTS is_enabled,
    DROP COLUMN IF EXISTS is_builtin;

ALTER TABLE map_configs
    DROP COLUMN IF EXISTS is_enabled,
    DROP COLUMN IF EXISTS is_builtin;
