package handlers

import (
	"gofiles/internal/models"
	"log"
	"net/http"
	"runtime"
	"time"
)

func HandleSearchGet(w http.ResponseWriter, r *http.Request) {

	templatePage := "index.html"

	switch r.Method {
	case http.MethodGet:
		templatePage = "home.html"
		{
			tmpl.Render(w, templatePage, IndexData{
				Title:    "Search files",
				Body:     map[string]string{"message": "Wyszukaj pliki po słowach kluczowych"},
				HomePage: true,
			})
		}

	case http.MethodPost:
		{
			start := time.Now()
			var dataStream chan any
			dataStream = make(chan any)
			defer close(dataStream)

			r.ParseForm()
			keywords := r.FormValue("name")

			limit := 10
			offset := 0

			if keywords == "" {
				tmpl.Render(w, templatePage, IndexData{Title: "My Title", Body: map[string]string{"message": "TNie podano słów kluczowych"}})
				return
			}

			data := models.FilesDataList{}

			count := make(chan int, 1)
			errCh := make(chan error)
			go func() {
				defer close(count)
				// defer wg.Done()
				if err := data.SelectCount(keywords); err != nil {
					errCh <- err
				}
				count <- data.Count
			}()

			select {
			case c := <-count:
				if c == 0 {
					templatePage = "home.html"
					tmpl.Render(w, templatePage,
						IndexData{Title: "No results",
							Body:         map[string]string{"message": "No results found for the given keywords"},
							SearchParams: map[string]string{"keywords": keywords}})
					return
				}
			case err := <-errCh:
				i := IndexData{Title: "Error page", Body: map[string]string{"err": err.Error(), "msg": "Error retrieving search results"}}
				tmpl.Render(w, "error.html", i)
				return
			}

			res := make(chan models.FilesDataList)
			resErr := make(chan error)
			go func() {
				defer close(res)
				defer close(resErr)

				// defer wg.Done()
				if err := data.GetList(keywords, limit, offset); err != nil {
					resErr <- err
				}

				res <- data
			}()

			select {
			case err := <-resErr:
				i := IndexData{Title: "Error page", Body: map[string]string{"err": err.Error(), "msg": "Error retrieving search results"}}
				tmpl.Render(w, "error.html", i)
				return
			case result := <-res:
				for index := range result.List {
					result.List[index].SimplifyDetails()
				}
				tmpl.Render(w, templatePage, IndexData{
					Title:         "My Title",
					Body:          map[string]string{"message": "data", "keywords": keywords},
					FilesDataList: result,
					Count:         result.Count,
					SearchParams:  map[string]string{"keywords": keywords, "limit": "10", "offset": "0"},
				})

			}

			log.Printf("\n\n\nSearch completed in %v\n\n", time.Since(start))
			log.Printf("nr of gorutines (after Render): %v", runtime.NumGoroutine())
			return
		}
	}
}
