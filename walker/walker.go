package main

import (
	"fmt"
	"gofiles/internal/models"
	"gofiles/internal/utils"
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

func CreateTables() []string {

	var t = []string{}

	createExt := `CREATE TABLE IF NOT EXISTS ext (
	id SERIAL PRIMARY KEY,
	ext TEXT UNIQUE);`

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

	const searchedWords = `
	CREATE TABLE IF NOT EXISTS search
	(id SERIAL PRIMARY KEY,
	input TEXT,
	created TIMESTAMPTZ NOT NULL DEFAULT NOW());`

	const indexedDirs = `CREATE TABLE IF NOT EXISTS indexed (
	id SERIAL PRIMARY KEY,
	path TEXT NOT NULL UNIQUE,
	done BOOLEAN NOT NULL DEFAULT FALSE,
	created TIMESTAMPTZ NOT NULL DEFAULT NOW());`

	t = append(t, createExt, createFiles, searchedWords, indexedDirs)

	return t
}

func CreateIndexes() []string {
	var t = []string{}

	const creteIndexOnExt = `
	CREATE INDEX IF NOT EXISTS idx_ext ON files(LOWER(ext));`
	const createGinOnKeywords = `
	CREATE INDEX IF NOT EXISTS idx_keywords_gin ON files USING GIN (to_tsvector('polish', keywords));`

	t = append(t, creteIndexOnExt, createGinOnKeywords)
	return t
}

func InsertExtKeywords() []string {

	var t = []string{}

	const insertIntoExt = `INSERT INTO ext (ext)
	SELECT DISTINCT ext FROM files WHERE files.ext IS NOT NULL ON CONFLICT (ext) DO NOTHING;`
	const updateKeywords = `
	UPDATE files SET keywords = ( to_tsvector('polish', (translate(name, ',._-+', '     ')||' '||ext)));`
	t = append(t, insertIntoExt, updateKeywords)

	return t
}

func UpdateExtId() []string {
	var t = []string{`UPDATE files SET ext_id = ext.id FROM ext WHERE files.ext = ext.ext AND files.is_dir = FALSE;`}
	return t
}

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
		path = strings.TrimSpace(path)
		path = strings.ReplaceAll(path, "/", "\\")
		if _, err := os.ReadDir(path); err != nil {
			os.Exit(c)
		}
	}

	createTables := CreateTables()
	if err := CommitSql(createTables); err != nil {
		fmt.Println("Error creating tables: ", err)
		os.Exit(1)
	}

	createIndexes := CreateIndexes()
	if err := CommitSql(createIndexes); err != nil {
		fmt.Println("Error creating indexes: ", err)
		os.Exit(1)
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

	insertExtKeywords := InsertExtKeywords()
	if err := CommitSql(insertExtKeywords); err != nil {
		fmt.Println("Error inserting extension keywords: ", err)
		os.Exit(1)
	}

	updateExtId := UpdateExtId()
	if err := CommitSql(updateExtId); err != nil {
		fmt.Println("Error updating extension IDs: ", err)
		os.Exit(1)
	}

	if err := insertIntoDirs(path); err != nil {
		fmt.Println("Error inserting into indexed directories: ", err)
	}

}

func insertIntoDirs(path string) error {

	const insertIntoDirs = `INSERT INTO indexed (path, done)
	VALUES ($1, $2) ON CONFLICT (path) DO NOTHING;`

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
			f[i].Directory,
			fileList[i].Name,
			f[i].Ext,
			f[i].IsDir,
			f[i].Size,
			f[i].ModTime,
		})

	}

	return params
}
