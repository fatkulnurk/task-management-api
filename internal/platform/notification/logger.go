package notification

import (
	"context"
	"log/slog"
	applicationnotification "taskmanagement/internal/application/notification"
)

type Logger struct {
	Logger *slog.Logger
}

func NewLogger(logger *slog.Logger) applicationnotification.NotificationService {
	return &Logger{Logger: logger}
}

func (notificationLogger *Logger) Send(ctx context.Context, notification applicationnotification.Notification) error {
	notificationLogger.Logger.InfoContext(ctx, "task notification",
		"user_id", notification.UserID,
		"task_id", notification.TaskID,
		"message", notification.Message,
	)
	return nil
}
