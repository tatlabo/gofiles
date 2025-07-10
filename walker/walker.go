package main

import (
	"fmt"
	"gofiles/models"
	"gofiles/utils"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/lib/pq"
)

const outputFile = "output.txt"

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}

var query = `INSERT INTO files (path, name, ext, is_dir, size, mod_time) 
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (path, name, ext, is_dir) DO UPDATE SET size = EXCLUDED.size, mod_time = EXCLUDED.mod_time;`

var fileList = []models.Finfo{}

var log = []string{}

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

const creteIndexOnExt = `
CREATE INDEX IF NOT EXISTS idx_ext ON files(LOWER(ext));`

const insertIntoExt = `INSERT INTO ext (ext)
SELECT DISTINCT ext FROM files WHERE files.ext IS NOT NULL ON CONFLICT (ext) DO NOTHING;`

const updateExtId = `
UPDATE files SET ext_id = ext.id FROM ext WHERE files.ext = ext.ext AND files.is_dir = FALSE;`

const updateKeywords = `
UPDATE files SET keywords = ( to_tsvector('polish', (name||' '||ext)));`

const createGinOnKeywords = `
CREATE INDEX IF NOT EXISTS idx_keywords_gin ON files USING GIN (to_tsvector('polish', keywords));`

func CreateFiles() error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	defer tx.Rollback()

	if _, err := tx.Exec(createExt); err != nil {
		return err
	}

	if _, err := tx.Exec(createFiles); err != nil {
		return err
	}

	return tx.Commit()
}

func CreateIndexes() error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	tx, err := db.Begin()
	if err != nil {
		return err
	}

	if _, err := tx.Exec(createGinOnKeywords); err != nil {
		return err
	}

	if _, err := tx.Exec(creteIndexOnExt); err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateFiles() error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	//transactions
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	if _, err := tx.Exec(insertIntoExt); err != nil {
		return err
	}

	if _, err := tx.Exec(updateKeywords); err != nil {
		return err
	}

	return tx.Commit()
}

func UpdateExtId() error {

	db, err := utils.PgConn()
	if err != nil {
		return (err)
	}
	defer db.Close()

	if _, err := db.Exec(updateExtId); err != nil {
		return err
	}

	return nil
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
		if _, err := os.ReadDir(path); err != nil {
			os.Exit(c)
		}
	}

	if err := CreateFiles(); err != nil {
		panic(err)
	}

	if err := CreateIndexes(); err != nil {
		panic(err)
	}

	err := filepath.WalkDir(path, visit)
	if err != nil {
		fmt.Println(err)
		writeLog(&log)
	}

	stmt := insertItem(fileList)
	fmt.Println("Count of items: ", len(stmt))

	if err := insertToPostgres(stmt); err != nil {
		fmt.Println("Error writing to file: ", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(stmt))
	}

	if err := UpdateFiles(); err != nil {
		panic(err)
	}

	if err := UpdateExtId(); err != nil {
		panic(err)
	}

}

func writeLog(log *[]string) error {

	f, err := os.Create("log.txt")
	if err != nil {
		return err
	}

	defer f.Close()

	if _, err := f.WriteString(fmt.Sprintf("%v\n", log)); err != nil {
		return err
	}

	return nil
}

func insertToPostgres(stmt [][]interface{}) error {
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

	// for {

	// 	if s < quantity {

	// 		if _, err = db.Exec(query, stmt[counter:]); err != nil {
	// 			panic(err)
	// 		}

	// 		fmt.Printf("Insetred %d : %d items\n", counter, s)
	// 		break

	// 	} else {
	// 		if _, err := db.Exec(strings.Join(stmt[counter:counter+quantity], " ")); err != nil {
	// 			fmt.Println("Error writing to database: ", err)
	// 			fmt.Printf("%v", strings.Join(stmt[counter:counter+quantity], " "))
	// 			os.Exit(1)
	// 		}

	// 		s -= quantity
	// 		counter += quantity

	// 	}
	// }

	for _, p := range stmt {
		_, err := db.Exec(query, p...)
		if err != nil {
			// handle error
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

func insertItem(f []models.Finfo) [][]interface{} {

	params := [][]interface{}{}

	for i := range len(f) {

		params = append(params, []interface{}{
			f[i].Path,
			fileList[i].Name,
			f[i].Ext,
			f[i].IsDir,
			f[i].Size,
			f[i].ModTime,
		})

	}

	return params
}
