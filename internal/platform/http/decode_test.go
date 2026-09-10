package http

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestDecodeJSON(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		wantOK     bool
		wantStatus int
		wantCode   string
	}{
		{name: "valid", body: `{"name":"Alice"}`, wantOK: true},
		{name: "empty body", body: "", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "whitespace only", body: "   ", wantStatus: http.StatusBadRequest, wantCode: "invalid_request"},
		{name: "malformed", body: `{"name":`, wantStatus: http.StatusBadRequest, wantCode: "invalid_json"},
		{name: "trailing data", body: `{"name":"Alice"}{"name":"Bob"}`, wantStatus: http.StatusBadRequest, wantCode: "invalid_json"},
		{name: "too large", body: `{"name":"` + strings.Repeat("a", maxRequestBodyBytes+16) + `"}`, wantStatus: http.StatusRequestEntityTooLarge, wantCode: "payload_too_large"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			request := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.body))
			var target struct {
				Name string `json:"name"`
			}

			ok := DecodeJSON(recorder, request, &target)
			if ok != test.wantOK {
				t.Fatalf("ok = %v, want %v", ok, test.wantOK)
			}
			if test.wantOK {
				if target.Name != "Alice" {
					t.Errorf("name = %q, want %q", target.Name, "Alice")
				}
				return
			}
			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			var envelope struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("failed to decode envelope: %v", err)
			}
			if envelope.Code != test.wantCode {
				t.Errorf("code = %q, want %q", envelope.Code, test.wantCode)
			}
		})
	}
}
