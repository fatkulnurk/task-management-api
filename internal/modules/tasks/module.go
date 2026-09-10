package tasks

import (
	"database/sql"
	"net/http"
	"taskmanagement/internal/application/notification"
	"taskmanagement/internal/modules/tasks/adapter/api"
	"taskmanagement/internal/modules/tasks/domain"
	"taskmanagement/internal/modules/tasks/repository"
	"taskmanagement/internal/modules/tasks/service"

	"github.com/go-chi/chi/v5"
)

type TaskModule struct{ TaskService domain.Service }

func New(database *sql.DB, notificationService notification.NotificationService) *TaskModule {
	taskRepository := repository.NewMySQLTaskRepository(database)
	taskService := service.NewTaskService(taskRepository, notificationService)
	return &TaskModule{
		TaskService: taskService,
	}
}

func (taskModule *TaskModule) RegisterRoutes(router chi.Router, authenticationMiddleware func(http.Handler) http.Handler) {
	api.Register(router, taskModule.TaskService, authenticationMiddleware)
}
