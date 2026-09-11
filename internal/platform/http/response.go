package http

import (
	"encoding/json"
	stdhttp "net/http"
	"taskmanagement/internal/application/errorcode"
	"time"
)

type ValidationError struct {
	Field   string `json:"field"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type envelope struct {
	Status    int               `json:"status"`
	Code      string            `json:"code,omitempty"`
	Message   string            `json:"message,omitempty"`
	Timestamp string            `json:"timestamp"`
	Errors    []ValidationError `json:"errors,omitempty"`
}

func write(responseWriter stdhttp.ResponseWriter, status int, value any) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	body, err := json.Marshal(value)
	if err != nil {
		return
	}
	_, _ = responseWriter.Write(body)
}

func JSONResponse2xx(responseWriter stdhttp.ResponseWriter, status int, value any) {
	write(responseWriter, status, value)
}

func JSONBytes2xx(responseWriter stdhttp.ResponseWriter, status int, body []byte) {
	responseWriter.Header().Set("Content-Type", "application/json")
	responseWriter.WriteHeader(status)
	_, _ = responseWriter.Write(body)
}

func JSONResponse3xx(responseWriter stdhttp.ResponseWriter, status int, value any) {
	write(responseWriter, status, value)
}

func JSONResponse4xx(w stdhttp.ResponseWriter, status int, code, message string) {
	write(w, status, envelope{
		Status:    status,
		Code:      code,
		Message:   message,
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Errors:    nil,
	})
}

func JSONResponse5xx(w stdhttp.ResponseWriter, status int) {
	write(w, status, envelope{
		Status:    status,
		Code:      errorcode.InternalError,
		Message:   "internal server error",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Errors:    nil,
	})
}

func JSONValidationError(w stdhttp.ResponseWriter, status int, es []ValidationError) {
	write(w, status, envelope{
		Status:    status,
		Code:      errorcode.InvalidRequest,
		Message:   "request validation failed",
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Errors:    es,
	})
}

func NoContent(w stdhttp.ResponseWriter) {
	w.Header().Set("Content-Length", "0")
	w.WriteHeader(stdhttp.StatusNoContent)
}
