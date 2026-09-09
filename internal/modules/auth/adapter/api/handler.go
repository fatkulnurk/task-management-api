package api

import (
	"errors"
	"net/http"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/modules/auth/domain"
	phttp "taskmanagement/internal/platform/http"
)

type AuthHandler struct{ AuthService domain.Service }

func (authHandler AuthHandler) refresh(responseWriter http.ResponseWriter, request *http.Request) {
	var refreshRequest RefreshRequest
	if !phttp.DecodeJSON(responseWriter, request, &refreshRequest) {
		return
	}

	if validationErrors := refreshRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}

	tokenPair, err := authHandler.AuthService.Refresh(request.Context(), domain.RefreshInput{
		RefreshToken: refreshRequest.RefreshToken,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			phttp.JSONResponse4xx(responseWriter, 401, errorcode.Unauthorized, "invalid refresh token")
		} else {
			phttp.JSONResponse5xx(responseWriter, 500)
		}
		return
	}
	phttp.JSONResponse2xx(responseWriter, 200, tokenPair)
}

func (authHandler AuthHandler) register(responseWriter http.ResponseWriter, request *http.Request) {
	var registerRequest RegisterRequest
	if !phttp.DecodeJSON(responseWriter, request, &registerRequest) {
		return
	}

	if validationErrors := registerRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}

	user, err := authHandler.AuthService.Register(request.Context(), domain.RegisterInput{
		Name:     registerRequest.Name,
		Email:    registerRequest.Email,
		Password: registerRequest.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrConflict) {
			phttp.JSONResponse4xx(responseWriter, 409, errorcode.Conflict, "email already registered")
		} else if errors.Is(err, domain.ErrInvalid) {
			phttp.JSONResponse4xx(responseWriter, 422, errorcode.InvalidRequest, "invalid registration")
		} else {
			phttp.JSONResponse5xx(responseWriter, 500)
		}
		return
	}

	phttp.JSONResponse2xx(responseWriter, 201, user)
}

func (authHandler AuthHandler) login(responseWriter http.ResponseWriter, request *http.Request) {
	var loginRequest LoginRequest
	if !phttp.DecodeJSON(responseWriter, request, &loginRequest) {
		return
	}

	if validationErrors := loginRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}

	tokenPair, err := authHandler.AuthService.Login(request.Context(), domain.LoginInput{
		Email:    loginRequest.Email,
		Password: loginRequest.Password,
	})
	if err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			phttp.JSONResponse4xx(responseWriter, 401, errorcode.Unauthorized, "invalid credentials")
		} else {
			phttp.JSONResponse5xx(responseWriter, 500)
		}
		return
	}

	phttp.JSONResponse2xx(responseWriter, 200, tokenPair)
}

func (authHandler AuthHandler) logout(responseWriter http.ResponseWriter, request *http.Request) {
	var logoutRequest LogoutRequest
	if !phttp.DecodeJSON(responseWriter, request, &logoutRequest) {
		return
	}

	if validationErrors := logoutRequest.Validate(); len(validationErrors) > 0 {
		phttp.JSONValidationError(responseWriter, 422, validationErrors)
		return
	}

	if err := authHandler.AuthService.Logout(request.Context(), domain.LogoutInput{
		RefreshToken: logoutRequest.RefreshToken,
	}); err != nil {
		if errors.Is(err, domain.ErrUnauthorized) {
			phttp.JSONResponse4xx(responseWriter, 401, errorcode.Unauthorized, "invalid refresh token")
		} else {
			phttp.JSONResponse5xx(responseWriter, 500)
		}
		return
	}

	phttp.NoContent(responseWriter)
}
