CREATE TEXT SEARCH DICTIONARY polish_dict (
    TEMPLATE = ispell,
    DictFile = polish,
    AffFile = polish,
    StopWords = polish
);




Select generate_series(100, 120) as numbers, random(), random(), trunc(random()*100)::integer;
Select repeat ('Some string ', 6);
Create Table textfun (content TEXT);
INSERT INTO textfun (content) Select trunc(random()*100)::integer::text || generate_series(1,10);
CREATE TABLE novel (id SERIAL)
Select unnest(string_to_array(  (SELECT line FROM novel WHERE LENGTH(line) > 20 LIMIT 1), ' ') );
CREATE TABLE novel_gin (keyword Text, doc_id integer REFERENCES novel(id) ON DELETE CASCADE);
CREATE TABLE keywords_gin (keyword TEXT, files_id INTEGER REFERENCES files(id) ON DELETE CASCADE);

SELECT id, s.keywords AS keywrord FROM files AS D, s(keyword) LIMIT 20;

INSERT INTO keywords_gin (files_id, keyword) SELECT f.id, keyword FROM files AS f, unnest ( string_to_array(f.keywords, ' ') ) AS keyword ORDER BY f.id;
SELECT DISTINCT f.id, f.name, f.path FROM files f JOIN keywords_gin kg ON f.id = kg.files_id JOIN ext ON ext.id = f.ext_id 
WHERE kg.keyword IN ('style', 'test') AND ext.ext = 'css';
SELECT to_tsvector('polish', 'zażółć gęślą jaźń');
SELECT to_tsquery('polish', 'gęślą');

DELETE FROM keywords_gin WHERE keyword = '';
SELECT DISTINCT (keyword), count(keyword) c FROM keywords_gin GROUP BY keyword ORDER BY c DESC LIMIT 120;
SELECT * FROM polish_stopwords LIMIT 33
DELETE FROM polish_stopwords;
ALTER SEQUENCE polish_stopwords RESTART WITH 1;

CREATE TABLE polish_stopwords(word text);
\copy polish_stopwords FROM 'C:/Program Files/PostgreSQL/15/share/tsearch_data/polish.stop' WITH (FORMAT text, ENCODING 'UTF8');
SELECT * FROM polish_stopwords;



INSERT INTO keywords_gin (files_id, keyword) SELECT DISTINCT f.id, keyword FROM files f, unnest( string_to_array( LOWER(f.keywords) , ' ')) keyword WHERE keywords NOT IN 
(SELECT word FROM polish_stopwords) ORDER BY id;



-- 1. Download Polish Dictionary Files
-- Download pl_PL.aff and pl_PL.dic from the LibreOffice dictionary repository or OpenOffice extensions.
-- Rename them to polish.aff and polish.dic.
-- 2. Find PostgreSQL’s tsearch_data Directory
-- Locate your PostgreSQL installation directory, e.g.:
-- C:\Program Files\PostgreSQL\16\share\tsearch_data
-- 3. Copy Dictionary Files
-- Copy polish.aff and polish.dic into the tsearch_data directory.
-- 4. (Optional) Create a Stopwords File
-- Create a text file named polish.stop in the same directory with common Polish stopwords (one per line).
-- 5. Create the Dictionary in PostgreSQL
-- Open psql or PgAdmin and run:

CREATE TEXT SEARCH DICTIONARY polish_dict (
    TEMPLATE = ispell,
    DictFile = polish,
    AffFile = polish,
    StopWords = polish
);

CREATE TEXT SEARCH CONFIGURATION polish ( COPY = simple );
ALTER TEXT SEARCH CONFIGURATION polish
ALTER MAPPING FOR word WITH polish_dict, simple;

CREATE TABLE polish_stopwords(word text);
\copy polish_stopwords FROM 'C:/Program Files/PostgreSQL/17/share/tsearch_data/polish.stop' WITH (FORMAT text);
SELECT * FROM polish_stopwords;

-- stemming
-- https://youtu.be/wHL1VVejFEk?t=365