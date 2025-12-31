package main

import (
	"context"
	"database/sql"
	"fmt"
	"html/template"
	"log"
	"strings"
	"time"

	"github.com/antchfx/htmlquery"
)

func main() {
	db, err := sql.Open("sqlite", "./websites.db")
	if err != nil {
		panic(err)
	}
	defer db.Close()

	w := Website{URL: "https://rmf24.pl"}

	err = w.GetSourceWebsite(db)
	if err != nil {
		log.Fatal(err)
	}

	if w.Body != "" {
		subpages, err := ParseSourceBody(&w)
		if err != nil {
			log.Fatal(err)
		}

		l := len(subpages)
		if l < 0 {
			log.Println("No new entries found.")
		} else {

			for i := range subpages {

				err := (subpages[i]).AddWebsite(db)
				if err != nil {
					log.Fatal("There are no new entries ", err)
				}
			}
		}

	}

}

func SourceWebsite(ctx context.Context, db *sql.DB, w *Website) error {

	CreateSourceTable(db)
	CreateArticleTable(db)

	w.URL = "https://rmf24.pl"
	w.CreatedAt = time.Now()
	w.Keywords = "rmf24"
	w.Display = 0

	if err := w.ProcessWebsite(); err != nil {
		return fmt.Errorf("failed to process source website: %w", err)
	}

	if err := w.SourceToDb(ctx, db); err != nil {
		return fmt.Errorf("failed to insert source website to db: %w", err)
	}

	return nil
}

func ParseSourceBody(w *Website) ([]Website, error) {

	doc, err := htmlquery.Parse(strings.NewReader(w.Body))
	if err != nil {
		return nil, err
	}
	list := htmlquery.Find(doc, "//div/div/h3/span")

	subpages := []Website{}

	for _, node := range list {
		subpage := Website{}

		a := htmlquery.FindOne(node, "//a")
		title := htmlquery.InnerText(a)
		title = strings.TrimSpace(title)
		link := htmlquery.SelectAttr(a, "href")

		subpage.Title = title
		subpage.URL = template.URL(link)
		subpage.CreatedAt = time.Now()

		subpage.SourceId = w.Id
		subpages = append(subpages, subpage)

	}

	return subpages, nil
}

func AddWebsite(ctx context.Context, db *sql.DB, w *Website) error {

	if err := w.ProcessWebsite(); err != nil {
		log.Fatal(err)
	}

	if err := w.AddWebsite(db); err != nil {
		log.Fatal(err)
	}

	fmt.Println("Database initialized and data inserted successfully.")
	return nil
}
