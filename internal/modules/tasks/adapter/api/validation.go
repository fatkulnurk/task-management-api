package api

import (
	"taskmanagement/internal/application/errorcode"
	phttp "taskmanagement/internal/platform/http"
	"taskmanagement/internal/platform/id"
)

func (request CreateRequest) Validate() []phttp.ValidationError {
	var errors []phttp.ValidationError
	if request.TeamID == "" {
		errors = append(errors, phttp.ValidationError{
			Field:   "team_id",
			Code:    errorcode.InvalidTeamID,
			Message: "is required",
		})
	}
	if request.Title == "" || len(request.Title) > 255 {
		errors = append(errors, phttp.ValidationError{
			Field:   "title",
			Code:    errorcode.InvalidField,
			Message: "is required and must be at most 255 characters",
		})
	}
	if request.Status == "" {
		errors = append(errors, phttp.ValidationError{
			Field:   "status",
			Code:    errorcode.InvalidStatus,
			Message: "is required",
		})
	} else if request.Status != errorcode.TaskStatusTodo && request.Status != errorcode.TaskStatusInProgress && request.Status != errorcode.TaskStatusDone {
		errors = append(errors, phttp.ValidationError{
			Field:   "status",
			Code:    errorcode.InvalidStatus,
			Message: "is invalid",
		})
	}
	return errors
}

func (request UpdateRequest) Validate() []phttp.ValidationError {
	var errors []phttp.ValidationError
	if request.Title != "" && len(request.Title) > 255 {
		errors = append(errors, phttp.ValidationError{
			Field:   "title",
			Code:    errorcode.InvalidField,
			Message: "must be at most 255 characters",
		})
	}
	if request.Status != "" && request.Status != errorcode.TaskStatusTodo && request.Status != errorcode.TaskStatusInProgress && request.Status != errorcode.TaskStatusDone {
		errors = append(errors, phttp.ValidationError{
			Field:   "status",
			Code:    errorcode.InvalidStatus,
			Message: "is invalid",
		})
	}
	return errors
}

func (request AssignRequest) Validate() []phttp.ValidationError {
	if request.AssigneeID != "" && !id.IsValid(request.AssigneeID) {
		return []phttp.ValidationError{{
			Field:   "assignee_id",
			Code:    errorcode.InvalidAssigneeID,
			Message: "must be a UUID",
		}}
	}
	return nil
}

func (request PaginationRequest) Validate() []phttp.ValidationError {
	var errors []phttp.ValidationError
	if request.Page < 1 {
		errors = append(errors, phttp.ValidationError{
			Field:   "page",
			Code:    errorcode.InvalidPagination,
			Message: "must be a positive integer",
		})
	}
	if request.Limit < 1 || request.Limit > 100 {
		errors = append(errors, phttp.ValidationError{
			Field:   "limit",
			Code:    errorcode.InvalidPagination,
			Message: "must be between 1 and 100",
		})
	}
	if request.Status != "" && request.Status != errorcode.TaskStatusTodo && request.Status != errorcode.TaskStatusInProgress && request.Status != errorcode.TaskStatusDone {
		errors = append(errors, phttp.ValidationError{
			Field:   "status",
			Code:    errorcode.InvalidStatus,
			Message: "is invalid",
		})
	}
	if request.TeamID != "" && !id.IsValid(request.TeamID) {
		errors = append(errors, phttp.ValidationError{
			Field:   "team_id",
			Code:    errorcode.InvalidTeamID,
			Message: "must be a UUID",
		})
	}
	return errors
}
