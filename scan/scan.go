package main

import (
	"fmt"
	"gofiles/internal/models"
	"gofiles/internal/utils"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"embed"

	"github.com/google/uuid"
	_ "github.com/lib/pq"

	"encoding/json"
)

const outputFile = "output.txt"

var directory string
var directoryId uuid.UUID

//go:embed migrations/*.sql
var migrations embed.FS

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData", "Windows", "AppData", ".cargo", ".conda", ".config"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}

var fileList = []models.Finfo{}

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

	if err := VanillaSql(f); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", path, err)
	}

	return nil
}

func VanillaSqlReturn(q string, param string) error {

	var e error

	db, e := utils.PgConn()
	if e != nil {
		return e
	}
	defer db.Close()

	tx, e := db.Begin()
	if e != nil {
		return e

	}
	err := tx.QueryRow(q, param).Scan(&directoryId)
	if err != nil {
		tx.Rollback() // Rollback on error
		return fmt.Errorf("failed to insert and scan: %w", err)
	}

	return tx.Commit()
}

func VanillaSql(xs []byte) error {

	var e error

	db, e := utils.PgConn()
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

		s := models.Finfo{}
		s.Name = strings.TrimSuffix(d.Name(), extension)
		s.Name = strings.ReplaceAll(s.Name, "'", "''")

		s.IsDir = d.IsDir()
		s.Directory = filepath.Dir(path) // Handle errors accessing a path}
		s.Directory = strings.ReplaceAll(s.Directory, "\\", "/")
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

		s.DirectoryId = directoryId

		fileList = append(fileList, s)
	}

	return nil
}

func main() {

	var path string

	switch len(os.Args) {
	case 1:
		c := 1
		fmt.Println("Main case", c, "needs a path argument")
		os.Exit(c)
	case 2:
		c := 2
		path = os.Args[1]
		path = strings.TrimSpace(path)
		path = strings.ReplaceAll(path, "\\", "/")
		directory = path
		if _, err := os.ReadDir(path); err != nil {
			fmt.Fprintf(os.Stderr, "Main case %d invalid path: %v\n", c, err)
			os.Exit(1)
		}

	}

	if err := ScanDir(path); err != nil {
		fmt.Fprintf(os.Stderr, "invalid path: %v\n", err)
		os.Exit(1)
	}

}

func insertIntoDirs(path string) error {

	const insertIntoDirs = `INSERT INTO directory (name, done)
	VALUES ($1, $2) ON CONFLICT (name) DO UPDATE SET done = EXCLUDED.done;`

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	_, err = conn.Exec(insertIntoDirs, path, true)
	if err != nil {
		return err
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

func insertToPostgres(flist *[]models.Finfo) error {

	query := `INSERT INTO files (directory_id, data) VALUES ($1, $2);`

	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	var (
		s = len(*flist)
		// counter  int
		// quantity = 100
	)

	j := 0
	for _, obj := range *flist {

		objJSON := obj.ToJSON()
		data, _ := json.Marshal(objJSON)
		j++
		if j < 5 {
			fmt.Printf("%v %v %v", query, obj.DirectoryId, obj)
			fmt.Println()
		}
		if _, err := tx.Exec(query, obj.DirectoryId, data); err != nil {
			tx.Rollback()
			return fmt.Errorf("error inserting file: %w", err)
		}
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("error committing: %w", err)
	}

	fmt.Printf("Inserted %d of %d items\n", len(*flist), s)

	return nil
}

func insertItem(f []models.Finfo) [][]any {

	params := [][]any{}

	for i := range len(f) {

		params = append(params, []any{
			f[i].Directory,
			f[i].Name,
			f[i].Ext,
			f[i].IsDir,
			f[i].Size,
			f[i].ModTime,
			directoryId,
		})

	}

	return params
}

func ScanDir(dir string) error {

	var sqlInsertReturn = `INSERT INTO directory (json) VALUES ($1::jsonb) RETURNING id;`
	var path string

	path = strings.TrimSpace(dir)
	if _, err := os.ReadDir(path); err != nil {
		return fmt.Errorf("invalid path: %s", path)
	}

	// create tables
	// use init/table_init.go SqlMigrations
	// if err := SqlMigrations("migrations/001_initial.sql"); err != nil {
	// 	return fmt.Errorf("Error: migrations/001_initial.sql creating tables: %w", err)
	// }

	var pathJson = `{"path": "` + path + `"}`

	fmt.Println(sqlInsertReturn, fmt.Sprintf("%s", pathJson))

	if err := VanillaSqlReturn(sqlInsertReturn, pathJson); err != nil {
		return fmt.Errorf("Error inserting and/or returning value into directory table: %w", err)
	}

	// visit function = main scan logic
	err := filepath.WalkDir(path, visit)
	if err != nil {
		writeLog(&log)
		return fmt.Errorf("Error walking through directories: %w", err)
	}

	// stmt := insertItem(fileList)
	// fmt.Println("Count of items: ", len(stmt))

	fmt.Printf("%#v", fileList[:5])

	for i := 0; i < 5; i++ {
		obj := fileList[i]
		m, err := json.Marshal(obj)

		fmt.Println(string(m), err)
	}

	if err := insertToPostgres(&fileList); err != nil {
		return fmt.Errorf("Error for insertToPostgres: %w", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(fileList))
	}

	// if err := SqlMigrations("migrations/002_initial.sql"); err != nil {
	// 	return fmt.Errorf("Error for: migrations/002_initial.sql: update ext, keywords and ext_id: %w", err)
	// }

	// if err := SqlMigrations("migrations/003_initial.sql"); err != nil {
	// 	return fmt.Errorf("Error for: in migrations/003_initial.sql update files.ext_id: %w", err)
	// }

	// if err := insertIntoDirs(path); err != nil {
	// 	return fmt.Errorf("Error inserting into directory: %w", err)
	// }

	return nil

}
