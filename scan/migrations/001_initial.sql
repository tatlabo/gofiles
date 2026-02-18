
CREATE TABLE IF NOT EXISTS directory (
id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
path TEXT UNIQUE,
is_done BOOLEAN NOT NULL DEFAULT FALSE,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
UNIQUE (path));

CREATE TABLE IF NOT EXISTS files (
id SERIAL PRIMARY KEY,
directory_id UUID, FOREIGN KEY(directory_id) REFERENCES directory(id) ON DELETE CASCADE,
data jsonb,
keywords TEXT,
created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
UNIQUE (directory_id, data));

CREATE TABLE IF NOT EXISTS search
(id SERIAL PRIMARY KEY,
input TEXT,
created TIMESTAMPTZ NOT NULL DEFAULT NOW());

DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_ts_config WHERE cfgname = 'polish'
    ) THEN
        CREATE TEXT SEARCH CONFIGURATION polish (COPY = simple);
    END IF;
END $$;

CREATE INDEX IF NOT EXISTS idx_keywords_gin ON files USING GIN (to_tsvector('polish', keywords));