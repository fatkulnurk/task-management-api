package http

import (
	"github.com/go-chi/chi/v5"
	stdhttp "net/http"
	"time"
)

func NewServer(addr string, router chi.Router) *stdhttp.Server {
	return &stdhttp.Server{
		Addr:              addr,
		Handler:           router,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       15 * time.Second,
		WriteTimeout:      15 * time.Second,
		IdleTimeout:       60 * time.Second,
	}
}
