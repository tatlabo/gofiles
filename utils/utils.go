package utils

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"regexp"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

// GenerateID creates a unique identifier.
func GenerateID() string {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b)
}

// FormatDate formats a time.Time object into a string.
func FormatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}

func PgConn() (*sql.DB, error) {

	const connStr = "user=golang password=golang dbname=go host=localhost sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, err
	}
	return db, nil
}

const createExt = `
CREATE TABLE IF NOT EXISTS ext (id SERIAL, ext TEXT UNIQUE, PRIMARY KEY (id));
`

const createFiles = `
CREATE TABLE IF NOT EXISTS files 
(id SERIAL PRIMARY KEY,
parent_id INTEGER,
path TEXT NOT NULL,
name TEXT NOT NULL,
ext TEXT,
ext_id INTEGER, FOREIGN KEY(ext_id) REFERENCES ext(id) ON DELETE CASCADE,
is_dir BOOLEAN NOT NULL DEFAULT FALSE,
size BIGINT,
keywords TEXT,
mod_time TIMESTAMPTZ NOT NULL DEFAULT NOW(),
UNIQUE (path, name, ext, is_dir));`

const creteIndexOnFiles = `
CREATE INDEX IF NOT EXISTS idx_name ON files(LOWER(name));`

const creteIndexOnExt = `
CREATE INDEX IF NOT EXISTS idx_ext ON ext(LOWER(ext));`

func CreateFiles() error {

	db, err := PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	if _, err := db.Exec(createExt); err != nil {
		return err
	}
	if _, err := db.Exec(createFiles); err != nil {
		return err
	}
	if _, err := db.Exec(creteIndexOnFiles); err != nil {
		return err
	}
	if _, err := db.Exec(creteIndexOnExt); err != nil {
		return err
	}
	return nil
}

func DropFiles() error {
	db, err := PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	_, err = db.Exec(`DROP TABLE IF EXISTS files;`)
	if err != nil {
		return (err)
	}

	return nil
}

func ConvertBytes(bytes int64) string {
	const unit = 1024
	if bytes < unit {
		return fmt.Sprintf("%d B", bytes)
	}
	div, exp := int64(unit), 0
	for n := bytes / unit; n >= unit; n /= unit {
		div *= unit
		exp++
	}
	return fmt.Sprintf("%.2f %cB", float64(bytes)/float64(div), "KMGTPE"[exp])
}

func CleanInput(input string) string {
	input = strings.TrimSpace(input)
	input = strings.ToLower(input)
	re := regexp.MustCompile(`[^a-zA-Z0-9.,\-_ ]+`)
	return re.ReplaceAllString(input, "")
}
