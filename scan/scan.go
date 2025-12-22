package scan

import (
	"encoding/json"
	"fmt"
	"gofiles/internal/models"
	"gofiles/utils"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"github.com/google/uuid"

	"embed"

	_ "github.com/lib/pq"
)

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", "bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx"}

var log = []string{}

func CommitSql(sql []string) error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	for i := range len(sql) {
		if _, err := tx.Exec(sql[i]); err != nil {
			return err
		}
	}

	return tx.Commit()
}

func SqlMigrations(path string) error {
	f, err := migrations.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", path, err)
	}

	if err := VanillaRaw(f); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", path, err)
	}

	return nil
}

func VanillaRaw(xs []byte) error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(string(xs)); err != nil {
		return err
	}
	return tx.Commit()
}

func VanillaSQL(s string) error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}
	if _, err := tx.Exec(s); err != nil {
		return err
	}
	return tx.Commit()
}

func visit(path string, d fs.DirEntry, err error) error {

	if err != nil {
		log = append(log, fmt.Sprintf("Error accessing path %s: %v", path, err))
		return nil // Handle errors accessing a path
	}

	// Check if the directory is in the skip list
	for _, skipDir := range skipDirectories {
		if strings.Contains(path, string(os.PathSeparator)+skipDir+string(os.PathSeparator)) || strings.HasSuffix(path, string(os.PathSeparator)+skipDir) {
			return fs.SkipDir // Skip this directory
		}
	}

	extension := filepath.Ext(path)

	// Skip macOS
	if strings.HasPrefix(d.Name(), "._") {
		return nil
	}

	if slices.Contains(skipFiles, d.Name()) {
		return nil
	}

	if extension == "" && !d.IsDir() {
		log = append(log, fmt.Sprintf("File has no extension: %s\n", path))
		return nil
	}

	s := models.FinfoJSON{}
	s.Name = strings.TrimSuffix(d.Name(), extension)
	s.Name = strings.ReplaceAll(s.Name, "'", "''")

	s.IsDir = d.IsDir()
	s.Directory = filepath.Dir(path) // Handle errors accessing a path}
	if len(extension) > 0 {
		extension = extension[1:]
	}

	s.Ext = extension
	info, err := d.Info()
	if err != nil {
		log = append(log, fmt.Sprintf("Error getting file info for %s: %v", path, err))
		return nil
	}

	s.Size = info.Size()
	s.ModTime = info.ModTime()

	fileList = append(fileList, s)
	return nil
}

//go:embed migrations/*.sql
var migrations embed.FS

var fileList = []models.FinfoJSON{}
var directory string
var directoryId uuid.UUID

func Scan(d models.Directory) error {

	// switch len(os.Args) {
	// case 1:
	// 	fmt.Println("Please provide a path")
	// 	os.Exit(1)
	// case 2:
	// 	path = os.Args[1]
	// 	path = strings.TrimSpace(path)
	// 	path = strings.ReplaceAll(path, "/", "\\")
	// 	if _, err := os.ReadDir(path); err != nil {
	// 		fmt.Fprintf(os.Stderr, "Error reading directory: %v\n", err)
	// 		os.Exit(1)
	// 	}

	// }

	directory = d.Path
	directoryId = d.Id
	// err := insertIntoDirs(path)
	// if err != nil {
	// 	fmt.Fprintf(os.Stderr, "Error inserting into directory table: %v\n", err)
	// 	return err
	// }

	if err := JsonFilesToDb(directory); err != nil {
		fmt.Fprintf(os.Stderr, "invalid path: %v\n", err)
		return err
	}

	if err := updateDirs(directory); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating directory table: %v\n", err)
		return err
	}

	SqlMigrations("migrations/002_initial.sql")

	return nil

}

func JsonFilesToDb(path string) error {
	db, err := utils.PgConn()
	if err != nil {
		return fmt.Errorf("failed to connect to database: %w", err)
	}
	defer db.Close()

	err = filepath.WalkDir(path, visit)
	if err != nil {
		writeLog(&log)
		return fmt.Errorf("Error walking through directories: %w", err)
	}

	s := len(fileList) // Use fileList length, not stmt bytes

	if s == 0 {
		fmt.Println("No items to insert")
		os.Exit(1)
	}

	batchSize := 1000
	totalInserted := 0

	// Process files in batches
	for i := 0; i < len(fileList); i += batchSize {
		end := min(i+batchSize, len(fileList))
		batch := fileList[i:end]

		// Begin transaction for this batch
		tx, err := db.Begin()
		if err != nil {
			return fmt.Errorf("error beginning transaction: %w", err)
		}

		// Prepare statement for batch
		stmt, err := tx.Prepare(`INSERT INTO files (directory_id, data) VALUES ($1, $2::jsonb)`)
		if err != nil {
			tx.Rollback()
			return fmt.Errorf("error preparing statement: %w", err)
		}

		// Insert batch
		for _, file := range batch {
			jsonData, err := json.Marshal(file)
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("error marshalling file: %w", err)
			}

			_, err = stmt.Exec(directoryId, string(jsonData))
			if err != nil {
				stmt.Close()
				tx.Rollback()
				return fmt.Errorf("error inserting file: %w", err)
			}
		}

		stmt.Close()

		// Commit transaction
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("error committing transaction: %w", err)
		}

		totalInserted += len(batch)
		fmt.Printf("Progress: %d/%d files inserted\n", totalInserted, s)
	}

	fmt.Printf("Successfully inserted %d files\n", s)
	return nil
}

func insertIntoDirs(path string) error {

	const stmt = `INSERT INTO directory(path)
	VALUES ($1) RETURNING id;`

	err := VanillaRawReturn(stmt, directory)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inserting directory: %v\n", err)
		os.Exit(10)
	}

	return nil
}

func updateDirs(path string) error {
	const stmt = `UPDATE directory SET is_done = $1, updated_at = NOW() WHERE id=$2;`

	db, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec(stmt, true, directoryId)
	if err != nil {
		return fmt.Errorf("error updating directory: %w", err)
	}

	return nil
}

func VanillaRawReturn(q string, param string) error {
	db, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	err = tx.QueryRow(q, param).Scan(&directoryId)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to insert and scan: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

func writeLog(log *[]string) error {

	f, err := os.Create("log.txt")
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := fmt.Fprintf(f, "%v\n", log); err != nil {
		return err
	}

	return nil
}
