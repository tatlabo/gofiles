-- INSERT INTO ext (ext)
-- SELECT DISTINCT ext FROM files WHERE files.ext IS NOT NULL ON CONFLICT (ext) DO NOTHING;
UPDATE files SET keywords = ( to_tsvector('polish', (translate ( (SELECT data->'name')::text, ',._-+', '     ') || ' ' || (SELECT(data->'ext')::text) ) ) );