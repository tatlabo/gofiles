package main

import (
	"log"
	"net/http"
	"time"
)

func main() {
	http.HandleFunc("/", handleCon)
	log.Println("Starting server on :8080")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("ListenAndServe:", err)
	}

}

func handleCon(w http.ResponseWriter, r *http.Request) {
	t := time.Now().Format("2006-01-02 15:04:05")
	w.Write([]byte(t))
	defer w.(http.Flusher).Flush()

}
