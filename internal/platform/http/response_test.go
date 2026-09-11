package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func decodeEnvelope(t *testing.T, recorder *httptest.ResponseRecorder) envelope {
	t.Helper()
	var decoded envelope
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode envelope: %v", err)
	}
	return decoded
}

func assertRFC3339UTC(t *testing.T, timestamp string) {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, timestamp)
	if err != nil {
		t.Fatalf("timestamp %q is not RFC3339: %v", timestamp, err)
	}
	if parsed.UTC().Format(time.RFC3339) != timestamp {
		t.Errorf("timestamp %q is not in UTC", timestamp)
	}
}

func TestJSONResponse4xx(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONResponse4xx(recorder, 418, "teapot", "short and stout")

	if recorder.Code != 418 {
		t.Fatalf("status = %d, want 418", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("content type = %q, want application/json", contentType)
	}
	decoded := decodeEnvelope(t, recorder)
	if decoded.Status != 418 {
		t.Errorf("status field = %d, want 418", decoded.Status)
	}
	if decoded.Code != "teapot" || decoded.Message != "short and stout" {
		t.Errorf("code/message = %q/%q", decoded.Code, decoded.Message)
	}
	if len(decoded.Errors) != 0 {
		t.Errorf("errors = %v, want none", decoded.Errors)
	}
	assertRFC3339UTC(t, decoded.Timestamp)
}

func TestJSONResponse5xx(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONResponse5xx(recorder, 500)

	if recorder.Code != 500 {
		t.Fatalf("status = %d, want 500", recorder.Code)
	}
	decoded := decodeEnvelope(t, recorder)
	if decoded.Code != "internal_error" {
		t.Errorf("code = %q, want internal_error", decoded.Code)
	}
	if decoded.Message != "internal server error" {
		t.Errorf("message = %q, want the generic message", decoded.Message)
	}
	assertRFC3339UTC(t, decoded.Timestamp)
}

func TestJSONValidationError(t *testing.T) {
	recorder := httptest.NewRecorder()
	wantErrors := []ValidationError{{Field: "title", Code: "invalid_field", Message: "is required"}}

	JSONValidationError(recorder, 422, wantErrors)

	if recorder.Code != 422 {
		t.Fatalf("status = %d, want 422", recorder.Code)
	}
	decoded := decodeEnvelope(t, recorder)
	if decoded.Code != "invalid_request" {
		t.Errorf("code = %q, want invalid_request", decoded.Code)
	}
	if len(decoded.Errors) != 1 || decoded.Errors[0] != wantErrors[0] {
		t.Errorf("errors = %+v, want %+v", decoded.Errors, wantErrors)
	}
	assertRFC3339UTC(t, decoded.Timestamp)
}

func TestJSONResponse2xx(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONResponse2xx(recorder, 201, map[string]string{"id": "task-1"})

	if recorder.Code != 201 {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if contentType := recorder.Header().Get("Content-Type"); contentType != "application/json" {
		t.Errorf("content type = %q, want application/json", contentType)
	}
	var decoded map[string]string
	if err := json.Unmarshal(recorder.Body.Bytes(), &decoded); err != nil {
		t.Fatalf("failed to decode body: %v", err)
	}
	if decoded["id"] != "task-1" {
		t.Errorf("id = %q, want task-1", decoded["id"])
	}
}

func TestJSONResponse3xx(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONResponse3xx(recorder, 304, map[string]string{"note": "cached"})

	if recorder.Code != 304 {
		t.Fatalf("status = %d, want 304", recorder.Code)
	}
}

func TestJSONBytes2xx(t *testing.T) {
	recorder := httptest.NewRecorder()
	body := []byte(`{"id":"task-1"}`)

	JSONBytes2xx(recorder, 201, body)

	if recorder.Code != 201 {
		t.Fatalf("status = %d, want 201", recorder.Code)
	}
	if recorder.Body.String() != string(body) {
		t.Errorf("body = %s, want %s", recorder.Body.String(), body)
	}
}

func TestWriteHasNoTrailingNewline(t *testing.T) {
	recorder := httptest.NewRecorder()

	JSONResponse2xx(recorder, 200, map[string]string{"status": "ok"})

	body := recorder.Body.Bytes()
	if len(body) == 0 || body[len(body)-1] == '\n' {
		t.Fatalf("body ends with a newline: %q", body)
	}
}

func TestNoContent(t *testing.T) {
	recorder := httptest.NewRecorder()

	NoContent(recorder)

	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if contentLength := recorder.Header().Get("Content-Length"); contentLength != "0" {
		t.Errorf("content length = %q, want 0", contentLength)
	}
	if recorder.Body.Len() != 0 {
		t.Errorf("body = %q, want empty", recorder.Body.String())
	}
}
