package application

import (
	"context"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"time"

	"gofiles/chroma"
	"gofiles/internal/filter"
	"gofiles/internal/models"
	"gofiles/utils"
)

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
	HTML         template.HTML
	IsText       bool
}

func (s SimpleReq) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	s.FillData(w, r)
}

func (app *Application) HandleSearch(w http.ResponseWriter, r *http.Request) {
	type IndexPage struct {
		models.FilesDataList
		FileData models.FileData
		Title    string
		Count    int
		HTML     template.HTML
		QP       filter.QueryParams
		Message  string
		Result   []models.FileData
	}

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

	model := models.FileModel{DB: app.DB}

	list, count, err := model.GetList(qp.Keywords, qp.Limit, qp.Offset)
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

	app.Logger.Debug("LIST", "LIST", list)
	app.Logger.Debug("count", "Count", count)

	for index := range list {
		list[index].SimplifyDetails()
	}

	filesDataList := models.FilesDataList{
		Count: count,
		List:  list,
	}

	indexPage := IndexPage{
		Title:         fmt.Sprintf("wyszukano %d plików", count),
		FilesDataList: filesDataList,
		Count:         count,
		QP:            filter,
		Result:        list,
	}

	app.Render(w, templatePage, indexPage)
}

func (app *Application) ItemDetailsID(w http.ResponseWriter, r *http.Request) {
	var f models.FileData

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := f.GetByID(id); err != nil {
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	f.SimplifyDetails()

	tmpl.Render(w, "entry-details.html", f)
}

func DetailsID(w http.ResponseWriter, r *http.Request) {
	body, err := getItem(r)
	if err != nil {
		i := IndexData{Title: "Error page", Body: map[string]string{"err": err.Error(), "msg": "ItemDetailsId Error"}}
		tmpl.Render(w, "error.html", i)
		return
	}
	tmpl.Render(w, "simple-entry-details.html", body)
}

func getItem(r *http.Request) (models.FileData, error) {
	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	f := models.FileData{}

	if err != nil {
		return f, err
	}

	if err := f.GetByID(id); err != nil {
		return f, err
	}

	f.SimplifyDetails()

	return f, nil
}

func getItemByID(w http.ResponseWriter, r *http.Request) (IndexData, error) {
	f, err := getItem(r)
	if err != nil {
		http.Error(w, "Error retrieving file details in getItemById", http.StatusInternalServerError)
		return IndexData{}, err
	}

	body := IndexData{
		Title:    "Details",
		FileData: f,
		Body:     map[string]string{"message": "Detail Page", "id": fmt.Sprintf("%v", f.ID)},
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

func (app *Application) HandleAppend(w http.ResponseWriter, r *http.Request) {
	type IndexPage struct {
		models.FilesDataList
		FileData models.FileData
		Title    string
		Count    int
		HTML     template.HTML
		QP       filter.QueryParams
		Message  string
		Result   []models.FileData
	}

	filter := filter.Parse(r)

	data := models.FilesDataList{}

	qp := models.QueryParams{
		Keywords:  filter.Keywords,
		Limit:     filter.Limit,
		Offset:    filter.Offset,
		Order:     filter.Order,
		Ascending: filter.Ascending,
		Method:    filter.Method,
	}

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

	tmpl.Render(
		w, "append.html", IndexPage{
			Title:         "My Title",
			FilesDataList: data,
			Count:         data.Count,
			QP:            filter,
			Result:        data.List,
		},
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

	if err := f.GetByID(id); err != nil {
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
	if err := copyImageFile(srcPath, destPath); err != nil {
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

	f.URL = template.URL("/" + destPath)
	// tmpl.Render(w, "entry-details.html", f)
	i.FileData = f
	i.Title = "Image Preview"

	tmpl.Render(w, "preview-media.html", i)
}

func (app *Application) PreviewMedia(w http.ResponseWriter, r *http.Request) {
	var f models.FileData
	var i IndexData

	idStr := r.PathValue("id")

	id, err := strconv.Atoi(idStr)
	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	if err := f.GetByID(id); err != nil {
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
	if err := copyImageFile(srcPath, destPath); err != nil {
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

	f.URL = template.URL("/" + destPath)
	// tmpl.Render(w, "entry-details.html", f)
	i.FileData = f
	i.Title = "Media Preview"

	tmpl.Render(w, "preview-media.html", i)
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

func (app *Application) PreviewByID(w http.ResponseWriter, r *http.Request) {
	body, err := getItemByID(w, r)
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
		body.HTML = html

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
