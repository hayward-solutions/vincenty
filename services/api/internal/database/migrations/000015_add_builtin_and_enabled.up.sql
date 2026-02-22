-- Add is_builtin and is_enabled columns to map_configs and terrain_configs.
-- Built-in configs are seeded at startup and cannot be deleted or have their
-- core fields modified. They can be enabled/disabled and toggled as default.

ALTER TABLE map_configs
    ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_enabled BOOLEAN NOT NULL DEFAULT true;

ALTER TABLE terrain_configs
    ADD COLUMN is_builtin BOOLEAN NOT NULL DEFAULT false,
    ADD COLUMN is_enabled BOOLEAN NOT NULL DEFAULT true;
