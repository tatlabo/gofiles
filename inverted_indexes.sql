-- Select generate_series(100, 120) as numbers, random(), random(), trunc(random()*100)::integer;
-- Select repeat ('Some string ', 6);
-- Create Table textfun (content TEXT);
-- INSERT INTO textfun (content) Select trunc(random()*100)::integer::text || generate_series(1,10);
-- CREATE TABLE novel (id SERIAL)
-- Select unnest(string_to_array(  (SELECT line FROM novel WHERE LENGTH(line) > 20 LIMIT 1), ' ') );
-- CREATE TABLE novel_gin (keyword Text, doc_id integer REFERENCES novel(id) ON DELETE CASCADE);
-- CREATE TABLE keywords_gin (keyword TEXT, files_id INTEGER REFERENCES files(id) ON DELETE CASCADE);

-- SELECT id, s.keywords AS keywrord FROM files AS D, s(keyword) LIMIT 20;

-- INSERT INTO keywords_gin (files_id, keyword) SELECT f.id, keyword FROM files AS f, unnest ( string_to_array(f.keywords, ' ') ) AS keyword ORDER BY f.id;

SELECT DISTINCT id, (path || '\'|| name) FROM files f JOIN keywords_gin kg ON f.id = kg.files_id WHERE kg.keyword IN ('main', 'style');