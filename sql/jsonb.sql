SELECT DISTINCT(pg_typeof ( body )) FROM track;
SELECT body->> 'name' as name FROM track LIMIT 10;
SELECT MAX((body->>'count') :: INT) FROM track;
SELECT body ->> 'name' As NAME, body ->> 'artist' AS artist, (body->> 'count')::int AS count FROM track ORDER BY count DESC LIMIT 10;
UPDATE track SET body = body || '{"favorite": "yes"}' WHERE (body->'count')::INT > 200;
SELECT body FROM track WHERE (body->'count')::INT > 150 LIMIT 10;
SELECT COUNT (*) FROM track WHERE body @> '{"favorite": "yes"}';
CREATE INDEX track_btree ON track USING BTREE((body->>'name'));
CREATE INDEX track_gin ON track USING gin(body);
CREATE INDEX track_gin_path_ops ON track USING gin(body jsonb_path_ops);

EXPLAIN ANALYZE SELECT body FROM track WHERE body->>'name' = 'Summer Nights';