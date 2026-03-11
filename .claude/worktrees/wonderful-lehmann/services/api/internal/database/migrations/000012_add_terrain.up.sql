ALTER TABLE map_configs ADD COLUMN terrain_url VARCHAR(512);
ALTER TABLE map_configs ADD COLUMN terrain_encoding VARCHAR(20) NOT NULL DEFAULT 'terrarium';
