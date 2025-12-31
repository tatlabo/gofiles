package main

import (
	"bufio"
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"net/http"
	"time"

	_ "modernc.org/sqlite"
)

type Website struct {
	Id        int          `db:"id"`
	SourceId  int          `db:"source_id"`
	URL       template.URL `db:"url"`
	Title     string       `db:"title"`
	Body      string       `db:"body"`
	Blob      []byte       `db:"blob"`
	CreatedAt time.Time    `db:"created_at"`
	Keywords  string       `db:"keywords"`
	Display   int          `db:"display"`
	Done      int          `db:"done"`
	MD5       string       `db:"md5"`
}

type SqlInit struct {
	Create string
	Config []string
	Delete string
}

func getWebsite(sql string, db *sql.DB, w *Website) error {

	timeStr := ""
	err := db.QueryRow(sql, w.URL).Scan(&w.Id, &w.URL, &w.Body, &timeStr, &w.Keywords, &w.Display)
	if err != nil {
		return err
	}

	w.CreatedAt, _ = time.Parse("2006-01-02 15:04:05", timeStr)

	return nil

}

func SelectAllWebsites(db *sql.DB, w *Website) (l []Website, err error) {

	sql := `SELECT id, url, body, created_at, keywords, display FROM daily;`

	timeStr := ""
	rows, err := db.Query(sql)
	if err != nil {
		return []Website{}, err
	}

	for rows.Next() {
		next := Website{}
		err := rows.Scan(&next.Id, &next.URL, &next.Body, &timeStr, &next.Keywords, &next.Display)
		if err != nil {
			return []Website{}, err
		}

		l = append(l, next)
	}

	return l, nil

}

func (w *Website) GetSourceWebsite(db *sql.DB) error {

	sql := `SELECT id, url, body, created_at, keywords, display FROM source WHERE url = ?;`
	err := getWebsite(sql, db, w)
	if err != nil {
		return err
	}

	return nil

}

func (w *Website) GetWebsite(db *sql.DB) error {

	sql := `SELECT id, url, body, created_at, keywords, display FROM daily WHERE url = ? ORDER BY created_at DESC LIMIT 1;`
	err := getWebsite(sql, db, w)
	if err != nil {
		return err
	}

	return nil

}

func (w *Website) ProcessWebsite() error {
	www := string(w.URL)
	res, err := http.Get(www)
	if err != nil {
		return fmt.Errorf("Error getting website: %w", err)
	}
	defer res.Body.Close()

	scanner := bufio.NewScanner(res.Body)
	doc := ""
	for scanner.Scan() {
		doc += scanner.Text() + "\n"
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("Error scanning website: %w", err)
	}

	w.Body = doc
	return nil
}

func (w *Website) AddWebsite(db *sql.DB) error {

	stmt := `INSERT OR IGNORE INTO daily (source_id, url, body, title, created_at, keywords, display) 
	VALUES (?, ?, ?, ?, ?, ?, ?) RETURNING id;`

	err := db.QueryRow(stmt, w.SourceId, w.URL, w.Body, w.Title, w.CreatedAt.Format("2006-01-02 15:04:05"), w.Keywords, w.Display).Scan(&w.Id)
	if err != nil {
		return err
	}

	return nil
}

func (w *Website) SourceToDb(ctx context.Context, db *sql.DB) error {

	stmt := `INSERT INTO source (url, body, created_at, keywords, display) VALUES (?, ?, ?, ?, ?) RETURNING id;`

	err := db.QueryRow(stmt, w.URL, w.Body, w.CreatedAt.Format("2006-01-02 15:04:05"), w.Keywords, w.Display).Scan(&w.Id)
	if err != nil {
		return err
	}

	return nil
}

func PragmaConfig(db *sql.DB) error {

	config := [3]string{
		`PRAGMA journal_mode = WAL;`,
		`PRAGMA foreign_keys = ON;`,
		`PRAGMA busy_timeout = 5000;`,
	}

	for _, pragma := range config {
		_, err := db.Exec(pragma)
		if err != nil {
			return err
		}
	}

	return nil

}
