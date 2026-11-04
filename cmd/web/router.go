package main

import (
	"io"
	"log/slog"
	"net/http"

	"gofiles/internal/application"
	"gofiles/static"
)

func Router(logger *slog.Logger) http.Handler {
	router := http.NewServeMux()

	http.HandleFunc("/hello", func(w http.ResponseWriter, r *http.Request) {
		msg := "hello server"
		_, _ = io.WriteString(w, msg)
	})

	router.Handle("GET /static/", StaticCache(http.StripPrefix("/static/", http.FileServerFS(static.StaticFiles))))
	media := http.FileServer(http.Dir("media"))
	router.Handle("/media/", http.StripPrefix("/media/", media))

	router.HandleFunc("/h3", application.HandleCtx)
	router.HandleFunc("/", application.HandleSearch)
	router.HandleFunc("/append", application.HandleAppend)
	router.HandleFunc("/detail/{id}", application.ItemDetailsID)
	router.HandleFunc("/item-detail/{id}", application.ItemDetailsID)
	router.HandleFunc("/preview/{id}", application.PreviewByID)
	router.HandleFunc("/preview-media/{id}", application.PreviewMedia)

	// protected routes as admin
	protected := http.NewServeMux()
	router.Handle("/admin/", http.StripPrefix("/admin", Authenticate(protected.ServeHTTP)))

	idata := application.SimpleReq{}
	router.Handle("/data/", idata) // pprof
	router.Handle("/data/{id}", idata)
	//
	protected.HandleFunc("/", application.HandleDirs)
	protected.HandleFunc("/dirs/delete", application.HandleDirDelete)
	protected.HandleFunc("/dirs/scan", application.HandleScan)

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
