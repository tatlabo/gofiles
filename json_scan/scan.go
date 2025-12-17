package main

import (
	"encoding/json"
	"fmt"
	"gofiles/internal/models"
	"gofiles/utils"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/google/uuid"

	"embed"

	_ "github.com/lib/pq"
)

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}

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
	} else {
		// Check if the directory is in the skip list
		for _, skipDir := range skipDirectories {
			if strings.Contains(path, skipDir) {
				return nil // Skip this directory
			}
		}

		extension := filepath.Ext(path)

		for _, skipFile := range skipFiles {
			if strings.Contains(extension, skipFile) {
				return nil // Skip this directory
			}
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
			panic(err) // Handle errors accessing a path
		}
		s.Size = info.Size()
		s.ModTime = info.ModTime()

		fileList = append(fileList, s)
	}

	return nil
}

//go:embed migrations/*.sql
var migrations embed.FS
var fileList = []models.FinfoJSON{}

type Fdirectory struct {
	ID        uuid.UUID `json:"id" db:"id"`
	Directory string    `json:"directory " db:"directory"`
	IsDone    bool      `json:"isDone" db:"is_done"`
	CreatedAt time.Time `json:"createdAt" db:"created_at"`
	UpdatedAt time.Time `json:"updatedAt" db:"updated_at"`
}

var directory string
var directoryId uuid.UUID

func main() {

	var path string
	switch len(os.Args) {
	case 1:
		c := 1
		fmt.Println("Please provide a path")
		os.Exit(c)
	case 2:
		c := 2
		path = os.Args[1]
		path = strings.TrimSpace(path)
		path = strings.ReplaceAll(path, "/", "\\")
		if _, err := os.ReadDir(path); err != nil {
			os.Exit(c)
		}

	}

	directory = path
	err := insertIntoDirs(path)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error inserting into directory table: %v\n", err)
		os.Exit(1)
	}

	if err := JsonInsert(path); err != nil {
		fmt.Fprintf(os.Stderr, "invalid path: %v\n", err)
		os.Exit(1)
	}

	if err := updateDirs(path); err != nil {
		fmt.Fprintf(os.Stderr, "Error updating directory table: %v\n", err)
		os.Exit(1)
	}

}

func JsonInsert(path string) error {

	// walk through the directories
	err := filepath.WalkDir(path, visit)
	if err != nil {
		writeLog(&log)
		return fmt.Errorf("Error walking through directories: %w", err)
	}

	// insert into files
	stmt := filelistToSQL(fileList)

	fmt.Println("Count of items: ", len(stmt))
	fmt.Printf("There was %#v items to insert", stmt)
	fmt.Println()

	if err := insertToPostgres(stmt); err != nil {
		return fmt.Errorf("Error for insertToPostgres: %w", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(stmt))
	}

	// if err := insertIntoDirs(path); err != nil {
	// 	return fmt.Errorf("Error inserting into indexed directories: %w", err)
	// }

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

func insertToPostgres(stmt []byte) error {
	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	s := len(fileList) // Use fileList length, not stmt bytes

	const query = `INSERT INTO files (directory_id, data) VALUES ($1, $2::jsonb);`

	// Insert each file from fileList
	for _, file := range fileList {
		jsonData, err := json.Marshal(file)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshalling file: %v\n", err)
			continue
		}

		_, err = db.Exec(query, directoryId, string(jsonData))
		if err != nil {
			return fmt.Errorf("error inserting file: %w", err)
		}
	}

	if s == 0 {
		fmt.Println("No items to insert")
		os.Exit(1)
	}

	fmt.Printf("Inserted %d files\n", s)
	return nil
}

func filelistToSQL(f []models.FinfoJSON) []byte {
	// This function is no longer needed with the fixed insertToPostgres
	// But keeping it for compatibility
	var jsonData []byte
	for i := range f {
		j, err := json.Marshal(models.FinfoJSON{
			Directory: strings.ReplaceAll(f[i].Directory, "\\", "/"),
			Name:      f[i].Name,
			Ext:       f[i].Ext,
			IsDir:     f[i].IsDir,
			Size:      f[i].Size,
			ModTime:   f[i].ModTime,
		})
		if err != nil {
			fmt.Fprintf(os.Stderr, "Error marshalling to JSON: %v\n", err)
			continue
		}
		jsonData = append(jsonData, j...)
	}
	return jsonData
}

func VanillaRawReturn(q string, param string) error {

	var e error

	db, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer db.Close()

	tx, err := db.Begin()
	if e != nil {
		return err
	}

	err = tx.QueryRow(q, param).Scan(&directoryId)
	if err != nil {
		tx.Rollback() // Rollback on error
		return fmt.Errorf("failed to insert and scan: %w", err)
	}

	tx.Commit()

	return nil
}
