package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"testing"

	tokenmocks "taskmanagement/internal/application/token/mocks"
	"taskmanagement/internal/modules/auth/domain"
	domainmocks "taskmanagement/internal/modules/auth/domain/mocks"
	"taskmanagement/internal/platform/password"

	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func newAuthService(t *testing.T) (*authService, *domainmocks.MockRepository, *tokenmocks.MockTokenService) {
	t.Helper()
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)
	tokenService := tokenmocks.NewMockTokenService(controller)
	return &authService{Repository: repository, TokenService: tokenService}, repository, tokenService
}

func TestRegister(t *testing.T) {
	tests := []struct {
		name         string
		input        domain.RegisterInput
		setup        func(t *testing.T, repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService)
		wantError    error
		wantAnyError bool
		check        func(t *testing.T, output domain.UserOutput)
	}{
		{
			name: "success",
			input: domain.RegisterInput{
				Name:     "  Alice Smith  ",
				Email:    "  ALICE@fatkulnurk.com  ",
				Password: "correct-horse",
			},
			setup: func(t *testing.T, repository *domainmocks.MockRepository, _ *tokenmocks.MockTokenService) {
				repository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, user domain.User) error {
						if user.PasswordHash == "" || user.PasswordHash == "correct-horse" {
							t.Errorf("password must be hashed, got %q", user.PasswordHash)
						}
						if !password.Compare(user.PasswordHash, "correct-horse") {
							t.Errorf("stored hash does not match the plaintext password")
						}
						return nil
					})
			},
			check: func(t *testing.T, output domain.UserOutput) {
				if output.Name != "Alice Smith" {
					t.Errorf("name = %q, want %q", output.Name, "Alice Smith")
				}
				if output.Email != "alice@fatkulnurk.com" {
					t.Errorf("email = %q, want %q", output.Email, "alice@fatkulnurk.com")
				}
				if output.ID == "" {
					t.Error("output id must not be empty")
				}
			},
		},
		{
			name: "repository conflict",
			input: domain.RegisterInput{
				Name:     "Alice",
				Email:    "alice@fatkulnurk.com",
				Password: "correct-horse",
			},
			setup: func(_ *testing.T, repository *domainmocks.MockRepository, _ *tokenmocks.MockTokenService) {
				repository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(domain.ErrConflict)
			},
			wantError: domain.ErrConflict,
		},
		{
			name: "password too long",
			input: domain.RegisterInput{
				Name:     "Alice",
				Email:    "alice@fatkulnurk.com",
				Password: strings.Repeat("a", 73),
			},
			setup: func(_ *testing.T, repository *domainmocks.MockRepository, _ *tokenmocks.MockTokenService) {
				repository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantAnyError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authService, repository, tokenService := newAuthService(t)
			test.setup(t, repository, tokenService)

			output, err := authService.Register(context.Background(), test.input)
			if test.wantAnyError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
			} else if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, output)
			}
		})
	}
}

func TestLogin(t *testing.T) {
	passwordHash, hashErr := password.Hash("correct-horse")
	if hashErr != nil {
		t.Fatalf("failed to prepare password hash: %v", hashErr)
	}

	tests := []struct {
		name      string
		input     domain.LoginInput
		setup     func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService)
		wantError error
		check     func(t *testing.T, pair domain.TokenPair)
	}{
		{
			name: "success",
			input: domain.LoginInput{
				Email:    "  ALICE@fatkulnurk.com  ",
				Password: "correct-horse",
			},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					ByEmail(gomock.Any(), "alice@fatkulnurk.com").
					Return(domain.User{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", PasswordHash: passwordHash}, nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), "user-1").
					Return("access-token", int64(900), nil)
				repository.EXPECT().
					CreateRefresh(gomock.Any(), gomock.Any(), "user-1", gomock.Any()).
					Return(nil)
			},
			check: func(t *testing.T, pair domain.TokenPair) {
				if pair.AccessToken != "access-token" {
					t.Errorf("access token = %q, want %q", pair.AccessToken, "access-token")
				}
				if pair.RefreshToken == "" {
					t.Error("refresh token must not be empty")
				}
				if pair.TokenType != "Bearer" {
					t.Errorf("token type = %q, want %q", pair.TokenType, "Bearer")
				}
				if pair.ExpiresIn != 900 {
					t.Errorf("expires in = %d, want 900", pair.ExpiresIn)
				}
			},
		},
		{
			name: "wrong password",
			input: domain.LoginInput{
				Email:    "alice@fatkulnurk.com",
				Password: "wrong-password",
			},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					ByEmail(gomock.Any(), "alice@fatkulnurk.com").
					Return(domain.User{ID: "user-1", PasswordHash: passwordHash}, nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name: "unknown email",
			input: domain.LoginInput{
				Email:    "missing@fatkulnurk.com",
				Password: "correct-horse",
			},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					ByEmail(gomock.Any(), "missing@fatkulnurk.com").
					Return(domain.User{}, domain.ErrUnauthorized)
				tokenService.EXPECT().
					Issue(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name: "issue token fails",
			input: domain.LoginInput{
				Email:    "alice@fatkulnurk.com",
				Password: "correct-horse",
			},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					ByEmail(gomock.Any(), "alice@fatkulnurk.com").
					Return(domain.User{ID: "user-1", PasswordHash: passwordHash}, nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), "user-1").
					Return("", int64(0), errTest)
			},
			wantError: errTest,
		},
		{
			name: "create refresh fails",
			input: domain.LoginInput{
				Email:    "alice@fatkulnurk.com",
				Password: "correct-horse",
			},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					ByEmail(gomock.Any(), "alice@fatkulnurk.com").
					Return(domain.User{ID: "user-1", PasswordHash: passwordHash}, nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), "user-1").
					Return("access-token", int64(900), nil)
				repository.EXPECT().
					CreateRefresh(gomock.Any(), gomock.Any(), "user-1", gomock.Any()).
					Return(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authService, repository, tokenService := newAuthService(t)
			test.setup(repository, tokenService)

			pair, err := authService.Login(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, pair)
			}
		})
	}
}

func TestRefresh(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.RefreshInput
		setup     func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService)
		wantError error
		check     func(t *testing.T, pair domain.TokenPair)
	}{
		{
			name:  "empty token",
			input: domain.RefreshInput{RefreshToken: ""},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					RotateRefresh(gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any(), gomock.Any()).
					Times(0)
				tokenService.EXPECT().
					Issue(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name:  "success",
			input: domain.RefreshInput{RefreshToken: "old-refresh-token"},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					RotateRefresh(gomock.Any(), gomock.Any(), gomock.Any(), "", gomock.Any()).
					Return("user-1", nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), "user-1").
					Return("new-access-token", int64(900), nil)
			},
			check: func(t *testing.T, pair domain.TokenPair) {
				if pair.AccessToken != "new-access-token" {
					t.Errorf("access token = %q, want %q", pair.AccessToken, "new-access-token")
				}
				if pair.RefreshToken == "" || pair.RefreshToken == "old-refresh-token" {
					t.Errorf("refresh token must be rotated, got %q", pair.RefreshToken)
				}
			},
		},
		{
			name:  "rotate unauthorized",
			input: domain.RefreshInput{RefreshToken: "stale-token"},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					RotateRefresh(gomock.Any(), gomock.Any(), gomock.Any(), "", gomock.Any()).
					Return("", domain.ErrUnauthorized)
				tokenService.EXPECT().
					Issue(gomock.Any(), gomock.Any()).
					Times(0)
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name:  "issue token fails",
			input: domain.RefreshInput{RefreshToken: "old-refresh-token"},
			setup: func(repository *domainmocks.MockRepository, tokenService *tokenmocks.MockTokenService) {
				repository.EXPECT().
					RotateRefresh(gomock.Any(), gomock.Any(), gomock.Any(), "", gomock.Any()).
					Return("user-1", nil)
				tokenService.EXPECT().
					Issue(gomock.Any(), "user-1").
					Return("", int64(0), errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authService, repository, tokenService := newAuthService(t)
			test.setup(repository, tokenService)

			pair, err := authService.Refresh(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, pair)
			}
		})
	}
}

func TestLogout(t *testing.T) {
	expectedHash := sha256.Sum256([]byte("refresh-token"))
	expected := hex.EncodeToString(expectedHash[:])

	tests := []struct {
		name      string
		input     domain.LogoutInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
	}{
		{
			name:  "hashes token",
			input: domain.LogoutInput{RefreshToken: "refresh-token"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					DeleteRefresh(gomock.Any(), expected).
					Return(nil)
			},
		},
		{
			name:  "propagates error",
			input: domain.LogoutInput{RefreshToken: "refresh-token"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					DeleteRefresh(gomock.Any(), gomock.Any()).
					Return(domain.ErrUnauthorized)
			},
			wantError: domain.ErrUnauthorized,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			authService, repository, _ := newAuthService(t)
			test.setup(repository)

			err := authService.Logout(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
		})
	}
}

func TestHash(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "refresh token", input: "refresh-token"},
		{name: "other token", input: "other-token"},
		{name: "empty", input: ""},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			first := hash(test.input)
			second := hash(test.input)
			if first != second {
				t.Errorf("hash is not deterministic: %q != %q", first, second)
			}
			if len(first) != 64 {
				t.Errorf("hash length = %d, want 64", len(first))
			}
		})
	}
}

func TestHashDifferentInputs(t *testing.T) {
	if hash("refresh-token") == hash("other-token") {
		t.Error("different inputs must produce different hashes")
	}
}

func TestNewAuthService(t *testing.T) {
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)
	tokenService := tokenmocks.NewMockTokenService(controller)

	if NewAuthService(repository, tokenService) == nil {
		t.Fatal("expected a non-nil service")
	}
}
