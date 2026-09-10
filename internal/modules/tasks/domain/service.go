package domain

import "context"

type Service interface {
	Create(context.Context, CreateInput) (CreateOutput, error)
	List(context.Context, ListInput) ([]Task, int, error)
	Get(context.Context, GetInput) (Task, error)
	Update(context.Context, UpdateInput) (Task, error)
	Delete(context.Context, DeleteInput) error
	Assign(context.Context, AssignInput) (Task, error)
}

type CreateInput struct {
	UserID         string
	IdempotencyKey string
	Task           Task
}

type ListInput struct {
	UserID string
	TeamID string
	Status string
	Search string
	Page   int
	Limit  int
}

type GetInput struct {
	TaskID string
	UserID string
}

type UpdateInput struct {
	TaskID string
	UserID string
	Task   Task
}

type DeleteInput struct {
	TaskID string
	UserID string
}

type AssignInput struct {
	TaskID       string
	UserID       string
	TargetUserID string
}

type CreateOutput struct {
	Status int
	Body   []byte
	Replay bool
}
