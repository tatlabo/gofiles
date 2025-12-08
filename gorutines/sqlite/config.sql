PRAGMA journal_mode = WAL; -- write ahead logging
PRAGMA busy_timeout = 5000; -- 5 seconds
PRAGMA foreign_keys = ON; -- enable foreign key constraints

--sqlite
-- Only 5 types of data are supported: 
-- NULL
-- INTEGER
-- REAL
-- TEXT
-- BLOBs

-- coulms does not have strict types, but type affinities
-- only in strict mode type checking can be enforced

-- time functions
-- ISO 8601 strings: "YYYY-MM-DD HH:MM:SS.SSS"
-- unix time (seconds since 1970-01-01 00:00:00 UTC)
-- julian day numbers - weird