package domain

import "context"

type Repository interface {
	Member(context.Context, string, string) (bool, error)
	CreateIdempotent(context.Context, Task, string, string, string, []byte) (CreateOutput, error)
	List(context.Context, string, string, string, string, int, int) ([]Task, int, error)
	Get(context.Context, string, string) (Task, error)
	Update(context.Context, Task, string) error
	Delete(context.Context, string, string) error
	Assign(context.Context, string, string, *string, string, func() error) error
}

type Task struct {
	ID          string  `json:"id"`
	TeamID      string  `json:"team_id"`
	CreatorID   string  `json:"creator_id"`
	AssigneeID  *string `json:"assignee_id"`
	Title       string  `json:"title"`
	Description string  `json:"description"`
	Status      string  `json:"status"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}
