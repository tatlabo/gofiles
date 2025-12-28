package handlers

import (
	"fmt"
	"gofiles/internal/models"
	"gofiles/scan"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func HandleScan(w http.ResponseWriter, r *http.Request) {

	templatePage := "dirs.html"

	switch r.Method {
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

			d := models.Directory{Id: uuid}

			err = d.Row(uuid)
			if err != nil {
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

			redirectURL := fmt.Sprintf("/admin/dirs?scanning=true&path=%s&id=%s", d.Path, d.Id)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
	}
}
