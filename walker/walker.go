package main

import (
	"fmt"
	"gofiles/utils"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

const outputFile = "output.txt"

var skipDirectories = []string{".git", "node_modules", "tmp", "temp", ".vscode", ".idea", "vendor", "build", "dist", "__pycache__", ",bin", ".vite", "$SysReset", "$Windows.~WS", "OneDriveTemp", "AppData"}
var skipFiles = []string{".DS_Store", ".gitignore", ".gitattributes", ".gitmodules", "package-lock.json", "yarn.lock", "dpx", ".gitignore"}

type Finfo struct {
	Id       int       `db:"id"`
	ParentId int       `db:"parent_id"`
	Path     string    `db:"path"`
	Name     string    `db:"name"`
	Ext      string    `db:"ext"`
	IsDir    bool      `db:"is_dir"`
	Size     int64     `db:"size"`
	ModTime  time.Time `db:"mod_time"`
}

func (f *Finfo) String() string {
	return fmt.Sprintf("%s, %s, %s, %T\n", f.Path, f.Name, f.Ext, f.IsDir)
	// 	return fmt.Sprintf("Path: %s, Name: %s, Extension: %s, IsDir: %t, Size: %d, ModTime: %s",
	// 		f.Path, f.Name, f.Ext, f.IsDir, f.Size, f.ModTime.Format("1999-01-02 15:04:05"))
}

var fileList = []Finfo{}

var log = []string{}

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

		s := Finfo{}
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

	err := filepath.WalkDir(path, visit)
	if err != nil {
		fmt.Println(err)
		writeLog(&log)
	}

	if err := utils.CreateFiles(); err != nil {
		panic(err)
	}

	stmt := insertItem(fileList)
	fmt.Println("Count of items: ", len(stmt))

	if err := insertToPostgres(stmt); err != nil {
		fmt.Println("Error writing to file: ", err)
	} else {
		fmt.Printf("There was %d items inserted\n", len(stmt))
	}

	// stmt = simpleString(fileList)

	// if err := writeStmtToFile(stmt); err != nil {
	// 	fmt.Println("Error writing to file: ", err)
	// 	os.Exit(1)
	// }

	/* insert to postgres */
	// if err := insertToPostgres(stmt); err != nil {
	// 	fmt.Println("Error inserting to postgres: ", err)
	// 	os.Exit(1)
	// }

	/* write to file */

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

func insertToPostgres(stmt []string) error {
	db, err := utils.PgConn()
	if err != nil {
		panic(err)
	}
	defer db.Close()

	var (
		s        = len(stmt)
		counter  int
		quantity = 100
	)

	for {

		if s < quantity {

			if _, err = db.Exec(strings.Join(stmt[counter:], " ")); err != nil {
				panic(err)
			}

			fmt.Printf("Insetred %d : %d items\n", counter, s)
			break

		} else {
			if _, err := db.Exec(strings.Join(stmt[counter:counter+quantity], " ")); err != nil {
				fmt.Println("Error writing to database: ", err)
				fmt.Printf("%v", strings.Join(stmt[counter:counter+quantity], " "))
				os.Exit(1)
			}

			s -= quantity
			counter += quantity

		}
	}

	var cnt = len(stmt)

	if cnt == 0 {
		fmt.Println("No items to insert")
		os.Exit(1)
	} else if cnt < 5 {
		fmt.Println(stmt)
	} else {
		fmt.Println(stmt[len(stmt)-3:])
	}

	return nil
}

func simpleString(f []Finfo) []string {
	s := []string{}
	for _, item := range f {
		s = append(s, item.String())
	}

	return s
}

func insertItem(f []Finfo) []string {

	stmt := []string{}

	for i := range len(fileList) {
		inertItem := fmt.Sprintf(`
INSERT INTO files (path, name, ext, is_dir, size, mod_time) 
VALUES ('%s', '%s', '%s', %t, %d,'%v') ON CONFLICT (path, name, ext, is_dir) DO UPDATE SET size = EXCLUDED.size, mod_time = EXCLUDED.mod_time;`,
			fileList[i].Path, fileList[i].Name, fileList[i].Ext, fileList[i].IsDir, fileList[i].Size, fileList[i].ModTime.Format("2006-01-02 15:04:05"))

		stmt = append(stmt, inertItem)
	}

	return stmt
}
