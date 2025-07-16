-- SQL INSERT MULTIPLE VALUES Examples

-- Method 1: INSERT with multiple VALUES rows (Standard SQL)
INSERT INTO indexed (path, done, created) VALUES 
    ('Z:/0000_ICE/250409_Ice_Wielkanoc_2025', false, NOW()),
    ('/home/user/downloads', false, NOW()),
    ('/home/user/pictures', false, NOW()),
    ('/var/log', false, NOW());

-- Method 2: Multiple separate INSERT statements
INSERT INTO indexed (path, done, created) VALUES ('/path1', 0, NOW());
INSERT INTO indexed (path, done, created) VALUES ('/path2', 1, NOW());
INSERT INTO indexed (path, done, created) VALUES ('/path3', 0, NOW());

-- Method 3: INSERT with SELECT and UNION (when you need computed values)
INSERT INTO indexed (path, done, created)
SELECT * FROM (
    SELECT '/computed/path1' as path, 0 as done, NOW() as created
    UNION ALL
    SELECT '/computed/path2', 1, NOW()
    UNION ALL
    SELECT '/computed/path3', 0, NOW()
) AS temp_data;

-- Method 4: INSERT with VALUES and variables (PostgreSQL specific)
INSERT INTO indexed (path, done, created) VALUES 
    ($1, $2, $3),
    ($4, $5, $6),
    ($7, $8, $9);

-- Method 5: Using UNNEST for array inputs (PostgreSQL specific)
INSERT INTO indexed (path, done, created)
SELECT * FROM UNNEST(
    ARRAY['/path1', '/path2', '/path3'],
    ARRAY[0, 1, 0],
    ARRAY[NOW(), NOW(), NOW()]
);

-- Method 6: INSERT with ON CONFLICT (PostgreSQL UPSERT)
INSERT INTO indexed (path, done, created) VALUES 
    ('/existing/path', 1, NOW()),
    ('/new/path', 0, NOW())
ON CONFLICT (path) 
DO UPDATE SET 
    done = EXCLUDED.done,
    created = EXCLUDED.created;

-- Method 7: INSERT with default values and specific columns
INSERT INTO indexed (path, done) VALUES 
    ('/path1', 0),
    ('/path2', 1),
    ('/path3', 0);
-- Note: 'created' will use default value (likely NOW() or CURRENT_TIMESTAMP)

-- Method 8: INSERT from another table
INSERT INTO indexed (path, done, created)
SELECT file_path, 0, CURRENT_TIMESTAMP 
FROM temp_files 
WHERE processed = false;

-- Method 9: Conditional INSERT with CASE
INSERT INTO indexed (path, done, created) VALUES 
    ('/path1', CASE WHEN '/path1' LIKE '%important%' THEN 1 ELSE 0 END, NOW()),
    ('/path2', CASE WHEN '/path2' LIKE '%important%' THEN 1 ELSE 0 END, NOW());

-- Method 10: INSERT with JSON data (PostgreSQL)
INSERT INTO indexed (path, done, created)
SELECT 
    json_data->>'path',
    (json_data->>'done')::integer,
    NOW()
FROM (
    VALUES 
        ('{"path": "/json/path1", "done": 0}'),
        ('{"path": "/json/path2", "done": 1}')
) AS t(json_data_text),
LATERAL (SELECT json_data_text::json AS json_data) AS parsed;
