package main

import (
	"gofiles/internal/sequel"
	"gofiles/utils"
	"log"
)

func SequelMigrations() error {

	initialMigration := sequel.InitialMigrations()

	if err := utils.VanillaSql(initialMigration, true); err != nil {
		return err
	}

	return nil
}

func main() {

	SequelMigrations()

	tables, err := GetExistingTables()
	if err != nil {
		log.Fatalf("Error getting existing tables: %v", err)
	}
	for table := range tables {
		log.Printf("Existing table: %s", tables[table])
	}
}

func GetExistingTables() ([]string, error) {
	query := `
        SELECT table_name 
        FROM information_schema.tables 
        WHERE table_schema = 'public' 
          AND table_type = 'BASE TABLE'
        ORDER BY table_name;
    `

	conn, err := utils.PgConn()
	if err != nil {
		return nil, err
	}
	defer conn.Close()

	rows, err := conn.Query(query)
	if err != nil {
		return nil, err
	}

	var tables []string
	for rows.Next() {
		var tableName string
		if err := rows.Scan(&tableName); err != nil {
			return nil, err
		}
		tables = append(tables, tableName)
	}

	return tables, nil
}
