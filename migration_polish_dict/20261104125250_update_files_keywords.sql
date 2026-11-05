-- +goose Up
SELECT 'up SQL query';
UPDATE files SET keywords = ( to_tsvector('polish', (translate ( (SELECT data->'name')::text, ',._-+', '     ') || ' ' || (SELECT(data->'ext')::text) ) ) ) WHERE keywords IS NULL;
UPDATE files SET ext_id = ext.id FROM ext WHERE files.ext = ext.ext AND files.is_dir = FALSE;

-- +goose Down
SELECT 'down SQL query';
