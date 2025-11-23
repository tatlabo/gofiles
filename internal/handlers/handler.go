package handlers

import (
	"encoding/json"
	"fmt"
	"gofiles/internal/models"
	"gofiles/internal/utils"
	"html/template"
	"io"
	"net/http"
	"strconv"

	"github.com/google/uuid"
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

	stmt := fmt.Sprintf(`
	SELECT id, data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC
	LIMIT $2 OFFSET $3;`, "'polish'")

	fmt.Printf("%s %s %d %d \n", stmt, name, limit, offset)

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(stmt, name, limit, offset)
	if err != nil {
		return err
	}

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

		if err != nil {
			return err
		}

		flist.List = append(flist.List, dataWithId)
	}

	return nil
}

func (flist *FilesDataList) GetCount(name string) error {

	stmt := fmt.Sprintf(`
	SELECT COUNT(*) FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords);`, "'polish'")

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	err = conn.QueryRow(stmt, name).Scan(&flist.Count)

	if err != nil {
		return err
	}

	return nil
}

func Details(w http.ResponseWriter, r *http.Request) {

	tmpl.Render(w, "detail.html", IndexData{Title: "Details", Body: map[string]string{"message": "DetailPage"}})

}

func DetailsId(w http.ResponseWriter, r *http.Request) {

	// io.WriteString(w, "Hello from a HandleFunc #2!\n")

	idStr := r.PathValue("id")
	id, err := uuid.Parse(idStr)

	if err != nil {
		http.Error(w, "Invalid ID", http.StatusBadRequest)
		return
	}

	fileData := models.FileData{}

	if err := fileData.GetById(id); err != nil {
		http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
		return
	}

	body := IndexData{
		Title:    "Details",
		FileData: fileData,
		Body:     map[string]string{"message": "Detail Page", "id": fmt.Sprintf("%v", id)},
	}

	tmpl.Render(w, "detail.html", body)

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

func (flist *FilesDataList) GetListOffset(name string, limit int, offset int) error {

	stmt := fmt.Sprintf(`
	SELECT id, data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC
	LIMIT $2 OFFSET $3;`, "'polish'")

	fmt.Printf("%s %s %d %d \n", stmt, name, limit, offset)

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(stmt, name, limit, offset)
	if err != nil {
		return err
	}

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

		if err != nil {
			return err
		}

		flist.List = append(flist.List, dataWithId)
	}

	return nil
}
