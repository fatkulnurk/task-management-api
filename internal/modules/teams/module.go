package teams

import (
	"database/sql"
	"net/http"
	"taskmanagement/internal/modules/teams/adapter/api"
	"taskmanagement/internal/modules/teams/domain"
	"taskmanagement/internal/modules/teams/repository"
	"taskmanagement/internal/modules/teams/service"

	"github.com/go-chi/chi/v5"
)

type TeamModule struct{ TeamService domain.Service }

func New(database *sql.DB) *TeamModule {
	teamRepository := repository.NewMySQLTeamRepository(database)
	teamService := service.NewTeamService(teamRepository)
	return &TeamModule{
		TeamService: teamService,
	}
}

func (teamModule *TeamModule) RegisterRoutes(router chi.Router, authenticationMiddleware func(http.Handler) http.Handler) {
	api.Register(router, teamModule.TeamService, authenticationMiddleware)
}
