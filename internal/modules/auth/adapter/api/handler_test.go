package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskmanagement/internal/modules/auth/domain"
	domainmocks "taskmanagement/internal/modules/auth/domain/mocks"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func newAuthHandler(t *testing.T) (AuthHandler, *domainmocks.MockService) {
	t.Helper()
	controller := gomock.NewController(t)
	service := domainmocks.NewMockService(controller)
	return AuthHandler{AuthService: service}, service
}

func postRequest(path, body string) *http.Request {
	return httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
}

func TestHandlerRegister(t *testing.T) {
	const validBody = `{"name":"Alice","email":"alice@fatkulnurk.com","password":"correct-horse"}`

	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
		wantLeak   string
	}{
		{
			name: "success",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Register(gomock.Any(), domain.RegisterInput{Name: "Alice", Email: "alice@fatkulnurk.com", Password: "correct-horse"}).
					Return(domain.UserOutput{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com"}, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "user-1",
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Register(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			body: `{"name":"","email":"bad","password":""}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Register(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid registration",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Register(gomock.Any(), gomock.Any()).
					Return(domain.UserOutput{}, domain.ErrInvalid)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "conflict",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Register(gomock.Any(), gomock.Any()).
					Return(domain.UserOutput{}, domain.ErrConflict)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "internal error",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Register(gomock.Any(), gomock.Any()).
					Return(domain.UserOutput{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
			wantLeak:   "test error",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newAuthHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.register(recorder, postRequest("/auth/register", test.body))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
			if test.wantLeak != "" && strings.Contains(recorder.Body.String(), test.wantLeak) {
				t.Errorf("internal error detail leaked in response: %s", recorder.Body.String())
			}
		})
	}
}

func TestHandlerLogin(t *testing.T) {
	const validBody = `{"email":"alice@fatkulnurk.com","password":"correct-horse"}`

	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Login(gomock.Any(), domain.LoginInput{Email: "alice@fatkulnurk.com", Password: "correct-horse"}).
					Return(domain.TokenPair{AccessToken: "access", RefreshToken: "refresh", TokenType: "Bearer", ExpiresIn: 900}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "access",
		},
		{
			name: "unauthorized",
			body: `{"email":"alice@fatkulnurk.com","password":"wrong"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(domain.TokenPair{}, domain.ErrUnauthorized)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "validation error",
			body: `{"email":"","password":""}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Login(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Login(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: validBody,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Login(gomock.Any(), gomock.Any()).
					Return(domain.TokenPair{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newAuthHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.login(recorder, postRequest("/auth/login", test.body))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerRefresh(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"refresh_token":"refresh-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Refresh(gomock.Any(), domain.RefreshInput{RefreshToken: "refresh-token"}).
					Return(domain.TokenPair{AccessToken: "new-access", RefreshToken: "new-refresh", TokenType: "Bearer", ExpiresIn: 900}, nil)
			},
			wantStatus: http.StatusOK,
		},
		{
			name: "unauthorized",
			body: `{"refresh_token":"stale-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Refresh(gomock.Any(), gomock.Any()).
					Return(domain.TokenPair{}, domain.ErrUnauthorized)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "validation error",
			body: `{"refresh_token":""}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Refresh(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Refresh(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "internal error",
			body: `{"refresh_token":"refresh-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Refresh(gomock.Any(), gomock.Any()).
					Return(domain.TokenPair{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newAuthHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.refresh(recorder, postRequest("/auth/refresh", test.body))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestHandlerLogout(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
	}{
		{
			name: "success",
			body: `{"refresh_token":"refresh-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Logout(gomock.Any(), domain.LogoutInput{RefreshToken: "refresh-token"}).
					Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "unauthorized",
			body: `{"refresh_token":"stale-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Logout(gomock.Any(), gomock.Any()).
					Return(domain.ErrUnauthorized)
			},
			wantStatus: http.StatusUnauthorized,
		},
		{
			name: "internal error",
			body: `{"refresh_token":"refresh-token"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Logout(gomock.Any(), gomock.Any()).
					Return(errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Logout(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			body: `{"refresh_token":""}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Logout(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newAuthHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.logout(recorder, postRequest("/auth/logout", test.body))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	_, service := newAuthHandler(t)
	router := chi.NewRouter()

	Register(router, service)

	if len(router.Routes()) != 4 {
		t.Fatalf("routes = %d, want 4", len(router.Routes()))
	}
}
