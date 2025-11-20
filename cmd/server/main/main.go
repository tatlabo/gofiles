package main

import (
	"gofiles/internal/handlers"
	"html/template"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
)

var limit int

type Template struct {
	templates *template.Template
}

func (t *Template) Render(w io.Writer, name string, data any) error {
	return t.templates.ExecuteTemplate(w, name, data)
}

func main() {

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}

	type IndexData struct {
		Title string
		Body  []string
	}

	// handleJSON := func(w http.ResponseWriter, r *http.Request) {
	// 	switch r.Method {
	// 	case http.MethodGet:
	// 		{
	// 			tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"This is the body", "Second Line"}})
	// 		}
	// 	case http.MethodPost:
	// 		{
	// 			r.ParseForm()
	// 			name := r.FormValue("name")

	// 			if name != "" {
	// 				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"You searched for: " + name}})
	// 			} else {
	// 				tmpl.Render(w, "index.html", IndexData{Title: "My Title", Body: []string{"This is the body", "Second Line"}})
	// 			}
	// 		}
	// 	}

	// }

	http.HandleFunc("/endpoint", h2)
	http.HandleFunc("/search", handlers.HandleJSON)

	log.Fatal(http.ListenAndServe(":80", nil))

	// certPath := "C:/Users/adam/gofiles/certs/localhost+2.pem"
	// keyPath := "C:/Users/adam/gofiles/certs/localhost+2-key.pem"

	// Initialize Echo framework
	// e := echo.New()

	// e.Use(middleware.CORS())
	// e.Use(middleware.Recover())

	// e.Static("/static", "static")
	// e.Static("/media", "media")

	// e.Renderer = &Template{
	// 	templates: template.Must(template.New("").Funcs(template.FuncMap{
	// 		"formatDate": utils.FormatDate,
	// 		"not":        utils.Not,
	// 		"equals":     utils.Equals,
	// 		"notequals":  utils.Notequals,
	// 	}).ParseGlob("public/views/*.html")),
	// }

	// e.GET("/", handlers.StartPage)
	// e.GET("/search", handlers.SearchInDb)
	// e.POST("/search", handlers.SearchInDb)
	// e.GET("/detail/:id", handlers.DetailById)
	// e.GET("/details/:id", handlers.DetailById)
	// e.GET("/preview/:id", handlers.PreviewById)
	// e.GET("/preview/image/:id", handlers.PreviewImage)
	// e.GET("/json/search", handlers.ResponseJson)
	// e.GET("/append", handlers.ResponseAppend)

	// log.Println("Starting HTTPS server on https://localhost:8443")
	// if err := e.StartTLS(":8443", certPath, keyPath); err != nil {
	// 	e.Logger.Fatal(err)
	// }
}
