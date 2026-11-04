package main

import (
	"crypto/tls"
	"log"
	"net/http"
	"time"

	cert "gofiles/certs"
)

func NewServer(handler http.Handler) *http.Server {
	cert, err := cert.LocalCert()
	if err != nil {
		log.Fatalf("Failed to load certificate: %v", err)
	}

	s := &http.Server{
		Addr:           ":443",
		Handler:        handler,
		MaxHeaderBytes: 1 << 20,
		IdleTimeout:    time.Minute,
		ReadTimeout:    5 * time.Second,
		WriteTimeout:   10 * time.Second,
		TLSConfig: &tls.Config{
			Certificates: []tls.Certificate{cert},
		},
	}

	return s
}
