package handlers

import (
	"encoding/json"
	"fmt"
	"gofiles/internal/models"
	"gofiles/internal/utils"
	"html/template"
	"io"
	"net/http"

	"github.com/google/uuid"
)

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

var tmpl = Template{}

type IndexData struct {
	Title string
	Body  []string
	FilesDataList
	Count int
	models.FileData
}

func HandleJSON(w http.ResponseWriter, r *http.Request) {

	idxData := IndexData{}
	tmpl.templates = template.Must(template.New("").ParseGlob("public/views/*.html"))

	switch r.Method {
	case http.MethodGet:
		{
			idxData.Title = "My Title"
			idxData.Body = []string{"This is the body", "Second Line"}
			tmpl.Render(w, "index.html", idxData)
		}
	case http.MethodPost:
		{
			r.ParseForm()
			keywords := r.FormValue("name")

			data := FilesDataList{}
			if err := data.GetList(keywords); err != nil {
				fmt.Println("Error getting search list: ", err)
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"Error retrieving search results"}})
				return
			}

			if err := data.GetCount(keywords); err != nil {
				fmt.Println("Error getting search count: ", err)
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"Error retrieving search results"}})
				return
			}

			if keywords != "" {
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"data"}, FilesDataList: data})
			} else {
				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"TNie podano słów kluczowych"}})
			}
		}
	}
}

type FilesDataList struct {
	List  []models.FileData
	Count int
}

func (flist *FilesDataList) GetList(name string) error {

	stmt := fmt.Sprintf(`
	SELECT id, data, ts_rank_cd( to_tsvector(%[1]s, keywords), websearch_to_tsquery(%[1]s, $1) ) as ts_rank
	FROM files
	WHERE websearch_to_tsquery(%[1]s, $1) @@ to_tsvector(%[1]s, keywords)
	ORDER BY ts_rank DESC
	LIMIT $2 OFFSET $3;`, "'polish'")

	fmt.Printf("%s %s %d %d \n", stmt, name, 10, 0)

	conn, err := utils.PgConn()
	if err != nil {
		return err
	}
	defer conn.Close()

	rows, err := conn.Query(stmt, name, 10, 0)
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

func JsonDetailById(w http.ResponseWriter, r *http.Request) {

	// io.WriteString(w, "Hello from a HandleFunc #2!\n")

	// tmpl.templates = template.Must(template.New("").ParseGlob("public/views/*.html"))

	idStr := r.URL.Query().Get("id")
	// id, err := uuid.Parse(idStr)

	// if err != nil {
	// 	http.Error(w, "Invalid ID", http.StatusBadRequest)
	// 	return
	// }

	// data := models.FileData{}

	// if err := data.GetById(id); err != nil {
	// 	http.Error(w, "Error retrieving file details", http.StatusInternalServerError)
	// 	return
	// }

	tmpl.Render(w, "detail.html", IndexData{Title: "My Title", Body: []string{"TNie podano słów kluczowych", "istr: " + idStr}})
	// tmpl.Render(w, "detail.html", IndexData{Title: "Details", Body: []string{"TNie podano słów kluczowych", fmt.Sprintf("%v", id)}})

}
