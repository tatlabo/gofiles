package main

import (
	"gofiles/internal/handlers"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
)

var limit int

// type Template struct {
// 	templates *template.Template
// }

// func (t *Template) Render(w io.Writer, name string, data any) error {
// 	return t.templates.ExecuteTemplate(w, name, data)
// }

func main() {

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	media := http.FileServer(http.Dir("media"))
	http.Handle("/media/", http.StripPrefix("/media/", media))

	h2 := func(w http.ResponseWriter, _ *http.Request) {
		io.WriteString(w, "Hello from a HandleFunc #2!\n")
	}

	type IndexData struct {
		Title string
		Body  []string
	}

	http.HandleFunc("/h2", h2)
	http.HandleFunc("/h3", handlers.HandleCtx)
	http.HandleFunc("/", handlers.HandleSearch)
	http.HandleFunc("/append", handlers.HandleAppend)
	http.HandleFunc("/detail/{id}", handlers.ItemDetailsId)
	http.HandleFunc("/item-detail/{id}", handlers.ItemDetailsId)
	http.HandleFunc("/preview/{id}", handlers.PreviewById)
	//protected routes
	http.HandleFunc("/dirs", handlers.HandleDirs)
	http.HandleFunc("/dirs/delete", handlers.HandleDirDelete)
	http.HandleFunc("/dirs/scan", handlers.HandleScan)
	http.HandleFunc("/preview-image/{id}", handlers.PreviewImage)

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
	// e.GET("/preview/image/:id", handlers.PreviewImage)
	// e.GET("/json/search", handlers.ResponseJson)
	// e.GET("/append", handlers.ResponseAppend)

	// log.Println("Starting HTTPS server on https://localhost:8443")
	// if err := e.StartTLS(":8443", certPath, keyPath); err != nil {
	// 	e.Logger.Fatal(err)
	// }
}
