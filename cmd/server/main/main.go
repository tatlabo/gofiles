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

type Middleware func(http.HandlerFunc) http.HandlerFunc
type User struct {
	Username string
	Password string
	Ok       bool
}

func (m Middleware) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	m(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	}))(w, r)

}

func Login(fn http.HandlerFunc) Middleware {
	return func(next http.HandlerFunc) http.HandlerFunc {
		return func(w http.ResponseWriter, r *http.Request) {

			var user User
			user.Username, user.Password, user.Ok = r.BasicAuth()

			if !user.Ok || user.Username != "admin" || user.Password != "secret" {
				w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
				http.Error(w, "Unauthorized.", http.StatusUnauthorized)
				return
			}

			next(w, r)
		}
	}
}

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

	http.Handle("/h2", Login(h2))
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

	helloHandler := func(w http.ResponseWriter, req *http.Request) {
		io.WriteString(w, "Hello, world!\n")
	}

	http.HandleFunc("/hello", helloHandler)

	// HTTP server that redirects to HTTPS
	go func() {
		redirectToTls := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://localhost:443"+r.RequestURI, http.StatusMovedPermanently)
		})
		log.Println("Starting HTTP redirect server on :80")
		if err := http.ListenAndServe(":80", redirectToTls); err != nil {
			log.Fatalf("HTTP server error: %v", err)
		}
	}()

	// HTTPS server on main goroutine
	tlsServer := &http.Server{
		Addr:    ":443",
		Handler: nil, // Use default mux
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}
	log.Println("Starting HTTPS server on :443")
	log.Fatal(tlsServer.ListenAndServeTLS("", ""))

}
