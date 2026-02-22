-- Re-add terrain columns to map_configs.
ALTER TABLE map_configs ADD COLUMN terrain_url VARCHAR(512);
ALTER TABLE map_configs ADD COLUMN terrain_encoding VARCHAR(20) NOT NULL DEFAULT 'terrarium';

-- Drop terrain_configs table.
DROP TABLE IF EXISTS terrain_configs;
