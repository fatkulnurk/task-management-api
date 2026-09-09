package domain

import "context"

type Repository interface {
	Create(context.Context, User) error
	ByEmail(context.Context, string) (User, error)
	CreateRefresh(context.Context, string, string, string) error
	RotateRefresh(context.Context, string, string, string, string) (string, error)
	DeleteRefresh(context.Context, string) error
}

type User struct {
	ID           string
	Name         string
	Email        string
	PasswordHash string
}
