SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE LOWER(name) LIKE LOWER($1)
ORDER BY mod_time DESC
LIMIT $2 OFFSET $3; [marker_to_ae% 10 0]
-- Execution Time: 1078.389 ms


-- Execution Time: 0.577 ms
SELECT id, name, ext, is_dir, path, size, mod_time FROM files WHERE LOWER(name) =$1
ORDER BY mod_time DESC 
LIMIT $2 OFFSET $3; [marker_to_ae 10 0]