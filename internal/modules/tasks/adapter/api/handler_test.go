package api

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"taskmanagement/internal/application/authorization"
	"taskmanagement/internal/modules/tasks/domain"
	domainmocks "taskmanagement/internal/modules/tasks/domain/mocks"

	"github.com/go-chi/chi/v5"
	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

const (
	validTaskID         = "0123456789abcdef0123456789abcdef"
	validIdempotencyKey = "fedcba9876543210fedcba9876543210"
)

func newTaskHandler(t *testing.T) (TaskHandler, *domainmocks.MockService) {
	t.Helper()
	controller := gomock.NewController(t)
	service := domainmocks.NewMockService(controller)
	return TaskHandler{TaskService: service}, service
}

func taskRequest(method, target, body, userID, idempotencyKey string, params map[string]string) *http.Request {
	request := httptest.NewRequest(method, target, strings.NewReader(body))
	if idempotencyKey != "" {
		request.Header.Set("Idempotency-Key", idempotencyKey)
	}
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
		name           string
		body           string
		idempotencyKey string
		setup          func(service *domainmocks.MockService)
		wantStatus     int
		wantBody       string
	}{
		{
			name:           "success",
			body:           `{"team_id":"team-1","title":"Prepare report","status":"todo"}`,
			idempotencyKey: validIdempotencyKey,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Create(gomock.Any(), domain.CreateInput{
						UserID:         "user-1",
						IdempotencyKey: validIdempotencyKey,
						Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Status: "todo"},
					}).
					Return(domain.CreateOutput{Status: http.StatusCreated, Body: []byte(`{"id":"task-1"}`)}, nil)
			},
			wantStatus: http.StatusCreated,
			wantBody:   "task-1",
		},
		{
			name:           "missing idempotency key",
			body:           `{"team_id":"team-1","title":"Prepare report","status":"todo"}`,
			idempotencyKey: "",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid idempotency key",
			body:           `{"team_id":"team-1","title":"Prepare report","status":"todo"}`,
			idempotencyKey: "not-a-uuid",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:           "invalid json",
			body:           `{`,
			idempotencyKey: validIdempotencyKey,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:           "validation error",
			body:           `{"team_id":"","title":"","status":""}`,
			idempotencyKey: validIdempotencyKey,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Create(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:           "forbidden",
			body:           `{"team_id":"team-1","title":"Prepare report","status":"todo"}`,
			idempotencyKey: validIdempotencyKey,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(domain.CreateOutput{}, domain.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:           "service error",
			body:           `{"team_id":"team-1","title":"Prepare report","status":"todo"}`,
			idempotencyKey: validIdempotencyKey,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(domain.CreateOutput{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.create(recorder, taskRequest(http.MethodPost, "/tasks", test.body, "user-1", test.idempotencyKey, nil))

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
	tasks := []domain.Task{{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}}

	tests := []struct {
		name       string
		target     string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "success",
			target: "/tasks?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					List(gomock.Any(), domain.ListInput{UserID: "user-1", Page: 1, Limit: 20}).
					Return(tasks, 1, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "task-1",
		},
		{
			name:   "with filters",
			target: "/tasks?page=1&limit=20&status=todo&team_id=" + validTaskID + "&search=report",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					List(gomock.Any(), domain.ListInput{UserID: "user-1", TeamID: validTaskID, Status: "todo", Search: "report", Page: 1, Limit: 20}).
					Return(tasks, 1, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "task-1",
		},
		{
			name:   "validation error",
			target: "/tasks?page=0&limit=0",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "non numeric pagination",
			target: "/tasks?page=abc&limit=xyz",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().List(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "client error",
			target: "/tasks?page=1&limit=20",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					List(gomock.Any(), gomock.Any()).
					Return(nil, 0, domain.ErrInvalid)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "service error",
			target: "/tasks?page=1&limit=20",
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
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.list(recorder, taskRequest(http.MethodGet, test.target, "", "user-1", "", nil))

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

func TestHandlerGet(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "invalid id",
			taskID: "not-a-uuid",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Get(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "success",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), domain.GetInput{TaskID: validTaskID, UserID: "user-1"}).
					Return(domain.Task{ID: validTaskID, TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   validTaskID,
		},
		{
			name:   "not found",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "service error",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Get(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.get(recorder, taskRequest(http.MethodGet, "/tasks/"+test.taskID, "", "user-1", "", map[string]string{"id": test.taskID}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerUpdate(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "invalid id",
			taskID: "not-a-uuid",
			body:   `{"title":"Prepare report v2"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "invalid json",
			taskID: validTaskID,
			body:   `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error",
			taskID: validTaskID,
			body:   `{"status":"blocked"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Update(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "success",
			taskID: validTaskID,
			body:   `{"title":"Prepare report v2"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), domain.UpdateInput{TaskID: validTaskID, UserID: "user-1", Task: domain.Task{Title: "Prepare report v2"}}).
					Return(domain.Task{ID: validTaskID, Title: "Prepare report v2", Status: "todo", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-02T00:00:00Z"}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   `"created_at":"2026-01-01T00:00:00Z"`,
		},
		{
			name:   "assignee id in body is ignored",
			taskID: validTaskID,
			body:   `{"title":"Prepare report v2","assignee_id":"22222222-2222-4222-8222-222222222222"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), domain.UpdateInput{TaskID: validTaskID, UserID: "user-1", Task: domain.Task{Title: "Prepare report v2"}}).
					Return(domain.Task{ID: validTaskID, Title: "Prepare report v2", Status: "todo"}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   "Prepare report v2",
		},
		{
			name:   "forbidden",
			taskID: validTaskID,
			body:   `{"title":"Hijacked"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "invalid",
			taskID: validTaskID,
			body:   `{"title":"Prepare report v2"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrInvalid)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "not found",
			taskID: validTaskID,
			body:   `{"title":"Prepare report v2"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "service error",
			taskID: validTaskID,
			body:   `{"title":"Prepare report v2"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Update(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.update(recorder, taskRequest(http.MethodPut, "/tasks/"+test.taskID, test.body, "user-1", "", map[string]string{"id": test.taskID}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerDelete(t *testing.T) {
	tests := []struct {
		name       string
		taskID     string
		setup      func(service *domainmocks.MockService)
		wantStatus int
	}{
		{
			name:   "invalid id",
			taskID: "not-a-uuid",
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Delete(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "success",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Delete(gomock.Any(), domain.DeleteInput{TaskID: validTaskID, UserID: "user-1"}).
					Return(nil)
			},
			wantStatus: http.StatusNoContent,
		},
		{
			name:   "not found",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "service error",
			taskID: validTaskID,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Delete(gomock.Any(), gomock.Any()).
					Return(errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.delete(recorder, taskRequest(http.MethodDelete, "/tasks/"+test.taskID, "", "user-1", "", map[string]string{"id": test.taskID}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
		})
	}
}

func TestHandlerAssign(t *testing.T) {
	assignee := validIdempotencyKey

	tests := []struct {
		name       string
		taskID     string
		body       string
		setup      func(service *domainmocks.MockService)
		wantStatus int
		wantBody   string
	}{
		{
			name:   "invalid id",
			taskID: "not-a-uuid",
			body:   `{"assignee_id":"` + assignee + `"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Assign(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "invalid json",
			taskID: validTaskID,
			body:   `{`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Assign(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusBadRequest,
		},
		{
			name:   "validation error",
			taskID: validTaskID,
			body:   `{"assignee_id":"not-a-uuid"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().Assign(gomock.Any(), gomock.Any()).Times(0)
			},
			wantStatus: http.StatusUnprocessableEntity,
		},
		{
			name:   "success",
			taskID: validTaskID,
			body:   `{"assignee_id":"` + assignee + `"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Assign(gomock.Any(), domain.AssignInput{TaskID: validTaskID, UserID: "user-1", TargetUserID: assignee}).
					Return(domain.Task{ID: validTaskID, AssigneeID: &assignee, Title: "Prepare report"}, nil)
			},
			wantStatus: http.StatusOK,
			wantBody:   assignee,
		},
		{
			name:   "forbidden",
			taskID: validTaskID,
			body:   `{"assignee_id":"` + assignee + `"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Assign(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrForbidden)
			},
			wantStatus: http.StatusForbidden,
		},
		{
			name:   "not found",
			taskID: validTaskID,
			body:   `{"assignee_id":"` + assignee + `"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Assign(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, domain.ErrNotFound)
			},
			wantStatus: http.StatusNotFound,
		},
		{
			name:   "service error",
			taskID: validTaskID,
			body:   `{"assignee_id":"` + assignee + `"}`,
			setup: func(service *domainmocks.MockService) {
				service.EXPECT().
					Assign(gomock.Any(), gomock.Any()).
					Return(domain.Task{}, errTest)
			},
			wantStatus: http.StatusInternalServerError,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			test.setup(service)

			recorder := httptest.NewRecorder()
			handler.assign(recorder, taskRequest(http.MethodPost, "/tasks/"+test.taskID+"/assign", test.body, "user-1", "", map[string]string{"id": test.taskID}))

			if recorder.Code != test.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantStatus)
			}
			if test.wantBody != "" && !strings.Contains(recorder.Body.String(), test.wantBody) {
				t.Errorf("body = %s, want it to contain %q", recorder.Body.String(), test.wantBody)
			}
		})
	}
}

func TestHandlerErrorCode(t *testing.T) {
	tests := []struct {
		name      string
		err       error
		wantCode  string
		wantState int
	}{
		{name: "forbidden", err: domain.ErrForbidden, wantCode: "forbidden", wantState: http.StatusForbidden},
		{name: "not found", err: domain.ErrNotFound, wantCode: "not_found", wantState: http.StatusNotFound},
		{name: "invalid", err: domain.ErrInvalid, wantCode: "invalid_request", wantState: http.StatusUnprocessableEntity},
		{name: "unknown", err: errTest, wantCode: "internal_error", wantState: http.StatusInternalServerError},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			handler, service := newTaskHandler(t)
			service.EXPECT().
				Assign(gomock.Any(), gomock.Any()).
				Return(domain.Task{}, test.err)

			recorder := httptest.NewRecorder()
			handler.assign(recorder, taskRequest(http.MethodPost, "/tasks/"+validTaskID+"/assign", `{"assignee_id":"abcdefabcdefabcdefabcdefabcdefab"}`, "user-1", "", map[string]string{"id": validTaskID}))

			if recorder.Code != test.wantState {
				t.Fatalf("status = %d, want %d", recorder.Code, test.wantState)
			}
			var envelope struct {
				Code string `json:"code"`
			}
			if err := json.Unmarshal(recorder.Body.Bytes(), &envelope); err != nil {
				t.Fatalf("failed to decode error envelope: %v", err)
			}
			if envelope.Code != test.wantCode {
				t.Errorf("code = %q, want %q", envelope.Code, test.wantCode)
			}
		})
	}
}

func TestRegisterRoutes(t *testing.T) {
	_, service := newTaskHandler(t)
	router := chi.NewRouter()

	Register(router, service, func(next http.Handler) http.Handler { return next })

	if len(router.Routes()) == 0 {
		t.Fatal("expected routes to be registered")
	}
}
