package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"taskmanagement/internal/application/token"
	"taskmanagement/internal/modules/auth/domain"
	"taskmanagement/internal/platform/id"
	"taskmanagement/internal/platform/password"
	"time"
)

type authService struct {
	Repository   domain.Repository
	TokenService token.TokenService
}

func NewAuthService(authRepository domain.Repository, tokenService token.TokenService) domain.Service {
	return &authService{
		Repository:   authRepository,
		TokenService: tokenService,
	}
}

func (authService *authService) Register(ctx context.Context, input domain.RegisterInput) (domain.UserOutput, error) {
	trimmedName, normalizedEmail, plainPassword := strings.TrimSpace(input.Name), strings.ToLower(strings.TrimSpace(input.Email)), input.Password
	passwordHash, err := password.Hash(plainPassword)
	if err != nil {
		return domain.UserOutput{}, err
	}

	user := domain.User{
		ID:           id.New(),
		Name:         trimmedName,
		Email:        normalizedEmail,
		PasswordHash: passwordHash,
	}
	if err = authService.Repository.Create(ctx, user); err != nil {
		return domain.UserOutput{}, err
	}

	return domain.UserOutput{
		ID:    user.ID,
		Name:  trimmedName,
		Email: normalizedEmail,
	}, nil
}

func (authService *authService) Login(ctx context.Context, input domain.LoginInput) (domain.TokenPair, error) {
	user, err := authService.Repository.ByEmail(ctx, strings.ToLower(strings.TrimSpace(input.Email)))
	if err != nil || !password.Compare(user.PasswordHash, input.Password) {
		return domain.TokenPair{}, domain.ErrUnauthorized
	}

	return authService.pair(ctx, user.ID)
}

func (authService *authService) pair(ctx context.Context, userID string) (domain.TokenPair, error) {
	accessToken, ttl, err := authService.TokenService.Issue(ctx, userID)
	if err != nil {
		return domain.TokenPair{}, err
	}

	raw := id.New() + id.New()
	tokenHash := hash(raw)
	refreshExpiration := time.Now().UTC().Add(30 * 24 * time.Hour)
	if err = authService.Repository.CreateRefresh(ctx, id.New(), userID, tokenHash); err != nil {
		return domain.TokenPair{}, err
	}

	return domain.TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          raw,
		TokenType:             "Bearer",
		ExpiresIn:             ttl,
		AccessTokenExpiresAt:  time.Now().UTC().Add(time.Duration(ttl) * time.Second),
		RefreshTokenExpiresAt: refreshExpiration,
	}, nil
}

func (authService *authService) Refresh(ctx context.Context, input domain.RefreshInput) (domain.TokenPair, error) {
	raw := input.RefreshToken
	if raw == "" {
		return domain.TokenPair{}, domain.ErrUnauthorized
	}

	newRaw := id.New() + id.New()
	userID, err := authService.Repository.RotateRefresh(ctx, hash(raw), id.New(), "", hash(newRaw))
	if err != nil {
		return domain.TokenPair{}, err
	}
	accessToken, ttl, err := authService.TokenService.Issue(ctx, userID)
	if err != nil {
		return domain.TokenPair{}, err
	}

	now := time.Now().UTC()
	return domain.TokenPair{
		AccessToken:           accessToken,
		RefreshToken:          newRaw,
		TokenType:             "Bearer",
		ExpiresIn:             ttl,
		AccessTokenExpiresAt:  now.Add(time.Duration(ttl) * time.Second),
		RefreshTokenExpiresAt: now.Add(30 * 24 * time.Hour),
	}, nil
}

func (authService *authService) Logout(ctx context.Context, input domain.LogoutInput) error {
	return authService.Repository.DeleteRefresh(ctx, hash(input.RefreshToken))
}

func hash(value string) string {
	hashSum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(hashSum[:])
}
