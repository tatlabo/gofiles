package application

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"net/http"
	"os"
	"strconv"
	"time"

	"gofiles/chroma"
	"gofiles/internal/filter"
	"gofiles/internal/models"
)

type SimpleReq struct{}

func (s SimpleReq) FillData(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Index Data Page nr 555"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Accept-Language", "pl-PL")
	w.WriteHeader(404)
	w.Header().Add("X-Content-Type-Options", "nosniff")
}

type IndexPage struct {
	Title   string
	HTML    template.HTML
	Message string
	Count   int
	Result  []models.FileData
	QP      filter.QueryParams
}

func (s SimpleReq) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.FillData(w, r)
}

func (app *Application) HandleSearch(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)

	templatePage := "index.html"

	r = r.WithContext(ctx)
	defer cancel()

	qp := processQueryParams(r)
	filter := filter.Parse(r)

	// home page
	if filter.Keywords == "" {
		app.Render(w, "home.html", IndexPage{})
		return
	}

	list, count, err := app.File.Rows(qp.Keywords, qp.Limit, qp.Offset)
	if err != nil {
		w.WriteHeader(500)
		// TO DO
		w.Write([]byte("error in getlist" + err.Error()))
		return
	}

	if count == 0 {
		app.Render(w, "home.html", IndexPage{
			Title:   "No results",
			Message: "No results found for the given keywords",
			QP:      filter,
		})
		return
	}

	indexPage := IndexPage{
		Title:  fmt.Sprintf("wyszukano %d plików", count),
		Count:  count,
		QP:     filter,
		Result: list,
	}

	app.Render(w, templatePage, indexPage)
}

func (app *Application) FileDetails(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.Log.Warn("ItemDetailsID", "wrong ID", err)
		http.Error(w, "Invalid ID", http.StatusNotFound)
		return
	}

	f, err := app.File.Row(id)
	if err != nil {
		app.Log.Debug("FileDetails", "error", err)
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	tmpl.Render(w, "entry-details", f)
}

func (app *Application) WrongID(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	htmx := r.Header.Get("Hx-Request") != "true"
	app.Log.Warn("wrong Id format", "ID", idStr)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !htmx {
		w.WriteHeader(http.StatusBadRequest)
	}
	msg := fmt.Sprintf(`<div class="alert alert-error">Invalid ID format: <strong>%s</strong></div>`, idStr)
	_, _ = w.Write([]byte(msg))
}

func (app *Application) NotFound(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	ID, _ := strconv.Atoi(idStr)
	htmx := r.Header.Get("Hx-Request") != "true"
	app.Log.Warn("not found file", "ID", ID)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if !htmx {
		w.WriteHeader(http.StatusNotFound)
	}
	w.Write([]byte(`<div class="alert alert-error">record not found</div>`))
}

func (app *Application) Preview(w http.ResponseWriter, r *http.Request) {
	type PageData struct {
		File  models.FileData
		Title string
		Body  string
		HTML  template.HTML
	}

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.WrongID(w, r)
		return
	}

	f, err := app.File.Row(id)
	if err != nil {
		app.NotFound(w, r)
		return
	}
	app.Log.Info("preview", "file", f)

	pageData := PageData{
		Title: "Details",
		File:  f,
		Body:  fmt.Sprintf("Detail Page for ID: %v", f.ID),
	}

	//
	if f.Type != "txt" {
		tmpl.Render(w, "preview.html", pageData)
		return
	}

	// chroma preview
	address := fmt.Sprintf("%s\\%s.%s", pageData.File.Directory, pageData.File.Name, pageData.File.Ext)
	// file to chroma
	html, err := TxtToChoroma(address)
	if err != nil {
		http.Error(w, "Error converting text to choroma", http.StatusInternalServerError)
		return
	}

	_ = fmt.Sprintf("<div><h3>Text to chroma</h3><p>%s</p></div>", html)
	pageData.HTML = html

	tmpl.Render(w, "chroma-preview.html", pageData)
}

func processQueryParams(r *http.Request) models.QueryParams {
	qp := models.QueryParams{}
	switch r.Method {
	case http.MethodGet:
		qp.Method = "get"
		query := r.URL.Query()

		lStr := query.Get("limit")
		offsetStr := query.Get("offset")
		ascending := query.Get("ascending")

		s := ""
		if k, ok := query["keywords"]; ok && len(k) > 0 {
			s = k[0]
		} else if k, ok := query["keyword"]; ok && len(k) > 0 {
			s = k[0]
		} else if k, ok := query["search"]; ok && len(k) > 0 {
			s = k[0]
		} else if k, ok := query["name"]; ok && len(k) > 0 {
			s = k[0]
		}

		qp.Keywords = s
		qp.Limit, _ = strconv.Atoi(lStr)
		qp.Offset, _ = strconv.Atoi(offsetStr)
		qp.Order = query.Get("order")
		qp.Ascending = ascending == "true"

		if qp.Limit <= 0 {
			qp.Limit = 10
		}
		if qp.Offset < 0 {
			qp.Offset = 0
		}

	case http.MethodPost:
		qp.Method = "post"
		qp.Keywords = r.FormValue("keywords")
		qp.Limit = 10
		qp.Offset = 0
	}

	// log.Printf("Processed query params: %#v", qp)
	return qp
}

func (app *Application) HandleAppend(w http.ResponseWriter, r *http.Request) {
	filter := filter.Parse(r)

	qp := models.QueryParams{
		Keywords:  filter.Keywords,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		Order:     filter.Order,
		Ascending: filter.Ascending,
		Method:    filter.Method,
	}

	result, err := app.File.AppendListParams(qp)
	if err != nil {
		app.Log.Debug("getting search list", "fail", err)
		tmpl.Render(w, "index.html", IndexPage{Title: "Fetch error", Message: "Error retrieving search results"})
		return
	}

	tmpl.Render(
		w, "list", IndexPage{
			Title:  "My Title",
			QP:     filter,
			Result: result,
		},
	)
}

func (app *Application) PreviewMedia(w http.ResponseWriter, r *http.Request) {
	type PageData struct {
		File  models.FileData
		Title string
		Err   error
		HTML  template.HTML
	}

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		app.WrongID(w, r)
		return
	}

	f, err := app.File.Row(id)
	if err != nil {
		app.NotFound(w, r)
		return
	}

	base := "media"

	srcPath := fmt.Sprintf("%v/%v.%v", f.Directory, f.Name, f.Ext)
	destPath := fmt.Sprintf("%s/%s/%s.%v", base, f.Type, f.Name, f.Ext)

	if err := os.MkdirAll(base+"/"+f.Type, os.ModePerm); err != nil {
		app.Log.Error("preview media error", "media", err)
		tmpl.Render(w, "error.html", PageData{
			Title: "preview media error",
			Err:   fmt.Errorf("error in media/video directory copy/creation %w", err),
		})
		return
	}
	/*
		Copy the image file
	*/
	if err := copyImageFile(srcPath, destPath); err != nil {
		app.Log.Error("err copy file to media folder", "source", srcPath)
		tmpl.Render(w, "error.html", PageData{
			Title: "preview media error",
			Err:   fmt.Errorf("error copy to media folder %w", err),
		})
		return
	}

	go func(filePath string) {
		time.Sleep(120 * time.Second)
		if err := os.Remove(filePath); err != nil {
			app.Log.Error("Error deleting file", destPath, err)
		} else {
			app.Log.Info("File", destPath, "deleted successfully")
		}
	}(destPath)

	f.URL = template.URL("/" + destPath)

	pageData := PageData{
		Title: "Media Preview",
		File:  f,
	}

	tmpl.Render(w, "preview-media.html", pageData)
}

func copyImageFile(srcPath, destPath string) error {
	// Open the source file
	srcFile, err := os.Open(srcPath)
	if err != nil {
		return err
	}
	defer srcFile.Close()

	// Create the destination file
	destFile, err := os.Create(destPath)
	if err != nil {
		return err
	}
	defer destFile.Close()

	// Copy the content from source to destination
	_, err = io.Copy(destFile, srcFile)
	if err != nil {
		return err
	}

	// Ensure the destination file is properly written to disk
	err = destFile.Sync()
	if err != nil {
		return err
	}

	return nil
}

func TxtToChoroma(address string) (template.HTML, error) {
	fin, err := os.Open(address)
	if err != nil {
		return "", err
	}
	defer fin.Close()

	highlightCode, err := chroma.HighlightCode(address)
	if err != nil {
		return "", err
	}

	return template.HTML(highlightCode), nil
}
