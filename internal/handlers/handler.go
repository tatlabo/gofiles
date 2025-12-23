package handlers

import (
	"context"
	"fmt"
	"gofiles/chroma"
	"gofiles/internal/models"
	"gofiles/scan"
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

func HandleSearch(w http.ResponseWriter, r *http.Request) {

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
					result.List[index].CheckExtension()
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

	f.CheckExtension()
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
	f.CheckExtension()

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

func HandleAppend(w http.ResponseWriter, r *http.Request) {

	query := r.URL.Query()

	fmt.Println(query)

	keywords := query.Get("keywords")
	lStr := query.Get("limit")
	limit, _ := strconv.Atoi(lStr)
	offsetStr := query.Get("offset")
	offset, _ := strconv.Atoi(offsetStr)
	order := query.Get("order")
	ascending := query.Get("ascending")

	data := models.FilesDataList{}
	if err := data.AppendList(keywords, limit, offset, order, ascending); err != nil {
		fmt.Println("Error getting search list: ", err)
		tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
		return
	}

	tmpl.Render(w, "append.html", IndexData{Title: "My Title", Body: map[string]string{"message": "data", "keywords": keywords}, FilesDataList: data,
		Count: data.Count, SearchParams: map[string]string{"keywords": keywords, "limit": lStr, "offset": offsetStr},
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

	f.CheckExtension()
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

	f.Url = template.URL("/" + destPath)
	// tmpl.Render(w, "entry-details.html", f)
	i.FileData = f
	i.Title = "Image Preview"

	tmpl.Render(w, "preview-image.html", i)

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

func HandleDirs(w http.ResponseWriter, r *http.Request) {

	templatePage := "dirs.html"

	switch r.Method {
	case http.MethodGet:
		{
			// Get query parameters from URL
			successMsg := r.URL.Query().Get("success")
			addedPath := r.URL.Query().Get("added_path")

			message := "Zindexuj katalogi"
			if successMsg == "true" && addedPath != "" {
				message = fmt.Sprintf("Successfully added path: %s", addedPath)
			}

			data := models.Directries{
				Title: "Directory listing",
				Body:  map[string]string{"message": message},
			}

			if err := data.List(); err != nil {
				fmt.Println("Error getting search list: ", err)
				tmpl.Render(w, templatePage, data)
				return
			}

			tmpl.Render(w, templatePage, data)
		}

	case http.MethodPost:
		{
			r.ParseForm()
			path := r.FormValue("path")

			//todo: validate path
			// todo: check if path exists

			data := models.Directries{
				Title: path,
				Body:  map[string]string{"message": "Indexuj katalogi " + path},
			}

			err := data.AddPath(path)
			if err != nil {
				fmt.Println("Error adding path: ", err)
				data.Body = map[string]string{"message": "Error adding path: ", "error": err.Error()}

				if err := data.List(); err != nil {
					fmt.Println("Error getting search list: ", err)
				}
				tmpl.Render(w, templatePage, data)
				return
			}

			// Redirect to GET with query parameters
			redirectURL := fmt.Sprintf("%s?success=true&added_path=%s", r.URL.Path, path)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)

		}
	}
}

func HandleDirDelete(w http.ResponseWriter, r *http.Request) {

	templatePage := "dirs.html"

	switch r.Method {
	case http.MethodGet:
		{
			// Get query parameters from URL
			redirectURL := "/dirs"
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)

		}

	case http.MethodPost:
		{
			r.ParseForm()
			id := r.FormValue("id")

			uuid, err := uuid.Parse(id)
			if err != nil {
				fmt.Println("Invalid ID: ", err)
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}

			//todo: validate path
			// todo: check if path exists

			data := models.Directries{
				Title: id,
				Body:  map[string]string{"message": "Indexuj katalogi " + id},
			}

			d, err := data.DeletePath(uuid)
			if err != nil {
				fmt.Println("Error adding path: ", err)
				data.Body = map[string]string{"message": "Error adding path: ", "error": err.Error()}

				if err := data.List(); err != nil {
					fmt.Println("Error getting search list: ", err)
				}
				tmpl.Render(w, templatePage, data)
				return
			}

			redirectURL := fmt.Sprintf("/dirs?success=true&deleted_id=%s&path=%s", d.Id, d.Path)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
	}
}

func HandleScan(w http.ResponseWriter, r *http.Request) {

	templatePage := "dirs.html"

	switch r.Method {
	case http.MethodGet:
		{
			// Get query parameters from URL
			redirectURL := "/dirs"
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)

		}

	case http.MethodPost:
		{
			r.ParseForm()
			id := r.FormValue("scanid")

			uuid, err := uuid.Parse(id)
			if err != nil {
				fmt.Println("Invalid ID: ", err)
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}

			//todo: validate id
			// todo: check if path exists

			data := models.Directries{
				Title: id,
				Body:  map[string]string{"message": "Indexuj katalogi " + id},
			}

			d := models.Directory{}
			d.Id = uuid

			err = d.Row(uuid)
			if err != nil {
				fmt.Println("Error adding path: ", err)
				data.Body = map[string]string{"message": "Error adding path: ", "error": err.Error()}

				if err := data.List(); err != nil {
					fmt.Println("Error getting search list: ", err)
				}
				tmpl.Render(w, templatePage, data)
				return
			}

			go func(d models.Directory) {
				err := scan.Scan(d)
				if err != nil {
					log.Printf("Error scanning directory %s: %v", d.Path, err)
				}
			}(d)

			redirectURL := fmt.Sprintf("/dirs?scanning=true&path=%s&id=%s", d.Path, d.Id)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
	}
}

type Data struct {
	StartTime string
	EndTime   string
	Duration  string
	Text      string
}

type DataHtml struct {
	Msg    string
	Data   Data
	Status bool
}

func HandleCtx(w http.ResponseWriter, r *http.Request) {

	delayStr := r.URL.Query().Get("delay")
	delay, err := strconv.Atoi(delayStr)
	if err != nil {
		delay = 1000 // default delay in milliseconds
	}

	ctx := context.Background()
	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	// r = r.WithContext(ctx)
	d := make(chan DataHtml, 1)

	go func() {
		defer close(d)
		d <- dataR(delay)
	}()

	select {
	case <-ctx.Done():
		var c DataHtml
		c.Msg = "Request timed out"
		c.Status = false

		tmpl.Render(w, "teplate.html", c)
		return
	case res := <-d:
		tmpl.Render(w, "teplate.html", res)
	}

}

func dataR(t int) DataHtml {

	t1 := time.Now()
	time.Sleep(time.Duration(t) * time.Millisecond)
	t2 := time.Now()

	var data DataHtml
	data.Data = Data{
		StartTime: t1.Format("2006-01-02 15:04:05"),
		EndTime:   t2.Format("2006-01-02 15:04:05"),
		Duration:  t2.Sub(t1).String(),
		Text:      "some data",
	}
	data.Status = true
	data.Msg = "Processing completed successfully"
	return data
}
