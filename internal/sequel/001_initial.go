package sequel

var InitialMigration = []string{

	// `CREATE TABLE IF NOT EXISTS ext (id SERIAL PRIMARY KEY, ext TEXT UNIQUE);`,

	`CREATE TABLE IF NOT EXISTS directory 
	(id UUID PRIMARY KEY DEFAULT gen_random_uuid(), json jsonb, UNIQUE (json));`,

	`CREATE TABLE IF NOT EXISTS files (
	id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
	directory_id UUID, FOREIGN KEY(directory_id) REFERENCES directory(id) ON DELETE CASCADE,
	data jsonb,
	keywords TEXT,
	created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
	UNIQUE (directory_id, data));`,

	`CREATE TABLE IF NOT EXISTS search
	(id SERIAL PRIMARY KEY,
	input TEXT,
	created TIMESTAMPTZ NOT NULL DEFAULT NOW());`,

	`CREATE INDEX IF NOT EXISTS idx_keywords_gin ON files USING GIN (to_tsvector('polish', keywords));`,
}

func InitialMigrations() []string {
	return InitialMigration
}

func PolishWords() []string {
	return []string{
		`DO $$ 
		BEGIN 
    		CREATE TEXT SEARCH CONFIGURATION polish (COPY = simple);
		EXCEPTION 
			WHEN duplicate_object THEN NULL;
	END $$;`,
	}
}
