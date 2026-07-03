CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL DEFAULT ''
);

INSERT INTO settings (key, value) VALUES ('myboot_enabled', 'true')
ON CONFLICT (key) DO NOTHING;
