package api

import (
	"net/http"
	"taskmanagement/internal/modules/tasks/domain"

	"github.com/go-chi/chi/v5"
)

func Register(router chi.Router, taskService domain.Service, authenticationMiddleware func(http.Handler) http.Handler) {
	handler := TaskHandler{
		TaskService: taskService,
	}
	router.Route("/tasks", func(taskRouter chi.Router) {
		taskRouter.Use(authenticationMiddleware)
		taskRouter.Post("/", handler.create)
		taskRouter.Get("/", handler.list)
		taskRouter.Get("/{id}", handler.get)
		taskRouter.Put("/{id}", handler.update)
		taskRouter.Delete("/{id}", handler.delete)
		taskRouter.Post("/{id}/assign", handler.assign)
	})
}
