package main

import (
	"fmt"
	"gofiles/internal/utils"
	"os"

	"embed"

	_ "github.com/lib/pq"
)

const outputFile = "output.txt"

//go:embed migrations/*.sql
var migrations embed.FS

func SqlMigrations(filename string) error {
	// Read with full embedded path
	f, err := migrations.ReadFile(filename)
	if err != nil {
		return fmt.Errorf("failed to read migration file %s: %w", "migrations/"+filename, err)
	}

	if err := VanillaSql(f); err != nil {
		return fmt.Errorf("failed to execute migration %s: %w", filename, err)
	}

	return nil
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

func main() {

	if err := SqlMigrations("migrations/001_initial.sql"); err != nil {
		fmt.Fprintf(os.Stderr, "Error: migrations/001_initial.sql creating tables: %v\n", err)
		os.Exit(1)
	}

}
