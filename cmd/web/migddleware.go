package main

import "net/http"

// TO DO move user to .env

func Authenticate(fn http.HandlerFunc) http.HandlerFunc {
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
