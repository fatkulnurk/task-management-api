package api

import (
	"taskmanagement/internal/application/errorcode"
	phttp "taskmanagement/internal/platform/http"
	"taskmanagement/internal/platform/id"
)

func (request CreateRequest) Validate() []phttp.ValidationError {
	if request.Name == "" || len(request.Name) > 120 {
		return []phttp.ValidationError{{
			Field:   "name",
			Code:    errorcode.InvalidField,
			Message: "is required and must be at most 120 characters",
		}}
	}
	return nil
}

func (request AddRequest) Validate() []phttp.ValidationError {
	var errors []phttp.ValidationError
	if (request.UserID == "") == (request.Email == "") {
		errors = append(errors, phttp.ValidationError{
			Field:   "user_id",
			Code:    errorcode.InvalidField,
			Message: "provide either user_id or email",
		})
	}
	if request.UserID != "" && !id.IsValid(request.UserID) {
		errors = append(errors, phttp.ValidationError{
			Field:   "user_id",
			Code:    errorcode.InvalidID,
			Message: "must be a UUID",
		})
	}
	return errors
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
	return errors
}
