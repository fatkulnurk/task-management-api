package domain

import (
	"context"
	"time"
)

type Service interface {
	Register(context.Context, RegisterInput) (UserOutput, error)
	Login(context.Context, LoginInput) (TokenPair, error)
	Refresh(context.Context, RefreshInput) (TokenPair, error)
	Logout(context.Context, LogoutInput) error
}

type RegisterInput struct {
	Name     string
	Email    string
	Password string
}

type LoginInput struct {
	Email    string
	Password string
}

type RefreshInput struct {
	RefreshToken string
}

type LogoutInput struct {
	RefreshToken string
}

type UserOutput struct {
	ID    string
	Name  string
	Email string
}

type TokenPair struct {
	AccessToken           string
	RefreshToken          string
	TokenType             string
	ExpiresIn             int64
	AccessTokenExpiresAt  time.Time
	RefreshTokenExpiresAt time.Time
}
