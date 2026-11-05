-- +goose Up
CREATE TEXT SEARCH DICTIONARY polish_ispell (
	TEMPLATE = ispell,
	DictFile = polish,
	AffFile = polish,
	StopWords = polish
);
CREATE TEXT SEARCH CONFIGURATION polish (COPY = pg_catalog.simple);
ALTER TEXT SEARCH CONFIGURATION polish
	ALTER MAPPING FOR asciiword, asciihword, hword_asciipart, word, hword, hword_part
	WITH polish_ispell, pg_catalog.simple;

-- +goose Down
SELECT 'down SQL query';
