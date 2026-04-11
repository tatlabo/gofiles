package handlers

import (
	"context"
	"fmt"
	"gofiles/chroma"
	"gofiles/internal/models"
	"gofiles/utils"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"
)

type Template struct {
	templates *template.Template
}

//https://www.youtube.com/watch?v=0x_oUlxzw5A&t=64s

func (t *Template) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var tmpl = Template{
	templates: template.Must(template.New("").Funcs(template.FuncMap{
		"formatDate": utils.FormatDate,
		"not":        utils.Not,
		"equals":     utils.Equals,
		"notequals":  utils.Notequals,
	}).ParseGlob("public/views/*.html")),
}

type SimpleReq struct{}

func (s SimpleReq) FillData(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("Index Data Page nr 555"))
	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Accept-Language", "pl-PL")
	w.WriteHeader(404)
	w.Header().Add("X-Content-Type-Options", "nosniff")

}

type IndexData struct {
	models.FilesDataList
	SearchParams models.QueryParams
	FileData     models.FileData
	Title        string
	Body         map[string]string
	Count        int
	HomePage     bool
	Html         template.HTML
	IsText       bool
}

func (i SimpleReq) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	i.FillData(w, r)
}

func emptyKeywords(qp models.QueryParams) (data IndexData, empty bool) {

	if len(qp.Keywords) == 0 {
		data.Title = "My Title"
		data.Body = map[string]string{"message": "Nie podano słów kluczowych"}
		data.Title = "Search files"
		data.Body = map[string]string{"message": "Wyszukaj pliki po słowach kluczowych"}
		data.HomePage = true
		return data, true
	}

	return data, false

}

func Wraper(w http.ResponseWriter, r *http.Request) {

	ctx := r.Context()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)

	r = r.WithContext(ctx)
	defer cancel()

	qp := processQueryParams(r)
	if data, empty := emptyKeywords(qp); empty {
		tmpl.Render(w, "home.html", data)
		return
	}

	log.Printf("Received request for %s with paramas=%v", r.URL.Path, qp)

	dataCh := make(chan IndexData, 1)
	errCh := make(chan error, 1)

	go func(p models.QueryParams) {
		defer close(dataCh)
		data, err := mainSearch(ctx, p)
		dataCh <- data
		errCh <- err
	}(qp)

	select {
	case <-ctx.Done():
		log.Println("Request timed out in Wraper")
		http.Error(w, "Request timed out", http.StatusGatewayTimeout)
		return
	case err := <-errCh:
		data := IndexData{
			Title: "Error page",
			Body:  map[string]string{"err": err.Error(), "msg": fmt.Sprintf("%v", err)},
		}
		tmpl.Render(w, "error.html", data)
	case data := <-dataCh:
		if data.Count != 0 {
			tmpl.Render(w, "index.html", data)
			return
		} else {
			templatePage := "home.html"
			data = IndexData{Title: "No results",
				Body:         map[string]string{"message": "No results found for the given keywords"},
				SearchParams: qp}
			tmpl.Render(w, templatePage, data)
			return
		}

	}

}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	Wraper(w, r)
}

func mainSearch(ctx context.Context, qp models.QueryParams) (IndexData, error) {

	data := models.FilesDataList{}

	if err := data.SelectCount(qp.Keywords); err != nil {
		return IndexData{}, err
	}

	if data.Count == 0 {
		return IndexData{Title: "No results",
			Body:         map[string]string{"message": "No results found for the given keywords"},
			SearchParams: qp}, nil
	}

	// main SELECT
	if err := data.GetList(qp.Keywords, qp.Limit, qp.Offset); err != nil {
		return IndexData{}, err
	}

	for index := range data.List {
		data.List[index].SimplifyDetails()
	}

	return IndexData{
		SearchParams:  qp,
		Title:         "My Title",
		Body:          map[string]string{"message": "data"},
		FilesDataList: data,
		Count:         data.Count,
	}, nil

}

func ItemDetailsId(w http.ResponseWriter, r *http.Request) {

	var f models.FileData

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)

	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := f.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	f.SimplifyDetails()

	tmpl.Render(w, "entry-details.html", f)

}

func DetailsId(w http.ResponseWriter, r *http.Request) {

	body, err := getItem(w, r)
	if err != nil {
		i := IndexData{Title: "Error page", Body: map[string]string{"err": err.Error(), "msg": "ItemDetailsId Error"}}
		tmpl.Render(w, "error.html", i)
		return
	}
	tmpl.Render(w, "simple-entry-details.html", body)

}

func getItem(w http.ResponseWriter, r *http.Request) (models.FileData, error) {

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	f := models.FileData{}

	if err != nil {
		return f, err
	}

	if err := f.GetById(id); err != nil {
		return f, err
	}

	f.SimplifyDetails()

	return f, nil
}

func getItemById(w http.ResponseWriter, r *http.Request) (IndexData, error) {

	f, err := getItem(w, r)
	if err != nil {
		http.Error(w, "Error retrieving file details in getItemById", http.StatusInternalServerError)
		return IndexData{}, err
	}

	body := IndexData{
		Title:    "Details",
		FileData: f,
		Body:     map[string]string{"message": "Detail Page", "id": fmt.Sprintf("%v", f.Id)},
	}

	return body, nil
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

func HandleAppend(w http.ResponseWriter, r *http.Request) {

	qp := processQueryParams(r)

	data := models.FilesDataList{}

	log.Printf("qp: %#v", qp)

	if len(qp.Order) > 0 {
		if err := data.AppendListParams(qp); err != nil {
			fmt.Println("Error getting search list: ", err)
			tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
			return
		}
	} else {
		if err := data.AppendList(qp); err != nil {
			fmt.Println("Error getting search list: ", err)
			tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
			return
		}
	}

	tmpl.Render(w, "append.html", IndexData{Title: "My Title",
		Body:          map[string]string{"message": "data"},
		FilesDataList: data,
		Count:         data.Count,
		SearchParams:  qp},
	)
	log.Printf("nr of gorutines (HandleAppend): %v", runtime.NumGoroutine())

}

func PreviewImage(w http.ResponseWriter, r *http.Request) {

	var f models.FileData
	var i IndexData

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := f.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	f.SimplifyDetails()

	srcPath := fmt.Sprintf("%v/%v.%v", f.Directory, f.Name, f.Ext)
	destPath := fmt.Sprintf("media/images/%s.%v", f.Name, f.Ext)

	fmt.Println(srcPath)
	fmt.Println(destPath)

	// Ensure the destination directory exists
	if err := os.MkdirAll("media/images", os.ModePerm); err != nil {
		i.Title = "Error page"
		i.Body = map[string]string{"err": err.Error(), "msg": "Error in media/images directory creation"}
		tmpl.Render(w, "error.html", i)
		return
	}
	// Copy the image file
	if err := CopyImageFile(srcPath, destPath); err != nil {
		i.Title = "Error page"
		i.Body = map[string]string{"err": err.Error(), "msg": "Error copying image file"}
		tmpl.Render(w, "error.html", i)
		return
	}

	go func(filePath string) {
		time.Sleep(120 * time.Second)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error deleting file %s: %v", filePath, err)
		} else {
			log.Printf("File %s deleted successfully", filePath)
		}
	}(destPath)

	f.Url = template.URL("/" + destPath)
	// tmpl.Render(w, "entry-details.html", f)
	i.FileData = f
	i.Title = "Image Preview"

	tmpl.Render(w, "preview-media.html", i)

}

func PreviewMedia(w http.ResponseWriter, r *http.Request) {

	var f models.FileData
	var i IndexData

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := f.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	f.SimplifyDetails()

	base := "media"

	srcPath := fmt.Sprintf("%v/%v.%v", f.Directory, f.Name, f.Ext)
	destPath := fmt.Sprintf("%s/%s/%s.%v", base, f.Type, f.Name, f.Ext)

	fmt.Println(srcPath)
	fmt.Println(destPath)

	// Ensure the destination directory exists
	if err := os.MkdirAll(base+"/"+f.Type, os.ModePerm); err != nil {
		i.Title = "Error page"
		i.Body = map[string]string{"err": err.Error(), "msg": "Error in media/video directory creation"}
		tmpl.Render(w, "error.html", i)
		return
	}
	// Copy the image file
	if err := CopyImageFile(srcPath, destPath); err != nil {
		i.Title = "Error page"
		i.Body = map[string]string{"err": err.Error(), "msg": "Error copying image file"}
		tmpl.Render(w, "error.html", i)
		return
	}

	go func(filePath string) {
		time.Sleep(120 * time.Second)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error deleting file %s: %v", filePath, err)
		} else {
			log.Printf("File %s deleted successfully", filePath)
		}
	}(destPath)

	f.Url = template.URL("/" + destPath)
	// tmpl.Render(w, "entry-details.html", f)
	i.FileData = f
	i.Title = "Media Preview"

	tmpl.Render(w, "preview-media.html", i)

}

func CopyImageFile(srcPath, destPath string) error {
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

func PreviewById(w http.ResponseWriter, r *http.Request) {

	body, err := getItemById(w, r)
	if err != nil {
		http.Error(w, "Error retrieving file details (ItemDetailsId / getItemById)", http.StatusInternalServerError)
		return
	}

	if body.FileData.Type == "txt" {

		address := fmt.Sprintf("%s\\%s.%s", body.FileData.Directory, body.FileData.Name, body.FileData.Ext)
		body.Body["address"] = address
		html, err := utils.TxtToChoroma(address)
		if err != nil {
			http.Error(w, "Error converting text to choroma", http.StatusInternalServerError)
			return
		}

		_ = fmt.Sprintf("<div><h3>Text to chroma</h3><p>%s</p></div>", html)
		body.Html = html

		tmpl.Render(w, "chroma-preview.html", body)
		return
	}

	tmpl.Render(w, "detail.html", body)

}

func TxtToChoroma(f models.FinfoJSON) (template.HTML, error) {

	address := fmt.Sprintf("%s\\%s.%v", f.Directory, f.Name, f.Ext)
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
