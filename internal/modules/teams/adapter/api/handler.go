package api

import (
	"errors"
	"net/http"
	"strconv"
	"taskmanagement/internal/application/authorization"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/modules/teams/domain"
	phttp "taskmanagement/internal/platform/http"

	"github.com/go-chi/chi/v5"
)

type TeamHandler struct{ TeamService domain.Service }

func identity(request *http.Request) string {
	identity, _ := authorization.IdentityFrom(request.Context())
	return identity.UserID
}

func teamStatus(err error) int {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return 403
	case errors.Is(err, domain.ErrConflict):
		return 409
	case errors.Is(err, domain.ErrNotFound):
		return 404
	case errors.Is(err, domain.ErrInvalid):
		return 422
	default:
		return 500
	}
}

func page(request *http.Request) (int, int, []phttp.ValidationError) {
	pageRequest := PaginationRequest{
		Page:  1,
		Limit: 20,
	}
	if raw := request.URL.Query().Get("page"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil {
			pageRequest.Page = value
		} else {
			pageRequest.Page = 0
		}
	}
	if raw := request.URL.Query().Get("limit"); raw != "" {
		value, err := strconv.Atoi(raw)
		if err == nil {
			pageRequest.Limit = value
		} else {
			pageRequest.Limit = 0
		}
	}
	return pageRequest.Page, pageRequest.Limit, pageRequest.Validate()
}

func (handler TeamHandler) create(responseWriter http.ResponseWriter, request *http.Request) {
	var createRequest CreateRequest
	if !phttp.DecodeJSON(responseWriter, request, &createRequest) {
		return
	}
	if validationErrors := createRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}
	team, err := handler.TeamService.Create(request.Context(), domain.CreateInput{
		UserID: identity(request),
		Name:   createRequest.Name,
	})
	if err != nil {
		phttp.JSONResponse4xx(responseWriter, 422, errorcode.InvalidRequest, "invalid team")
		return
	}
	phttp.JSONResponse2xx(responseWriter, 201, team)
}

func (handler TeamHandler) list(responseWriter http.ResponseWriter, request *http.Request) {
	pageNumber, limit, validationErrors := page(request)
	if len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}
	teams, total, err := handler.TeamService.List(request.Context(), domain.ListInput{
		UserID: identity(request),
		Page:   pageNumber,
		Limit:  limit,
	})
	if err != nil {
		phttp.JSONResponse5xx(responseWriter, 500)
		return
	}
	phttp.JSONResponse2xx(responseWriter, 200, map[string]any{
		"data": teams,
		"meta": map[string]int{
			"page":  pageNumber,
			"limit": limit,
			"total": total,
		},
	})
}

func (handler TeamHandler) get(responseWriter http.ResponseWriter, request *http.Request) {
	team, err := handler.TeamService.Get(request.Context(), domain.GetInput{
		TeamID: chi.URLParam(request, "id"),
		UserID: identity(request),
	})
	if errors.Is(err, domain.ErrNotFound) {
		phttp.JSONResponse4xx(responseWriter, 404, errorcode.NotFound, "team not found")
		return
	}
	if err != nil {
		phttp.JSONResponse5xx(responseWriter, 500)
		return
	}
	phttp.JSONResponse2xx(responseWriter, 200, team)
}

func (handler TeamHandler) members(responseWriter http.ResponseWriter, request *http.Request) {
	pageNumber, limit, validationErrors := page(request)
	if len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}
	members, total, err := handler.TeamService.Members(request.Context(), domain.MembersInput{
		TeamID: chi.URLParam(request, "id"),
		UserID: identity(request),
		Page:   pageNumber,
		Limit:  limit,
	})
	if errors.Is(err, domain.ErrNotFound) {
		phttp.JSONResponse4xx(responseWriter, 404, errorcode.NotFound, "team not found")
		return
	}
	if err != nil {
		phttp.JSONResponse5xx(responseWriter, 500)
		return
	}
	phttp.JSONResponse2xx(responseWriter, 200, map[string]any{
		"data": members,
		"meta": map[string]int{
			"page":  pageNumber,
			"limit": limit,
			"total": total,
		},
	})
}

func (handler TeamHandler) add(responseWriter http.ResponseWriter, request *http.Request) {
	var addRequest AddRequest
	if !phttp.DecodeJSON(responseWriter, request, &addRequest) {
		return
	}
	if validationErrors := addRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}
	member, err := handler.TeamService.Add(request.Context(), domain.AddInput{
		TeamID:       chi.URLParam(request, "id"),
		UserID:       identity(request),
		TargetUserID: addRequest.UserID,
		Email:        addRequest.Email,
	})
	if err != nil {
		statusCode := teamStatus(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(responseWriter, 500)
		} else {
			phttp.JSONResponse4xx(responseWriter, statusCode, errorcode.InvalidRequest, "unable to add member")
		}
		return
	}
	phttp.JSONResponse2xx(responseWriter, 201, member)
}

func (handler TeamHandler) remove(responseWriter http.ResponseWriter, request *http.Request) {
	err := handler.TeamService.Remove(request.Context(), domain.RemoveInput{
		TeamID:       chi.URLParam(request, "id"),
		UserID:       identity(request),
		TargetUserID: chi.URLParam(request, "user_id"),
	})
	if err != nil {
		statusCode := teamStatus(err)
		if errors.Is(err, domain.ErrOwner) {
			statusCode = 403
		}
		if statusCode >= 500 {
			phttp.JSONResponse5xx(responseWriter, 500)
		} else {
			phttp.JSONResponse4xx(responseWriter, statusCode, errorcode.Forbidden, "unable to remove member")
		}
		return
	}
	phttp.NoContent(responseWriter)
}
