package notification

import "context"

type Notification struct {
	UserID  string
	TaskID  string
	Message string
}
type NotificationService interface {
	Send(ctx context.Context, notification Notification) error
}
