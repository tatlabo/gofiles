package handlers

import (
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

	"github.com/google/uuid"
)

type Template struct {
	templates *template.Template
}

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

type IndexData struct {
	models.FilesDataList
	FileData     models.FileData
	Title        string
	Body         map[string]string
	Count        int
	SearchParams map[string]string
	HomePage     bool
	Html         template.HTML
	IsText       bool
}

func WrapSearch(fn http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fn(w, r)
		// Pre-processing logic can be added here
		log.Println("Handling request:", r.URL.Path)
	})
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {
	WrapSearch(handleS)(w, r)
}

func handleS(w http.ResponseWriter, r *http.Request) {

	templatePage := "index.html"
	start := time.Now()
	var qp models.QueryParams

	switch r.Method {
	case http.MethodGet:

		qp = processQueryParams(r)

		// templatePage = "home.html"

		// tmpl.Render(w, templatePage, IndexData{
		// 	Title:    "Search files",
		// 	Body:     map[string]string{"message": "Wyszukaj pliki po słowach kluczowych"},
		// 	HomePage: true,
		// 	SearchParams: map[string]string{
		// 		"keywords":  qp.Keywords,
		// 		"limit":     strconv.Itoa(qp.Limit),
		// 		"offset":    strconv.Itoa(qp.Offset),
		// 		"order":     qp.Order,
		// 		"ascending": strconv.FormatBool(qp.Ascending),
		// 	},
		// })

	case http.MethodPost:
		qp = processQueryParams(r)
	}

	if qp.Keywords == "" {
		tmpl.Render(w, templatePage, IndexData{Title: "My Title", Body: map[string]string{"message": "TNie podano słów kluczowych"}})
		return
	}

	data := models.FilesDataList{}

	count := make(chan int, 1)
	errCh := make(chan error)
	go func() {
		defer close(count)
		// defer wg.Done()
		if err := data.SelectCount(qp.Keywords); err != nil {
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
					SearchParams: map[string]string{"keywords": qp.Keywords}})
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
		if err := data.GetList(qp.Keywords, qp.Limit, qp.Offset); err != nil {
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
			Body:          map[string]string{"message": "data", "keywords": qp.Keywords},
			FilesDataList: result,
			Count:         result.Count,
			SearchParams:  map[string]string{"keywords": qp.Keywords, "limit": strconv.Itoa(qp.Limit), "offset": strconv.Itoa(qp.Offset)},
		})

		log.Printf("\n\n\nSearch completed in %v\n\n", time.Since(start))
		log.Printf("nr of gorutines (after Render): %v", runtime.NumGoroutine())
		return
	}

}

func ItemDetailsId(w http.ResponseWriter, r *http.Request) {

	var f models.FileData

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
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

	id, err := uuid.Parse(idStr)
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
		query := r.URL.Query()

		lStr := query.Get("limit")
		offsetStr := query.Get("offset")
		ascending := query.Get("ascending")

		qp.Keywords = query.Get("keywords")
		qp.Limit, _ = strconv.Atoi(lStr)
		qp.Offset, _ = strconv.Atoi(offsetStr)
		qp.Order = query.Get("order")
		qp.Ascending = ascending == "true"
	case http.MethodPost:
		qp.Keywords = r.FormValue("keywords")
		qp.Limit = 10
		qp.Offset = 0
	}

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
		Body: map[string]string{
			"message":  "data",
			"keywords": qp.Keywords},
		FilesDataList: data,
		Count:         data.Count,
		SearchParams:  map[string]string{"keywords": qp.Keywords, "limit": strconv.Itoa(qp.Limit), "offset": strconv.Itoa(qp.Offset), "order": qp.Order, "ascending": strconv.FormatBool(qp.Ascending)},
	})
	log.Printf("nr of gorutines (HandleAppend): %v", runtime.NumGoroutine())

}

func PreviewImage(w http.ResponseWriter, r *http.Request) {

	var f models.FileData
	var i IndexData

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
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

	tmpl.Render(w, "preview-image.html", i)

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
