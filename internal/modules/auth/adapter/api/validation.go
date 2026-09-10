package api

import (
	"net/mail"
	"strings"
	"taskmanagement/internal/application/errorcode"
	phttp "taskmanagement/internal/platform/http"
	"unicode"
)

const (
	maxNameLength         = 120
	maxEmailLength        = 255
	minPasswordLength     = 8
	maxPasswordBytes      = 72
	maxRefreshTokenLength = 128
)

func (request RegisterRequest) Validate() []phttp.ValidationError {
	var validationErrors []phttp.ValidationError
	name := strings.TrimSpace(request.Name)
	email := strings.ToLower(strings.TrimSpace(request.Email))

	if name == "" {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "name",
			Code:    errorcode.InvalidField,
			Message: "is required",
		})
	} else if len([]rune(name)) > maxNameLength {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "name",
			Code:    errorcode.InvalidField,
			Message: "must be at most 120 characters",
		})
	} else if containsControl(name) {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "name",
			Code:    errorcode.InvalidField,
			Message: "must not contain control characters",
		})
	}

	if email == "" {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "email",
			Code:    errorcode.InvalidField,
			Message: "is required",
		})
	} else if len([]rune(email)) > maxEmailLength || !validEmail(email) {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "email",
			Code:    errorcode.InvalidEmail,
			Message: "must be a valid email address",
		})
	}

	validationErrors = append(validationErrors, validatePassword(request.Password)...)
	return validationErrors
}

func (request LoginRequest) Validate() []phttp.ValidationError {
	var validationErrors []phttp.ValidationError
	email := strings.TrimSpace(request.Email)
	if email == "" {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "email",
			Code:    errorcode.InvalidField,
			Message: "is required",
		})
	} else if len([]rune(email)) > maxEmailLength || !validEmail(email) {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "email",
			Code:    errorcode.InvalidEmail,
			Message: "must be a valid email address",
		})
	}

	if strings.TrimSpace(request.Password) == "" {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "password",
			Code:    errorcode.InvalidPassword,
			Message: "is required",
		})
	} else if len([]byte(request.Password)) > maxPasswordBytes {
		validationErrors = append(validationErrors, phttp.ValidationError{
			Field:   "password",
			Code:    errorcode.InvalidPassword,
			Message: "must be at most 72 bytes",
		})
	}
	return validationErrors
}

func (request RefreshRequest) Validate() []phttp.ValidationError {
	if invalidToken(request.RefreshToken) {
		return []phttp.ValidationError{{
			Field:   "refresh_token",
			Code:    errorcode.InvalidField,
			Message: "is required and must be a valid token",
		}}
	}
	return nil
}

func (request LogoutRequest) Validate() []phttp.ValidationError {
	if invalidToken(request.RefreshToken) {
		return []phttp.ValidationError{{
			Field:   "refresh_token",
			Code:    errorcode.InvalidField,
			Message: "is required and must be a valid token",
		}}
	}
	return nil
}

func validatePassword(password string) []phttp.ValidationError {
	if password == "" {
		return []phttp.ValidationError{{
			Field:   "password",
			Code:    errorcode.InvalidPassword,
			Message: "is required",
		}}
	}
	if len([]rune(password)) < minPasswordLength {
		return []phttp.ValidationError{{
			Field:   "password",
			Code:    errorcode.InvalidPassword,
			Message: "must be at least 8 characters",
		}}
	}
	if len([]byte(password)) > maxPasswordBytes {
		return []phttp.ValidationError{{
			Field:   "password",
			Code:    errorcode.InvalidPassword,
			Message: "must be at most 72 bytes",
		}}
	}
	return nil
}

func validEmail(email string) bool {
	normalizedEmail := strings.ToLower(strings.TrimSpace(email))
	if len([]rune(normalizedEmail)) > maxEmailLength || strings.IndexFunc(normalizedEmail, unicode.IsSpace) >= 0 || containsControl(normalizedEmail) {
		return false
	}
	address, err := mail.ParseAddress(normalizedEmail)
	return err == nil && address.Address == normalizedEmail
}

func invalidToken(token string) bool {
	trimmed := strings.TrimSpace(token)
	return trimmed == "" || len([]rune(trimmed)) > maxRefreshTokenLength || strings.IndexFunc(trimmed, unicode.IsSpace) >= 0 || containsControl(trimmed)
}

func containsControl(value string) bool {
	return strings.IndexFunc(value, unicode.IsControl) >= 0
}
