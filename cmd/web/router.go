package main

import (
	"io"
	"log/slog"
	"net/http"

	"gofiles/internal/application"
	"gofiles/static"
)

func NewRouter(app *application.Application, logger *slog.Logger) http.Handler {
	router := http.NewServeMux()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		msg := "hello server"
		_, _ = io.WriteString(w, msg)
	})

	router.Handle("GET /static/", StaticCache(http.StripPrefix("/static/", http.FileServerFS(static.StaticFiles))))
	media := http.FileServer(http.Dir("media"))
	router.Handle("/media/", http.StripPrefix("/media/", media))

	router.HandleFunc("/h3", app.HandleCtx)
	router.HandleFunc("/", app.HandleSearch)
	router.HandleFunc("/append", app.HandleAppend)
	router.HandleFunc("/detail/{id}", app.ItemDetailsID)
	router.HandleFunc("/item-detail/{id}", app.ItemDetailsID)
	router.HandleFunc("/preview/{id}", app.PreviewByID)
	router.HandleFunc("/preview-media/{id}", app.PreviewMedia)

	// protected routes as admin
	protected := http.NewServeMux()
	router.Handle("/admin/", http.StripPrefix("/admin", Authenticate(protected.ServeHTTP)))

	idata := application.SimpleReq{}
	router.Handle("/data/", idata) // pprof
	router.Handle("/data/{id}", idata)
	//
	protected.HandleFunc("/", app.HandleDirs)
	protected.HandleFunc("/dirs/delete", app.HandleDirDelete)
	protected.HandleFunc("POST /dirs/scan", app.HandleScan)

	return PanicRecover(LogFunc(router, logger))
}

func PanicRecover(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer func() {
			rec := recover()
			if rec != nil {
				w.WriteHeader(500)
				w.Write([]byte("Panic..., Something wrong happend...."))
			}
		}()

		next.ServeHTTP(w, r)
	})
}
