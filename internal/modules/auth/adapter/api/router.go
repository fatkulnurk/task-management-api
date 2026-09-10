package api

import (
	"github.com/go-chi/chi/v5"
	"taskmanagement/internal/modules/auth/domain"
)

func Register(router chi.Router, authService domain.Service) {
	handler := AuthHandler{AuthService: authService}

	router.Post("/auth/register", handler.register)
	router.Post("/auth/login", handler.login)
	router.Post("/auth/refresh", handler.refresh)
	router.Post("/auth/logout", handler.logout)
}
