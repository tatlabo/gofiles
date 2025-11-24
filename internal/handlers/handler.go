package handlers

import (
	"encoding/json"
	"fmt"
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
	templates: template.Must(template.New("").ParseGlob("public/views/*.html")),
}

type IndexData struct {
	Title string
	Body  map[string]string
	FilesDataList
	Count        int
	FileData     models.FileData
	SearchParams map[string]string
}

func HandleJSON(w http.ResponseWriter, r *http.Request) {

	switch r.Method {
	case http.MethodGet:
		{
			tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Wyszukaj pliki po słowach kluczowych"}})

			// query := r.URL.Query()

			// keywords := query.Get("keywords")
			// append := query.Get("append")
			// lStr := query.Get("limit")
			// limit, _ := strconv.Atoi(lStr)
			// offsetStr := query.Get("offset")
			// offset, _ := strconv.Atoi(offsetStr)

			// data := FilesDataList{}
			// if err := data.GetList(keywords, limit, offset); err != nil {
			// 	fmt.Println("Error getting search list: ", err)
			// 	tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"Error retrieving search results"}})
			// 	return
			// }

			// if err := data.GetCount(keywords); err != nil {
			// 	fmt.Println("Error getting search count: ", err)
			// 	tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"Error retrieving search results"}})
			// 	return
			// }

			// if append != "" {
			// 	// Return only the list part for appending
			// 	tmpl.Render(w, "append.html", IndexData{Title: "My Title", Body: []string{"data", keywords}, FilesDataList: data,
			// 		Count: data.Count, SearchParams: map[string]string{"keywords": keywords, "limit": lStr, "offset": offsetStr},
			// 	})
			// 	return
			// }

		}

	case http.MethodPost:
		{
			r.ParseForm()
			keywords := r.FormValue("name")

			limit := 10
			offset := 0

			data := FilesDataList{}
			if err := data.GetList(keywords, limit, offset); err != nil {
				fmt.Println("Error getting search list: ", err)
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: map[string]string{"message": "Error retrieving search results"}})
				return
			}

			if err := data.GetCount(keywords); err != nil {
				fmt.Println("Error getting search count: ", err)
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

func getItemById(w http.ResponseWriter, r *http.Request) (IndexData, error) {

	idStr := r.PathValue("id")

	id, err := uuid.Parse(idStr)

	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return IndexData{}, err
	}

	fileData := models.FileData{}

	if err := fileData.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details in method GetById", http.StatusInternalServerError)
		return IndexData{}, err
	}

	fileData.SimplifyDetails()

	body := IndexData{
		Title:    "Details",
		FileData: fileData,
		Body:     map[string]string{"message": "Detail Page", "id": fmt.Sprintf("%v", id)},
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
