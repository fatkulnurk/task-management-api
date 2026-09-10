package http

import (
	"bytes"
	"encoding/json"
	"log/slog"
	stdhttp "net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskmanagement/internal/platform/id"
)

func TestStackRequestID(t *testing.T) {
	tests := []struct {
		name     string
		header   string
		wantSame bool
	}{
		{name: "generated when missing"},
		{
			name:     "kept when valid",
			header:   "9f8c1c1e-4d3b-4a2a-8f1e-0123456789ab",
			wantSame: true,
		},
		{
			name:   "replaced when invalid",
			header: "not-a-uuid",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var buffer bytes.Buffer
			logger := slog.New(slog.NewJSONHandler(&buffer, nil))
			contextRequestID := ""
			nextHandler := stdhttp.HandlerFunc(func(responseWriter stdhttp.ResponseWriter, request *stdhttp.Request) {
				contextRequestID = RequestID(request.Context())
				responseWriter.WriteHeader(stdhttp.StatusNoContent)
			})
			request := httptest.NewRequest(stdhttp.MethodGet, "/health", nil)
			if test.header != "" {
				request.Header.Set("X-Request-Id", test.header)
			}
			recorder := httptest.NewRecorder()

			Stack(nil, logger)(nextHandler).ServeHTTP(recorder, request)

			responseRequestID := recorder.Header().Get("X-Request-Id")
			if !id.IsValid(responseRequestID) {
				t.Fatalf("X-Request-Id = %q, want a UUID", responseRequestID)
			}
			if contextRequestID != responseRequestID {
				t.Errorf("context request id = %q, want %q", contextRequestID, responseRequestID)
			}
			if test.wantSame && responseRequestID != test.header {
				t.Errorf("X-Request-Id = %q, want %q", responseRequestID, test.header)
			}
			if !test.wantSame && responseRequestID == test.header {
				t.Errorf("X-Request-Id = %q, want a fresh UUID", responseRequestID)
			}

			var entry map[string]any
			if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
				t.Fatalf("failed to decode log entry: %v", err)
			}
			if entry["request_id"] != responseRequestID {
				t.Errorf("log request_id = %v, want %q", entry["request_id"], responseRequestID)
			}
			if entry["method"] != stdhttp.MethodGet {
				t.Errorf("log method = %v, want %q", entry["method"], stdhttp.MethodGet)
			}
			if entry["path"] != "/health" {
				t.Errorf("log path = %v, want %q", entry["path"], "/health")
			}
			if entry["status"] != float64(stdhttp.StatusNoContent) {
				t.Errorf("log status = %v, want %d", entry["status"], stdhttp.StatusNoContent)
			}
			if _, ok := entry["latency"]; !ok {
				t.Error("log entry must contain latency")
			}
		})
	}
}

func TestStackRecover(t *testing.T) {
	var buffer bytes.Buffer
	logger := slog.New(slog.NewJSONHandler(&buffer, nil))
	nextHandler := stdhttp.HandlerFunc(func(stdhttp.ResponseWriter, *stdhttp.Request) {
		panic("boom")
	})
	request := httptest.NewRequest(stdhttp.MethodGet, "/panic", nil)
	recorder := httptest.NewRecorder()

	Stack(nil, logger)(nextHandler).ServeHTTP(recorder, request)

	if recorder.Code != stdhttp.StatusInternalServerError {
		t.Fatalf("status = %d, want %d", recorder.Code, stdhttp.StatusInternalServerError)
	}

	var envelope map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
		t.Fatalf("failed to decode response envelope: %v", err)
	}
	if envelope["code"] != "internal_error" {
		t.Errorf("code = %v, want %q", envelope["code"], "internal_error")
	}
	if envelope["message"] != "internal server error" {
		t.Errorf("message = %v, want %q", envelope["message"], "internal server error")
	}
	if strings.Contains(recorder.Body.String(), "boom") {
		t.Errorf("panic detail leaked in response: %s", recorder.Body.String())
	}

	decoder := json.NewDecoder(&buffer)
	var panicEntry map[string]any
	if err := decoder.Decode(&panicEntry); err != nil {
		t.Fatalf("failed to decode panic log entry: %v", err)
	}
	if panicEntry["level"] != "ERROR" {
		t.Errorf("level = %v, want %q", panicEntry["level"], "ERROR")
	}
	if panicEntry["msg"] != "panic" {
		t.Errorf("msg = %v, want %q", panicEntry["msg"], "panic")
	}
	if panicEntry["panic"] != "boom" {
		t.Errorf("panic = %v, want %q", panicEntry["panic"], "boom")
	}
	requestID, _ := panicEntry["request_id"].(string)
	if !id.IsValid(requestID) {
		t.Errorf("request_id = %q, want a UUID", requestID)
	}
	if requestID != recorder.Header().Get("X-Request-Id") {
		t.Errorf("log request_id = %q, want %q", requestID, recorder.Header().Get("X-Request-Id"))
	}
	stack, _ := panicEntry["stack"].(string)
	if !strings.Contains(stack, "middleware_test.go") {
		t.Errorf("stack = %q, want it to contain the panic site", stack)
	}
}
