package main

import (
	"crypto/tls"
	_ "embed"
	cert "gofiles/certs"
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

	cert, err := cert.LocalCert()
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	server := &http.Server{
		Addr:    ":443",
		Handler: nil, // Use default mux
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	err = server.ListenAndServeTLS("", "")
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

}
