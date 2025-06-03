package main

import (
	"database/sql"
	"fmt"
	"io/fs"
	"maps"
	"os"
	"path/filepath"

	"gofiles/utils"
	"strings"

	_ "github.com/lib/pq"
)

type Stmt []string
type Filename map[int]string
type FileNames []Filename

const connStr = "user=golang password=golang dbname=go host=localhost sslmode=disable"

var stmt = Stmt{}
var filenameList = FileNames{}

func (f *Stmt) AddStmt(s string) {
	*f = append(*f, s)
}

func (f Stmt) Insert() string {
	var s string
	switch len(f) {
	case 0:
		return ""
	case 1:
		s += string(f[len(f)-1])
	default:
		for i := 0; i < len(f)-1; i++ {
			s += fmt.Sprintf("('%s'), ", f[i])
		}
		s += fmt.Sprintf("('%s')", (f[len(f)-1]))
	}

	return fmt.Sprintf(`INSERT INTO directories (dir) VALUES %s ON CONFLICT (dir) DO UPDATE SET dir = EXCLUDED.dir;`, s)
}

func (f *Filename) AddStmt(m map[int]string) {
	maps.Copy((*f), m)
}

func insertDirToDb(path string, db *sql.DB) error {

	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS directories (id SERIAL PRIMARY KEY, dir TEXT NOT NULL UNIQUE);`)
	if err != nil {
		panic(err)
	}

	err = filepath.WalkDir(path, visit)
	if err != nil {
		panic(err)
	}
	_, err = db.Exec(stmt.Insert())
	if err != nil {
		panic(err)
	}

	return nil
}

func fetchDirectories(db *sql.DB, stmt string) map[int]string {

	dirWithId := make(map[int]string)

	query := stmt
	rows, err := db.Query(query)
	if err != nil {
		panic(err)
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var dir string
		err := rows.Scan(&id, &dir)
		if err != nil {
			panic(err)
		}
		dirWithId[id] = dir
	}

	if err = rows.Err(); err != nil {
		panic(err)
	}

	return dirWithId
}

func visit(path string, d fs.DirEntry, err error) error {

	if err != nil {
		return err
	}
	if d.IsDir() {
		stmt.AddStmt(path)
		// fmt.Printf("%s\n", stmtList)
	}

	return nil
}

func createFIlesTable(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS files (id SERIAL PRIMARY KEY, name TEXT, dir_id INTEGER, UNIQUE (name, dir_id));`)
	if err != nil {
		panic(err)
	}
	return nil
}

func main() {

	var path = os.Args[1]

	if _, err := os.ReadDir(path); err != nil {
		panic(err)
	}

	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	insertDirToDb(path, db)

	fetchedDirs := fetchDirectories(db, "SELECT id, dir FROM directories;")

	createFIlesTable(db)

	for k, v := range fetchedDirs {

		files, err := os.ReadDir(v)
		if err != nil {
			panic(err)
		}

		for _, f := range files {
			if !f.IsDir() {
				filenameList = append(filenameList, Filename{k: f.Name()})
			}
		}

	}

	// fmt.Println(filenameList)
	var insertList []string

	for i := range filenameList {
		for k, v := range filenameList[i] {
			insertList = append(insertList, fmt.Sprintf(`INSERT INTO files (name, dir_id) VALUES ('%s', %d) 
				ON CONFLICT(name, dir_id) DO UPDATE SET name = EXCLUDED.name, dir_id = EXCLUDED.dir_id;`, v, k))

		}
	}

	if _, err := db.Exec(fmt.Sprint(strings.Join(insertList, "\n"))); err != nil {
		panic(err)
	}

	var countDir, countFiles int

	if err = db.QueryRow(`SELECT Count(*) AS c FROM directories;`).Scan(&countDir); err != nil {
		panic(err)
	}
	if err = db.QueryRow(`SELECT Count(*) AS c FROM files;`).Scan(&countFiles); err != nil {
		panic(err)
	}

	fmt.Printf("Count in directories: %d\n", countDir)
	fmt.Printf("Count in files: %d\n", countFiles)
}
