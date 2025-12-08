package main

import (
	"database/sql"
	"fmt"
	"math"
	"math/rand"

	_ "modernc.org/sqlite"
)

func run() error {
	// db, err := sql.Open("sqlite", ":memory:")
	db, err := sql.Open("sqlite", "db.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE x(id INTEGER PRIMARY KEY, name TEXT)")
	if err != nil {
		return err
	}

	words := []string{"go", "python", "java", "csharp", "ruby", "algorithm", "variable", "function", "loop", "recursion", "array", "object", "class", "inheritance", "polymorphism", "interface", "compiler", "interpreter", "syntax", "debugger", "exception", "stack", "queue", "hashmap", "pointer", "reference", "module", "package", "library", "framework", "API", "REST", "GraphQL", "JSON", "XML", "token", "encryption", "authentication", "authorization", "CI/CD", "version control", "Git", "branch", "merge", "commit", "container", "Docker", "Kubernetes", "cloud", "virtualization", "thread", "process", "concurrency", "parallelism"}

	randowmWord := func() string {

		random := rand.Float64() * float64(len(words))
		random = math.Floor(random)
		randomInt := int(random)

		return words[randomInt]
	}

	for range 1000 {
		_, err = db.Exec("INSERT INTO x(name) VALUES (?)", randowmWord())
		if err != nil {
			return err
		}
	}

	stmt := `SELECT id, name FROM x WHERE name = 'go';`
	rows, err := db.Query(stmt)
	if err != nil {
		return err
	}
	defer rows.Close()

	for rows.Next() {
		var id int
		var name string
		if err := rows.Scan(&id, &name); err != nil {
			return err
		}

		fmt.Println(id, name)
	}

	return nil
}

// func TestDB(t *testing.T) {
// 	db, err := sql.Open("sqlite3", "./database.db")
// 	if err != nil {
// 		t.Fatal(err)
// 	}
// 	defer db.Close()
// }

func main() {
	if err := run(); err != nil {
		fmt.Println("Error:", err)
	}
}
