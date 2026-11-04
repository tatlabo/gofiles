// Package application - main handlers
package application

import (
	"fmt"
	"net/http"

	"gofiles/internal/models"

	"github.com/google/uuid"
)

type DirsPaage struct {
	Title string

	Body  map[string]string
	Array []models.Directory
	Err   error
	Flash string
}

func HandleDirs(w http.ResponseWriter, r *http.Request) {
	templatePage := "dirs.html"

	data := DirsPaage{
		Title: "strona z katalogami",
	}
	dirs := models.Directries{}

	switch r.Method {
	case http.MethodGet:
		{
			// Get query parameters from URL
			successMsg := r.URL.Query().Get("success")
			addedPath := r.URL.Query().Get("added_path")

			success := r.URL.Query().Get("success")
			deleted := r.URL.Query().Get("deleted")
			path := r.URL.Query().Get("path")

			if success == "true" && deleted != "" {
				data.Flash = fmt.Sprintf("Successfully deleted path: %s (ID: %s)", path, deleted)
			}

			if successMsg == "true" && addedPath != "" {
				data.Flash = fmt.Sprintf("Successfully added path: %s", addedPath)
			}

			err := dirs.List()
			if err != nil {
				fmt.Println("Error getting search list: ", err)
				tmpl.Render(w, templatePage, data)
				return
			}

			data.Array = dirs.Array

			tmpl.Render(w, templatePage, data)
		}
	case http.MethodPost:
		//
		// TO DO DI
		//
		{
			r.ParseForm()
			path := r.FormValue("path")

			// todo: validate path
			// todo: check if path exists

			data.Title = path

			data := models.Directries{
				Title: path,
				Body:  map[string]string{"message": "Indexuj katalogi " + path},
			}

			err := dirs.AddPath(path)
			if err != nil {
				fmt.Println("Error adding path: ", err)
				data.Body = map[string]string{"message": "Error adding path: ", "error": err.Error()}
				tmpl.Render(w, templatePage, data)
				return
			}

			err = dirs.List()
			if err != nil {
				tmpl.Render(w, templatePage, data)
				return
			}

			// Redirect to GET with query parameters
			redirectURL := fmt.Sprintf("/admin/%s?success=true&added_path=%s", r.URL.Path, path)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)

		}
	}
}

func HandleDirDelete(w http.ResponseWriter, r *http.Request) {
	templatePage := "dirs.html"

	switch r.Method {
	// case http.MethodGet:
	// 	{
	// 		params := r.URL.Query()

	// 		deleted := params.Get("deleted")
	// 		path := params.Get("path")
	// 		success := params.Get("success")

	// 		fmt.Println("Deleted ID:", deleted)
	// 		fmt.Println("Path:", path)
	// 		fmt.Println("Success:", success)

	// 		redirectURL := "/admin/dirs"
	// 		http.Redirect(w, r, redirectURL, http.StatusSeeOther)

	// 	}

	case http.MethodPost:
		{
			_ = r.ParseForm()
			id := r.FormValue("id")

			uuid, err := uuid.Parse(id)
			if err != nil {
				fmt.Println("Invalid ID: ", err)
				http.Error(w, "Invalid ID", http.StatusBadRequest)
				return
			}

			// todo: validate path
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

			redirectURL := fmt.Sprintf("/admin/dirs?success=true&deleted=%s&path=%s", d.ID, d.Path)
			http.Redirect(w, r, redirectURL, http.StatusSeeOther)
		}
	}
}
