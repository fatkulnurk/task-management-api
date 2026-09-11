package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskmanagement/internal/application/authorization"
	"taskmanagement/internal/modules/teams/domain"
	domainmocks "taskmanagement/internal/modules/teams/domain/mocks"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func newTeamHandler(t *testing.T) (TeamHandler, *domainmocks.MockService) {
	t.Helper()
	controller := gomock.NewController(t)
	service := domainmocks.NewMockService(controller)
	return TeamHandler{TeamService: service}, service
}

func teamRequest(method, target, body, userID string, params map[string]string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	routeContext := chi.NewRouteContext()
	for key, value := range params {
		routeContext.URLParams.Add(key, value)
	}
	ctx := context.WithValue(request.Context(), chi.RouteCtxKey, routeContext)
	ctx = authorization.WithIdentity(ctx, authorization.Identity{UserID: userID})
	return request.WithContext(ctx)
}

func TestHandlerCreate(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: `{"name":"Platform Engineering"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Create(gomock.Any(), domain.CreateInput{UserID: "user-1", Name: "Platform Engineering"}).
					Return(domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "team-1",
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			body: `{"name":""}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "service error",
			body: `{"name":"Platform Engineering"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(domain.Team{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.create(recorder, teamRequest(http.MethodPost, "/teams", test.body, "user-1", nil))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerList(t *testing.T) {
	teams := []domain.Team{{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}}

	tests := []struct {
		name       string
		target     string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "success",
			target: "/teams?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					List(gomock.Any(), domain.ListInput{UserID: "user-1", Page: 1, Limit: 20}).
					Return(teams, 1, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "team-1",
		},
		{
			name:   "validation error",
			target: "/teams?page=0&limit=0",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "non numeric pagination",
			target: "/teams?page=abc&limit=xyz",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service error",
			target: "/teams?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, 0, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.list(recorder, teamRequest(http.MethodGet, test.target, "", "user-1", nil))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
			if test.wantStatus == http.StatusOK {
				if !strings.Contains(recorder.Body.String(), `"meta"`) {
					t.Errorf("body = %s, want it to contain meta", recorder.Body.String())
				}
			}
		})
	}
}

func TestHandlerGet(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), domain.GetInput{TeamID: "team-1", UserID: "user-1"}).
					Return(domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "team-1",
		},
		{
			name: "not found",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(domain.Team{}, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(domain.Team{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.get(recorder, teamRequest(http.MethodGet, "/teams/team-1", "", "user-1", map[string]string{"id": "team-1"}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerMembers(t *testing.T) {
	members := []domain.Member{{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", IsOwner: true}}

	tests := []struct {
		name       string
		target     string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "success",
			target: "/teams/team-1/members?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Members(gomock.Any(), domain.MembersInput{TeamID: "team-1", UserID: "user-1", Page: 1, Limit: 20}).
					Return(members, 1, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "alice@fatkulnurk.com",
		},
		{
			name:   "not found",
			target: "/teams/team-1/members?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Members(gomock.Any(), gomock.Any()).
					Return(nil, 0, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "validation error",
			target: "/teams/team-1/members?page=0&limit=0",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Members(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service error",
			target: "/teams/team-1/members?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Members(gomock.Any(), gomock.Any()).
					Return(nil, 0, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.members(recorder, teamRequest(http.MethodGet, test.target, "", "user-1", map[string]string{"id": "team-1"}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
			if test.wantStatus == http.StatusOK && !strings.Contains(recorder.Body.String(), `"meta"`) {
				t.Errorf("body = %s, want it to contain meta", recorder.Body.String())
			}
		})
	}
}

func TestHandlerAdd(t *testing.T) {
	tests := []struct {
		name       string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name: "success",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"}).
					Return(domain.Member{ID: "user-2", Name: "Bob", Email: "bob@fatkulnurk.com"}, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "bob@fatkulnurk.com",
		},
		{
			name: "invalid json",
			body: `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name: "validation error",
			body: `{}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Add(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "forbidden",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(domain.Member{}, domain.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "invalid",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(domain.Member{}, domain.ErrInvalid)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name: "conflict",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(domain.Member{}, domain.ErrConflict)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "not found",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(domain.Member{}, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			body: `{"email":"bob@fatkulnurk.com"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Add(gomock.Any(), gomock.Any()).
					Return(domain.Member{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.add(recorder, teamRequest(http.MethodPost, "/teams/team-1/members", test.body, "user-1", map[string]string{"id": "team-1"}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerRemove(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
	}{
		{
			name: "success",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Remove(gomock.Any(), domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"}).
					Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name: "owner cannot be removed",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Remove(gomock.Any(), gomock.Any()).
					Return(domain.ErrOwner)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name: "member has active assignments",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Remove(gomock.Any(), gomock.Any()).
					Return(domain.ErrAssignments)
			},
			wantStatus: http.StatusConflict,
		},
		{
			name: "not found",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Remove(gomock.Any(), gomock.Any()).
					Return(domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name: "service error",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Remove(gomock.Any(), gomock.Any()).
					Return(errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTeamHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.remove(recorder, teamRequest(http.MethodDelete, "/teams/team-1/members/user-2", "", "user-1", map[string]string{"id": "team-1", "user_id": "user-2"}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	_, service := newTeamHandler(t)
	router := chi.NewRouter()

	Register(router, service, func(next http.Handler) http.Handler { return next })

	if len(router.Routes()) == 0 {
		t.Fatal("expected routes to be registered")
	}
}
