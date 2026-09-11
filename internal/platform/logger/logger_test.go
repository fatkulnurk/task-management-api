package logger

import (
	"bytes"
	"encoding/json"
	"testing"
)

func TestNewWritesJSON(t *testing.T) {
	var buffer bytes.Buffer

	logger := New(&buffer)
	logger.Info("http", "request_id", "abc", "status", 200)

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (line=%q)", err, buffer.String())
	}
	if entry["level"] != "INFO" {
		t.Errorf("level = %v, want INFO", entry["level"])
	}
	if entry["msg"] != "http" {
		t.Errorf("msg = %v, want http", entry["msg"])
	}
	if entry["request_id"] != "abc" {
		t.Errorf("request_id = %v, want abc", entry["request_id"])
	}
	if entry["status"] != float64(200) {
		t.Errorf("status = %v, want 200", entry["status"])
	}
}

func TestNewWritesOneLinePerCall(t *testing.T) {
	var buffer bytes.Buffer

	logger := New(&buffer)
	logger.Info("first")
	logger.Warn("second")

	lines := bytes.Split(bytes.TrimSpace(buffer.Bytes()), []byte("\n"))
	if len(lines) != 2 {
		t.Fatalf("line count = %d, want 2", len(lines))
	}
	for _, line := range lines {
		var entry map[string]any
		if err := json.Unmarshal(line, &entry); err != nil {
			t.Fatalf("log line is not JSON: %v", err)
		}
	}
}
