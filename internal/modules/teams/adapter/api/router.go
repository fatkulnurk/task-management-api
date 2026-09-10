package api

import (
	"github.com/go-chi/chi/v5"
	"net/http"
	"taskmanagement/internal/modules/teams/domain"
)

func Register(router chi.Router, teamService domain.Service, authenticationMiddleware func(http.Handler) http.Handler) {
	handler := TeamHandler{TeamService: teamService}
	router.Route("/teams", func(teamRouter chi.Router) {
		teamRouter.Use(authenticationMiddleware)
		teamRouter.Post("/", handler.create)
		teamRouter.Get("/", handler.list)
		teamRouter.Get("/{id}", handler.get)
		teamRouter.Get("/{id}/members", handler.members)
		teamRouter.Post("/{id}/members", handler.add)
		teamRouter.Delete("/{id}/members/{user_id}", handler.remove)
	})
}
