package notification

import (
	"bytes"
	"context"
	"encoding/json"
	"log/slog"
	"testing"

	applicationnotification "taskmanagement/internal/application/notification"
)

func TestSendLogsNotification(t *testing.T) {
	var buffer bytes.Buffer
	logger := NewLogger(slog.New(slog.NewJSONHandler(&buffer, nil)))

	err := logger.Send(context.Background(), applicationnotification.Notification{
		UserID:  "user-2",
		TaskID:  "task-1",
		Message: "task assigned",
	})
	if err != nil {
		t.Fatalf("send error: %v", err)
	}

	var entry map[string]any
	if err := json.Unmarshal(buffer.Bytes(), &entry); err != nil {
		t.Fatalf("log line is not JSON: %v (line=%q)", err, buffer.String())
	}
	if entry["user_id"] != "user-2" {
		t.Errorf("user_id = %v, want user-2", entry["user_id"])
	}
	if entry["task_id"] != "task-1" {
		t.Errorf("task_id = %v, want task-1", entry["task_id"])
	}
	if entry["message"] != "task assigned" {
		t.Errorf("message = %v, want task assigned", entry["message"])
	}
}

func TestNewLogger(t *testing.T) {
	if NewLogger(slog.Default()) == nil {
		t.Fatal("expected a non-nil logger")
	}
}
