package http

import (
	"net/http"
	"testing"
	"time"

	"github.com/go-chi/chi/v5"
)

func TestNewServer(t *testing.T) {
	router := chi.NewRouter()

	server := NewServer(":8080", router)

	if server.Addr != ":8080" {
		t.Errorf("addr = %q, want :8080", server.Addr)
	}
	if server.Handler == nil {
		t.Error("handler must not be nil")
	}
	if server.ReadHeaderTimeout != 5*time.Second {
		t.Errorf("read header timeout = %s, want 5s", server.ReadHeaderTimeout)
	}
	if server.ReadTimeout != 15*time.Second {
		t.Errorf("read timeout = %s, want 15s", server.ReadTimeout)
	}
	if server.WriteTimeout != 15*time.Second {
		t.Errorf("write timeout = %s, want 15s", server.WriteTimeout)
	}
	if server.IdleTimeout != 60*time.Second {
		t.Errorf("idle timeout = %s, want 60s", server.IdleTimeout)
	}
}

func TestNewServerHandler(t *testing.T) {
	router := chi.NewRouter()
	router.Get("/health", func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusNoContent)
	})

	server := NewServer(":0", router)

	if server.Handler != router {
		t.Error("handler must be the given router")
	}
}
