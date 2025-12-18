package handlers

import (
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
	"github.com/labstack/echo/v4"
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
					tmpl.Render(w, templatePage, IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
					return
				}
				count <- data.Count
			}()

			select {
			case err := <-errCh:
				fmt.Println("Error getting search count: ", err)
				return
			case c := <-count:
				if c == 0 {
					templatePage = "home.html"
					tmpl.Render(w, templatePage,
						IndexData{Title: "No results",
							Body:         map[string]string{"message": "No results found for the given keywords"},
							SearchParams: map[string]string{"keywords": keywords}})
					return

				}
			}

			res := make(chan models.FilesDataList)
			resErr := make(chan error)
			go func() {
				defer close(res)
				defer close(resErr)

				// defer wg.Done()
				if err := data.GetList(keywords, limit, offset); err != nil {
					resErr <- err
					tmpl.Render(w, templatePage, IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
					return
				}
				res <- data
			}()

			select {
			case err := <-errCh:
				fmt.Println("Error getting search count: ", err)
				return
			case err := <-resErr:
				fmt.Println("Error getting search list: ", err)
				return
			case result := <-res:
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

	body, err := getItemById(w, r)
	if err != nil {
		http.Error(w, "Error retrieving file details (ItemDetailsId / getItemById)", http.StatusInternalServerError)
		return
	}
	tmpl.Render(w, "simple-entry-details.html", body)

}

func DetailsId(w http.ResponseWriter, r *http.Request) {

	body, err := getItemById(w, r)
	if err != nil {
		http.Error(w, "Error retrieving file details (DetailsId / getItemById)", http.StatusInternalServerError)
		return
	}

	tmpl.Render(w, "detail.html", body)

}

func getItem(w http.ResponseWriter, r *http.Request) (models.FileData, error) {

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)
	f := models.FileData{}

	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return f, err
	}

	if err := f.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details in method GetById", http.StatusInternalServerError)
		return f, err
	}

	f.SimplifyDetails()
	if err := f.CheckExtension(); err != nil {
		http.Error(w, "Error checking file extension", http.StatusInternalServerError)
		return f, err
	}

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

	data := models.FilesDataList{}
	if err := data.GetList(keywords, limit, offset); err != nil {
		fmt.Println("Error getting search list: ", err)
		tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
		return
	}

	tmpl.Render(w, "append.html", IndexData{Title: "My Title", Body: map[string]string{"message": "data", "keywords": keywords}, FilesDataList: data,
		Count: data.Count, SearchParams: map[string]string{"keywords": keywords, "limit": lStr, "offset": offsetStr},
	})
	log.Printf("nr of gorutines (HandleAppend): %v", runtime.NumGoroutine())

}

func PreviewImageNew(c echo.Context) error {

	id := c.Param("id")

	finfo, err := SelectFinfoById(id)
	if err != nil {
		return c.String(http.StatusInternalServerError, "Error executing query by id in\n PreviewImage "+err.Error())
	}

	srcPath := fmt.Sprintf("%s/%s.%v", finfo.Directory, finfo.Name, finfo.Ext)
	destPath := fmt.Sprintf("media/images/%s.%v", finfo.Name, finfo.Ext)

	// Ensure the destination directory exists
	if err := os.MkdirAll("media/images", os.ModePerm); err != nil {
		return c.String(http.StatusInternalServerError, "Error creating destination directory")
	}

	// Copy the image file
	if err := copyImageFile(srcPath, destPath); err != nil {
		log.Printf("Error copying file from %s to %s: %v", srcPath, destPath, err)
		return c.String(http.StatusInternalServerError, "Error copying image file")
	}

	go func(filePath string) {
		time.Sleep(120 * time.Second)
		if err := os.Remove(filePath); err != nil {
			log.Printf("Error deleting file %s: %v", filePath, err)
		} else {
			log.Printf("File %s deleted successfully", filePath)
		}
	}(destPath)

	finfo.Src = template.URL(destPath)

	return c.Render(http.StatusOK, "single_page", finfo)
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

func TxtToChoroma(f models.Finfo) (template.HTML, error) {

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
