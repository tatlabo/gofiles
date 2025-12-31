package main

import "database/sql"

func CreateSourceTable(db *sql.DB) error {

	sql := `
	CREATE TABLE IF NOT EXISTS 
		source (
			id INTEGER PRIMARY KEY,
			url TEXT,
			body TEXT not null,
			raw BLOB,
			created_at TEXT not null,
			keywords TEXT,
			md5 TEXT,
			done INTEGER NOT NULL DEFAULT 0,
			display INTEGER NOT NULL DEFAULT 0,
			UNIQUE (url, created_at)
		)
		STRICT;`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil

}

func CreateArticleTable(db *sql.DB) error {

	sql := `
	CREATE TABLE IF NOT EXISTS 
		daily (
			id INTEGER PRIMARY KEY,
			source_id INTEGER REFERENCES source(id) ON DELETE CASCADE,
			title TEXT,
			url TEXT,
			body TEXT not null,
			raw BLOB,
			created_at TEXT not null,
			keywords TEXT,
			md5 TEXT,
			done INTEGER NOT NULL DEFAULT 0,
			display INTEGER NOT NULL DEFAULT 0,
			UNIQUE (url, source_id)
		)
		STRICT;
	`

	_, err := db.Exec(sql)
	if err != nil {
		return err
	}

	return nil

}
