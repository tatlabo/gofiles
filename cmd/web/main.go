package main

import (
	"context"
	_ "embed"
	"log"
	"log/slog"
	"net/http"
	_ "net/http/pprof"
	"os"

	"gofiles/internal/application"
	"gofiles/sqldb"
)

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, &slog.HandlerOptions{
		Level: slog.LevelDebug,
	}))

	pool, err := sqldb.PgxPool()
	if err != nil {
		log.Fatal(err)
	}
	defer pool.Close()

	if err := pool.Ping(context.Background()); err != nil {
		log.Fatal(err)
	}

	app := application.Run(pool, logger)

	go func() {
		redirectToTLS := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, "https://localhost:443"+r.RequestURI, http.StatusMovedPermanently)
		})
		logger.Info("Starting HTTP redirect server on :80")
		err := http.ListenAndServe(":80", redirectToTLS)
		if err != nil {
			logger.Error("ListenAndServe", "error", err)
		}
	}()

	router := NewRouter(app, logger)
	tlsServer := NewServer(router)
	logger.Info("Starting HTTPS server on :443")

	err = tlsServer.ListenAndServeTLS("", "")
	if err != nil {
		logger.Error("ListenAndServeTLS", "error", err)
	}
}

func StaticCache(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Cache static assets aggressively; deploy new file names when content changes.
		w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
		next.ServeHTTP(w, r)
	})
}

func LogFunc(next http.Handler, logger *slog.Logger) http.Handler {
	fn := func(w http.ResponseWriter, r *http.Request) {
		defer println()
		logger.Info(
			"path", r.URL.Path,
			"request",
			"method", r.Method,
		)
		next.ServeHTTP(w, r)
	}

	return http.HandlerFunc(fn)
}
