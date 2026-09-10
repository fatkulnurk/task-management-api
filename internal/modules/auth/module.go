package auth

import (
	"database/sql"
	"taskmanagement/internal/application/token"
	"taskmanagement/internal/modules/auth/adapter/api"
	"taskmanagement/internal/modules/auth/domain"
	"taskmanagement/internal/modules/auth/repository"
	"taskmanagement/internal/modules/auth/service"

	"github.com/go-chi/chi/v5"
)

type AuthModule struct{ AuthService domain.Service }

func New(database *sql.DB, tokenService token.TokenService) *AuthModule {
	authRepository := repository.NewMySQLAuthRepository(database)
	authService := service.NewAuthService(authRepository, tokenService)
	return &AuthModule{
		AuthService: authService,
	}
}

func (authModule *AuthModule) RegisterRoutes(router chi.Router) {
	api.Register(router, authModule.AuthService)
}
