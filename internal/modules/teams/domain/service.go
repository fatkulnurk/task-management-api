package domain

import "context"

type Service interface {
	Create(context.Context, CreateInput) (Team, error)
	List(context.Context, ListInput) ([]Team, int, error)
	Get(context.Context, GetInput) (Team, error)
	Members(context.Context, MembersInput) ([]Member, int, error)
	Add(context.Context, AddInput) (Member, error)
	Remove(context.Context, RemoveInput) error
}

type CreateInput struct {
	UserID string
	Name   string
}

type ListInput struct {
	UserID string
	Page   int
	Limit  int
}

type GetInput struct {
	TeamID string
	UserID string
}

type MembersInput struct {
	TeamID string
	UserID string
	Page   int
	Limit  int
}

type AddInput struct {
	TeamID       string
	UserID       string
	TargetUserID string
	Email        string
}

type RemoveInput struct {
	TeamID       string
	UserID       string
	TargetUserID string
}
