package handlers

import (
    "net/http"
)

type Handler struct{}

func (h *Handler) HandleGet(w http.ResponseWriter, r *http.Request) {
    // Handle GET request
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("GET request handled"))
}

func (h *Handler) HandlePost(w http.ResponseWriter, r *http.Request) {
    // Handle POST request
    w.WriteHeader(http.StatusOK)
    w.Write([]byte("POST request handled"))
}