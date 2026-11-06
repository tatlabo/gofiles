// Package utils
package utils

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"html/template"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"gofiles/chroma"

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

var (
	db   *sql.DB
	once sync.Once
)

func PgConn() (*sql.DB, error) {
	var err error
	once.Do(func() {
		connStr := "user=golang password=golang dbname=json host=localhost sslmode=disable"
		db, err = sql.Open("postgres", connStr)
		if err != nil {
			return
		}
		// Configure pool
		db.SetMaxOpenConns(25)
		db.SetMaxIdleConns(5)
		db.SetConnMaxLifetime(5 * time.Minute)

		// Verify connection
		err = db.Ping()
	})
	return db, err
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

func VanillaSQL(s []string, group bool) error {
	var e error

	db, e := PgConn()
	if e != nil {
		return fmt.Errorf("cann't connect to database (utils.VanillaSql) with error: %s", e)
	}

	tx, e := db.Begin()
	if e != nil {
		return fmt.Errorf("failed to begin transaction (utils.VanillaSql): %s", e)
	}

	if group {
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
