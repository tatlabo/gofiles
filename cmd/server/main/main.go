package main

import (
	"gofiles/internal/handlers"
	"io"
	"log"
	"net/http"
	_ "net/http/pprof"
)

// type Template struct {
// 	templates *template.Template
// }

// func (t *Template) Render(w io.Writer, name string, data any) error {
// 	return t.templates.ExecuteTemplate(w, name, data)
// }

func WrapAuth(fn http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authentication logic can be added here
		username, password, ok := r.BasicAuth()

		if !ok || username != "admin" || password != "s3cr3t" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		fn(w, r)
		// Pre-processing logic can be added here
		log.Println("Handling request:", r.URL.Path)
	})
}

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
	http.HandleFunc("/preview-media/{id}", handlers.PreviewMedia)
	//protected routes
	protected := http.NewServeMux()
	http.Handle("/admin/", http.StripPrefix("/admin", WrapAuth(protected.ServeHTTP)))
	//
	protected.HandleFunc("/dirs", handlers.HandleDirs)
	protected.HandleFunc("/dirs/delete", handlers.HandleDirDelete)
	protected.HandleFunc("/dirs/scan", handlers.HandleScan)

	certPath := "C:/Users/adam/gofiles/localhost.pem"
	keyPath := "C:/Users/adam/gofiles/localhost-key.pem"

	err := http.ListenAndServeTLS(":443", certPath, keyPath, nil)
	log.Fatal(err)

	//httpToHTTPS redirects all http to https
	redirectToTls := func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "https://localhost:443"+r.RequestURI, http.StatusMovedPermanently)
	}

	go func() {
		if err := http.ListenAndServe(":80", http.HandlerFunc(redirectToTls)); err != nil {
			log.Fatalf("ListenAndServe error: %v", err)
		}
	}()

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
