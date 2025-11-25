package handlers

import (
	"encoding/json"
	"fmt"
	"gofiles/chroma"
	"gofiles/internal/models"
	"gofiles/internal/utils"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
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
	FilesDataList
	FileData     models.FileData
	Title        string
	Body         map[string]string
	Count        int
	SearchParams map[string]string
	HomePage     bool
	Html         template.HTML
}

func HandleSearch(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		{
			tmpl.Render(w, "home.html", IndexData{
				Title:    "Search files",
				Body:     map[string]string{"message": "Wyszukaj pliki po słowach kluczowych"},
				HomePage: true,
			})
		}

	case http.MethodPost:
		{
			r.ParseForm()
			keywords := r.FormValue("name")

			limit := 10
			offset := 0

			data := FilesDataList{}

			if err := data.GetCount(keywords); err != nil {
				fmt.Println("Error getting search count: ", err)
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
				return
			}

			if data.Count == 0 {
				tmpl.Render(w, "home.html",
					IndexData{Title: "No results",
						Body:         map[string]string{"message": "No results found for the given keywords"},
						SearchParams: map[string]string{"keywords": keywords}})
				return
			}

			if err := data.GetList(keywords, limit, offset); err != nil {
				fmt.Println("Error getting search list: ", err)
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
				return
			}

			if keywords != "" {
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "data", "keywords": keywords}, FilesDataList: data,
					Count: data.Count, SearchParams: map[string]string{"keywords": keywords, "limit": "10", "offset": "0"},
				})
			} else {
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "TNie podano słów kluczowych"}})
			}
		}
	}
}

type FilesDataList struct {
	List  []models.FileData
	Count int
}

func (flist *FilesDataList) GetList(name string, limit int, offset int) error {

	language := "'polish'"
	query := `
	SELECT DISTINCT(id), data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC, id ASC
	LIMIT $2 OFFSET $3;`

	stmt := fmt.Sprintf(query, language)

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(stmt, name, limit, offset)
	if err != nil {
		return err
	}

	go explain(stmt, name, limit, offset)

	for rows.Next() {

		tsRank := float64(0)
		id := uuid.UUID{}
		data := models.FinfoJSON{}

		rawData := []byte{}

		err := rows.Scan(
			&id,
			&rawData,
			&tsRank,
		)
		if err != nil {
			return err
		}

		err = json.Unmarshal(rawData, &data)

		dataWithId := models.FileData{
			FinfoJSON: data,
			Id:        id,
		}

		dataWithId.Keywords = name
		dataWithId.SimplifyDetails()

		if err != nil {
			return err
		}

		flist.List = append(flist.List, dataWithId)
	}

	return nil
}

func (flist *FilesDataList) GetCount(name string) error {

	stmt := fmt.Sprintf(`
	SELECT COUNT(DISTINCT id) FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, "'polish'")

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	go explain(stmt, name)

	err = conn.QueryRow(stmt, name).Scan(&flist.Count)

	if err != nil {
		return err
	}

	return nil
}

func Details(w http.ResponseWriter, r *http.Request) {
	tmpl.Render(w, "detail.html", IndexData{Title: "Details", Body: map[string]string{"message": "DetailPage"}})
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

	data := FilesDataList{}
	if err := data.GetList(keywords, limit, offset); err != nil {
		fmt.Println("Error getting search list: ", err)
		tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
		return
	}

	tmpl.Render(w, "append.html", IndexData{Title: "My Title", Body: map[string]string{"message": "data", "keywords": keywords}, FilesDataList: data,
		Count: data.Count, SearchParams: map[string]string{"keywords": keywords, "limit": lStr, "offset": offsetStr},
	})

}

func explain(stmt string, placeholders ...any) {
	conn, err := utils.PgConn()
	if err != nil {
		log.Println("Error connecting to the database for search words:", err)
		return
	}
	defer conn.Close()
	explainAnalyze := fmt.Sprintf("EXPLAIN ANALYZE %s", stmt)
	log.Printf("%v %v", explainAnalyze, placeholders)
	explainRows, err := conn.Query(explainAnalyze, placeholders...)
	if err != nil {
		log.Println("Error running EXPLAIN ANALYZE:", err)
	}
	defer explainRows.Close()
	for explainRows.Next() {
		var line string
		if err := explainRows.Scan(&line); err != nil {
			log.Println("Error scanning EXPLAIN ANALYZE line:", err)
			continue
		}
		log.Println(line)
	}
	if err := explainRows.Err(); err != nil {
		log.Println("Error iterating EXPLAIN ANALYZE rows:", err)
	}
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

	address := fmt.Sprintf("%s\\%s.%s", body.FileData.Directory, body.FileData.Name, body.FileData.Ext)
	body.Body["address"] = address
	html, err := txtToChoroma(address)
	if err != nil {
		http.Error(w, "Error converting text to choroma", http.StatusInternalServerError)
		return
	}

	// wrap := fmt.Sprintf("<div><h3>Text to chroma</h3><p>%s</p></div>", html)
	body.Html = html

	tmpl.Render(w, "chroma-preview.html", body)

	// tmpl.Render(w, "chromoa-preview.html", IndexData{Title: "Chroma preview", Body: map[string]string{"message": "data", "html": wrap}, FileData: f})
}

func txtToChoroma(address string) (template.HTML, error) {

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
