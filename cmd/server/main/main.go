package main

import (
	"context"
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

func TraceId(fn http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Authentication logic can be added here
		ctx := r.Context()
		username, password, ok := r.BasicAuth()

		if traceID := r.Header.Get("X-Trace-ID"); traceID != "" {
			ctx = context.WithValue(ctx, "X-Trace-ID", traceID)
		}

		if !ok || username != "admin" || password != "s3cr3t" {
			w.Header().Set("WWW-Authenticate", `Basic realm="Restricted"`)
			http.Error(w, "Unauthorized.", http.StatusUnauthorized)
			return
		}

		fn.ServeHTTP(w, r.WithContext(ctx))
	})
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

type hello struct {
	Message string
}

func (h *hello) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	io.WriteString(w, h.Message)
}

func main() {

	// Serve static files (CSS, JS, images)
	fs := http.FileServer(http.Dir("static"))
	http.Handle("/static/", http.StripPrefix("/static/", fs))

	media := http.FileServer(http.Dir("media"))
	http.Handle("/media/", http.StripPrefix("/media/", media))

	h2 := &hello{Message: "Hello, secure world!"}

	type IndexData struct {
		Title string
		Body  []string
	}

	var idata http.Handler
	idata = handlers.SimpleReq{}

	http.Handle("/h2", TraceId(h2))
	http.HandleFunc("/h3", handlers.HandleCtx)
	http.HandleFunc("/", handlers.HandleSearch)
	http.HandleFunc("/append", handlers.HandleAppend)
	http.HandleFunc("/detail/{id}", handlers.ItemDetailsId)
	http.HandleFunc("/item-detail/{id}", handlers.ItemDetailsId)
	http.HandleFunc("/preview/{id}", handlers.PreviewById)
	http.HandleFunc("/preview-media/{id}", handlers.PreviewMedia)

	http.Handle("/data/", idata) // pprof
	http.Handle("/data/{id}", idata)
	//protected routes
	protected := http.NewServeMux()
	http.Handle("/admin/", http.StripPrefix("/admin", WrapAuth(protected.ServeHTTP)))
	//
	protected.HandleFunc("/dirs", handlers.HandleDirs)
	protected.HandleFunc("/dirs/delete", handlers.HandleDirDelete)
	protected.HandleFunc("/dirs/scan", handlers.HandleScan)

	certPEM, keyPEM, err := cert.LocalCert()
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	tlsCert, err := tls.X509KeyPair(certPEM, keyPEM)
	if err != nil {
		log.Fatalf("Failed to create TLS certificate: %v", err)
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
			Certificates: []tls.Certificate{tlsCert},
		},
	}
	log.Println("Starting HTTPS server on :443")
	log.Fatal(tlsServer.ListenAndServeTLS("", ""))

}
