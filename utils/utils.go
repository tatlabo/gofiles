package utils

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"gofiles/chroma"
	"html/template"
	"os"
	"path/filepath"
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

func PgConn() (*sql.DB, error) {

	const connStr = "user=golang password=golang dbname=json host=localhost sslmode=disable"
	db, err := sql.Open("postgres", connStr)
	if err != nil {
		return nil, fmt.Errorf("Error opening database connection.")
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

// create index on files
const creteIndexOnFiles = `
CREATE INDEX IF NOT EXISTS idx_name ON files(LOWER(name));`

const creteIndexOnExt = `
CREATE INDEX IF NOT EXISTS idx_ext ON files(LOWER(ext));`

const insertIntoExt = `INSERT INTO ext (ext)
SELECT DISTINCT ext FROM files WHERE files.ext IS NOT NULL ON CONFLICT (ext) DO NOTHING;`

const updateExtId = `
UPDATE files SET ext_id = ext.id FROM ext WHERE files.ext = ext.ext AND files.ext_id != NULL;`

const updateKeywords = `
UPDATE files SET keywords = ( to_tsvector('polish', name) || ' ' || ext ) WHERE keywords IS NOT NULL;`

const createGinOnKeywords = `
CREATE INDEX idx_keywords_gin ON keywords_gin USING GIN (to_tsvector('polish', keyword));`

func CreateFiles() error {

	db, err := PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	if _, err := db.Exec(createFiles); err != nil {
		return err
	}

	if _, err := db.Exec(createExt); err != nil {
		return err
	}
	// if _, err := db.Exec(creteIndexOnFiles); err != nil {
	// 	return err
	// }
	if _, err := db.Exec(creteIndexOnExt); err != nil {
		return err
	}

	if _, err := db.Exec(insertIntoExt); err != nil {
		return err
	}

	if _, err := db.Exec(updateExtId); err != nil {
		return err
	}

	if _, err := db.Exec(updateKeywords); err != nil {
		return err
	}

	if _, err := db.Exec(createGinOnKeywords); err != nil {
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
	re := regexp.MustCompile(`[^a-zA-Z0-9.,;\/-_!$ ]+ `)
	// re := regexp.MustCompile(`[^a-zA-Z0-9.,\-_ ]+`)
	return re.ReplaceAllString(input, "")
}

// ValidatePath validates a directory path for security and accessibility
func ValidatePath(path string) (string, error) {
	// Trim whitespace
	path = strings.TrimSpace(path)

	// Check if path is empty
	if path == "" {
		return "", fmt.Errorf("path cannot be empty")
	}

	// Check for path traversal attempts before cleaning
	if strings.Contains(path, "..") {
		return "", fmt.Errorf("path traversal detected: path contains '..'")
	}

	// Clean and normalize the path
	cleanPath := filepath.Clean(path)

	// Convert to absolute path
	absPath, err := filepath.Abs(cleanPath)
	if err != nil {
		return "", fmt.Errorf("invalid path: %w", err)
	}

	// Check if path exists
	info, err := os.Stat(absPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", fmt.Errorf("path does not exist: %s", absPath)
		}
		return "", fmt.Errorf("cannot access path: %w", err)
	}

	// Check if path is a directory
	if !info.IsDir() {
		return "", fmt.Errorf("path is not a directory: %s", absPath)
	}

	// Check if path is readable
	file, err := os.Open(absPath)
	if err != nil {
		return "", fmt.Errorf("path is not accessible: %w", err)
	}
	defer file.Close()

	return absPath, nil
}

func FormatDate(t time.Time) string {
	return t.Format("2006-01-02 15:04:05") // Customize the format as needed
}

func Not(b bool) bool {
	return !b
}

func Equals(a, b any) bool {
	return a == b
}

func Notequals(a, b any) bool {
	return a != b
}

func VanillaSql(s []string, group bool) error {

	var e error

	db, e := PgConn()
	if e != nil {
		return fmt.Errorf("cann't connect to database (utils.VanillaSql) with error: %s", e)
	}
	defer db.Close()

	tx, e := db.Begin()
	if e != nil {
		return fmt.Errorf("failed to begin transaction (utils.VanillaSql): %s", e)
	}

	if group == true {
		groupCommands := strings.Join(s, "\n")
		if _, e = tx.Exec(groupCommands); e != nil {
			return fmt.Errorf("failed to group commands (utils.VanillaSql): %s", e)
		}
		return tx.Commit()
	}

	for _, s := range s {
		if _, e = tx.Exec(s); e != nil {
			return fmt.Errorf("failed to execute (utils.VanillaSql) grouped commands: %s", e)
		}
	}
	return tx.Commit()

}

func VanillaRaw(xs []byte) error {

	var e error

	db, e := PgConn()
	if e != nil {
		return e
	}
	defer db.Close()

	tx, e := db.Begin()
	if e != nil {
		return e
	}
	if _, e = tx.Exec(string(xs)); e != nil {
		return e
	}
	return tx.Commit()

}

func TxtToChoroma(address string) (template.HTML, error) {

	fin, err := os.Open(address)

	if err != nil {
		return "", err
	}
	defer fin.Close()

	highlightCode, err := chroma.HighlightCode(address)

	if err != nil {
		return "", err
	}

	return template.HTML(highlightCode), nil

}
