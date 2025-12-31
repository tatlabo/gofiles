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

	ctx := context.Background()

	CreateSourceTable(db)
	CreateArticleTable(db)

	w := Website{
		URL:       "https://rmf24.pl/",
		CreatedAt: time.Now(),
		Keywords:  "rmf24",
		Display:   0,
	}

	if err := w.ProcessWebsite(); err != nil {
		log.Fatal(err)
	}

	if err := w.SourceToDb(ctx, db); err != nil {
		log.Fatal(err)
	}

	// if w.Id != 0 {

	// 	err = w.GetSourceWebsite(db)
	// 	if err != nil {
	// 		log.Fatal(err)
	// 	}
	// }

	if w.Body != "" {
		subpages, err := ParseSourceBody(&w)
		if err != nil {
			log.Fatal(err)
		}

		for i := range subpages {
			fmt.Println(subpages[i].Title)

			err := (subpages[i]).AddWebsite(db)
			if err != nil {
				log.Fatal(err)
			}
		}
	}
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
		link = string(w.URL) + link
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
