package api

import (
	"errors"
	"github.com/go-chi/chi/v5"
	"net/http"
	"strconv"
	"taskmanagement/internal/application/authorization"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/modules/tasks/domain"
	phttp "taskmanagement/internal/platform/http"
	"taskmanagement/internal/platform/id"
)

type TaskHandler struct{ TaskService domain.Service }

func userID(request *http.Request) string {
	identity, _ := authorization.IdentityFrom(request.Context())
	return identity.UserID
}

func classify(err error) (int, string) {
	switch {
	case errors.Is(err, domain.ErrForbidden):
		return 403, errorcode.Forbidden
	case errors.Is(err, domain.ErrNotFound):
		return 404, errorcode.NotFound
	case errors.Is(err, domain.ErrInvalid):
		return 422, errorcode.InvalidRequest
	default:
		return 500, errorcode.InternalError
	}
}

func (h TaskHandler) create(w http.ResponseWriter, r *http.Request) {
	var createRequest CreateRequest
	if !id.IsValid(r.Header.Get("Idempotency-Key")) {
		phttp.JSONResponse4xx(w, 400, errorcode.InvalidIdempotencyKey, "idempotency key required")
		return
	}
	if !phttp.DecodeJSON(w, r, &createRequest) {
		return
	}
	if validationErrors := createRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(w, 422, validationErrors)
		return
	}
	task := domain.Task{
		TeamID:      createRequest.TeamID,
		Title:       createRequest.Title,
		Description: createRequest.Description,
		Status:      createRequest.Status,
	}
	createResult, err := h.TaskService.Create(r.Context(), domain.CreateInput{
		UserID:         userID(r),
		IdempotencyKey: r.Header.Get("Idempotency-Key"),
		Task:           task,
	})
	if err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "unable to create task")
		}
		return
	}
	phttp.JSONBytes2xx(w, createResult.Status, createResult.Body)
}

func requireID(w http.ResponseWriter, v string) bool {
	if id.IsValid(v) {
		return true
	}
	phttp.JSONValidationError(w, 422, []phttp.ValidationError{{
		Field:   "id",
		Code:    errorcode.InvalidID,
		Message: "must be a UUID",
	}})
	return false
}

func (h TaskHandler) list(w http.ResponseWriter, r *http.Request) {
	listRequest := PaginationRequest{
		Page:   1,
		Limit:  20,
		Status: r.URL.Query().Get("status"),
		TeamID: r.URL.Query().Get("team_id"),
		Search: r.URL.Query().Get("search"),
	}
	if raw := r.URL.Query().Get("page"); raw != "" {
		page, err := strconv.Atoi(raw)
		if err == nil {
			listRequest.Page = page
		} else {
			listRequest.Page = 0
		}
	}
	if raw := r.URL.Query().Get("limit"); raw != "" {
		limit, err := strconv.Atoi(raw)
		if err == nil {
			listRequest.Limit = limit
		} else {
			listRequest.Limit = 0
		}
	}
	if validationErrors := listRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(w, 422, validationErrors)
		return
	}
	tasks, total, err := h.TaskService.List(r.Context(), domain.ListInput{
		UserID: userID(r),
		TeamID: listRequest.TeamID,
		Status: listRequest.Status,
		Search: listRequest.Search,
		Page:   listRequest.Page,
		Limit:  listRequest.Limit,
	})
	if err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "unable to list tasks")
		}
		return
	}
	phttp.JSONResponse2xx(w, 200, map[string]any{
		"data": tasks,
		"meta": map[string]int{
			"page":  listRequest.Page,
			"limit": listRequest.Limit,
			"total": total,
		},
	})
}

func (h TaskHandler) get(w http.ResponseWriter, r *http.Request) {
	if !requireID(w, chi.URLParam(r, "id")) {
		return
	}
	task, err := h.TaskService.Get(r.Context(), domain.GetInput{
		TaskID: chi.URLParam(r, "id"),
		UserID: userID(r),
	})
	if err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "task not found")
		}
		return
	}
	phttp.JSONResponse2xx(w, 200, task)
}

func (h TaskHandler) update(w http.ResponseWriter, r *http.Request) {
	if !requireID(w, chi.URLParam(r, "id")) {
		return
	}
	var updateRequest UpdateRequest
	if !phttp.DecodeJSON(w, r, &updateRequest) {
		return
	}
	if validationErrors := updateRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(w, 422, validationErrors)
		return
	}
	task := domain.Task{
		TeamID:      updateRequest.TeamID,
		Title:       updateRequest.Title,
		Description: updateRequest.Description,
		Status:      updateRequest.Status,
	}
	task, err := h.TaskService.Update(r.Context(), domain.UpdateInput{
		TaskID: chi.URLParam(r, "id"),
		UserID: userID(r),
		Task:   task,
	})
	if err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "unable to update task")
		}
		return
	}
	phttp.JSONResponse2xx(w, 200, task)
}

func (h TaskHandler) delete(w http.ResponseWriter, r *http.Request) {
	if !requireID(w, chi.URLParam(r, "id")) {
		return
	}
	if err := h.TaskService.Delete(r.Context(), domain.DeleteInput{
		TaskID: chi.URLParam(r, "id"),
		UserID: userID(r),
	}); err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "task not found")
		}
		return
	}
	phttp.NoContent(w)
}

func (h TaskHandler) assign(w http.ResponseWriter, r *http.Request) {
	if !requireID(w, chi.URLParam(r, "id")) {
		return
	}
	var assignRequest AssignRequest
	if !phttp.DecodeJSON(w, r, &assignRequest) {
		return
	}
	if validationErrors := assignRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(w, 422, validationErrors)
		return
	}
	task, err := h.TaskService.Assign(r.Context(), domain.AssignInput{
		TaskID:       chi.URLParam(r, "id"),
		UserID:       userID(r),
		TargetUserID: assignRequest.AssigneeID,
	})
	if err != nil {
		statusCode, errorCode := classify(err)
		if statusCode >= 500 {
			phttp.JSONResponse5xx(w, statusCode)
		} else {
			phttp.JSONResponse4xx(w, statusCode, errorCode, "unable to assign task")
		}
		return
	}
	phttp.JSONResponse2xx(w, 200, task)
}
