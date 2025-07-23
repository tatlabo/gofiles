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

	_ "github.com/lib/pq"
)

const outputFile = "output.txt"

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}

var query = `INSERT INTO register (body) VALUES ($1);`

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

func VanillaSql(xs []byte) error {

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
		s.Path = filepath.Dir(path) // Handle errors accessing a path}
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

	if err := JsonInsert(path); err != nil {
		fmt.Fprintf(os.Stderr, "invalid path: %v\n", err)
		os.Exit(1)
	}

}

func insertIntoDirs(path string) error {

	const insertIntoDirs = `INSERT INTO indexed (path, done)
	VALUES ($1, $2) ON CONFLICT (path) DO UPDATE SET done = EXCLUDED.done;`

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

func insertToPostgres(stmt []string) error {
	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var (
		s = len(stmt)
		// counter  int
		// quantity = 100
	)

	for i := range stmt {
		_, err := db.Exec(query, i)
		if err != nil {
		}
	}

	if s == 0 {
		fmt.Println("No items to insert")
		os.Exit(1)
	} else if s < 5 {
		fmt.Println(stmt)
	} else {
		fmt.Println(stmt[s-3:])
	}

	return nil
}

func insertItem(f []models.Finfo) []string {
	var json []string
	for i := range len(f) {

		json = append(json, fmt.Sprintf(`'{"path":  "%s",
			"name":  "%s",
			"ext":   "%s",
			"is_dir": "%v",
			"size":  "%d",
			"mod_time": "%s"}'`, strings.ReplaceAll(f[i].Path, "\\", "/"), f[i].Name, f[i].Ext, f[i].IsDir, f[i].Size, f[i].ModTime))

	}

	return json
}

func ScanDir(dir string) error {

	var path string

	path = strings.TrimSpace(dir)
	// path = strings.ReplaceAll(path, "/", "\\")
	if _, err := os.ReadDir(path); err != nil {
		return fmt.Errorf("invalid path: %s", path)

	}

	// create tables
	if err := SqlMigrations("migrations/001_initial.sql"); err != nil {
		return fmt.Errorf("Error: migrations/001_initial.sql creating tables: %w", err)
	}

	// walk through the directories
	err := filepath.WalkDir(path, visit)
	if err != nil {
		writeLog(&log)
		return fmt.Errorf("Error walking through directories: %w", err)
	}

	// insert into files
	stmt := insertItem(fileList)
	fmt.Println("Count of items: ", len(stmt))

	if err := insertToPostgres(stmt); err != nil {
		return fmt.Errorf("Error for insertToPostgres: %w", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(stmt))
	}

	// update ext, keywords and ext_id
	if err := SqlMigrations("migrations/002_initial.sql"); err != nil {
		return fmt.Errorf("Error for: migrations/002_initial.sql: update ext, keywords and ext_id: %w", err)
	}

	// update files.ext_id
	if err := SqlMigrations("migrations/003_initial.sql"); err != nil {
		return fmt.Errorf("Error for: in migrations/003_initial.sql update files.ext_id: %w", err)
	}

	if err := insertIntoDirs(path); err != nil {
		return fmt.Errorf("Error inserting into indexed directories: %w", err)
	}

	return nil

}

func JsonInsert(dir string) error {

	var path string

	path = strings.TrimSpace(dir)
	path = strings.ReplaceAll(path, "\\", "/")
	if _, err := os.ReadDir(path); err != nil {
		return fmt.Errorf("invalid path: %s", path)

	}

	// walk through the directories
	err := filepath.WalkDir(path, visit)
	if err != nil {
		writeLog(&log)
		return fmt.Errorf("Error walking through directories: %w", err)
	}

	// insert into files
	stmt := insertItem(fileList)
	fmt.Println("Count of items: ", len(stmt))

	fmt.Printf("There was %#v items to insert\n", stmt)

	if err := insertToPostgres(stmt); err != nil {
		return fmt.Errorf("Error for insertToPostgres: %w", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(stmt))
	}

	if err := insertIntoDirs(path); err != nil {
		return fmt.Errorf("Error inserting into indexed directories: %w", err)
	}

	return nil

}
