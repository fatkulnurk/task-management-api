package domain

import "context"

type Repository interface {
	Create(context.Context, Team) error
	Add(context.Context, string, string) error
	List(context.Context, string, int, int) ([]Team, int, error)
	Get(context.Context, string, string) (Team, error)
	Members(context.Context, string, string, int, int) ([]Member, int, error)
	User(context.Context, string, string) (Member, error)
	Remove(context.Context, string, string) error
	IsMember(context.Context, string, string) (bool, error)
	IsOwner(context.Context, string, string) (bool, error)
	ActiveAssignments(context.Context, string, string) (bool, error)
}

type Team struct {
	ID        string `json:"id"`
	OwnerID   string `json:"owner_id"`
	Name      string `json:"name"`
	CreatedAt string `json:"created_at"`
	UpdatedAt string `json:"updated_at"`
}

type Member struct {
	ID      string `json:"id"`
	Name    string `json:"name"`
	Email   string `json:"email"`
	IsOwner bool   `json:"is_owner"`
}
