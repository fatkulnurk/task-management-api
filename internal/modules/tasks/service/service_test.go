package service

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"

	"taskmanagement/internal/application/notification"
	notificationmocks "taskmanagement/internal/application/notification/mocks"
	"taskmanagement/internal/modules/tasks/domain"
	domainmocks "taskmanagement/internal/modules/tasks/domain/mocks"

	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func pointer(value string) *string {
	return &value
}

func newTaskService(t *testing.T) (*taskService, *domainmocks.MockRepository, *notificationmocks.MockNotificationService) {
	t.Helper()
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)
	notificationService := notificationmocks.NewMockNotificationService(controller)
	return &taskService{Repository: repository, NotificationService: notificationService}, repository, notificationService
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.CreateInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
		check     func(t *testing.T, output domain.CreateOutput)
	}{
		{
			name: "success",
			input: domain.CreateInput{
				UserID:         "user-1",
				IdempotencyKey: "0123456789abcdef0123456789abcdef",
				Task:           domain.Task{TeamID: "team-1", Title: "  Prepare report  ", Description: "  Weekly  ", Status: "todo"},
			},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					CreateIdempotent(gomock.Any(), gomock.Any(), "user-1", "0123456789abcdef0123456789abcdef", gomock.Any()).
					DoAndReturn(func(_ context.Context, task domain.Task, _ string, _ string, body []byte) (domain.CreateOutput, error) {
						if task.CreatorID != "user-1" {
							t.Errorf("creator id = %q, want %q", task.CreatorID, "user-1")
						}
						if task.ID == "" {
							t.Error("task id must not be empty")
						}
						if task.Title != "Prepare report" || task.Description != "Weekly" {
							t.Errorf("task not normalized: %+v", task)
						}
						if task.CreatedAt == "" || task.UpdatedAt == "" {
							t.Error("task timestamps must not be empty")
						}
						if task.CreatedAt != task.UpdatedAt {
							t.Errorf("created at %q != updated at %q", task.CreatedAt, task.UpdatedAt)
						}
						return domain.CreateOutput{Status: 201, Body: body}, nil
					})
			},
			check: func(t *testing.T, output domain.CreateOutput) {
				if output.Status != 201 {
					t.Errorf("status = %d, want 201", output.Status)
				}
				if output.Replay {
					t.Error("first create must not be a replay")
				}
			},
		},
		{
			name: "not member",
			input: domain.CreateInput{
				UserID:         "user-1",
				IdempotencyKey: "0123456789abcdef0123456789abcdef",
				Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Status: "todo"},
			},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-1").
					Return(false, nil)
			},
			wantError: domain.ErrForbidden,
		},
		{
			name: "member error",
			input: domain.CreateInput{
				UserID:         "user-1",
				IdempotencyKey: "0123456789abcdef0123456789abcdef",
				Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Status: "todo"},
			},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-1").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name: "create idempotent error",
			input: domain.CreateInput{
				UserID:         "user-1",
				IdempotencyKey: "0123456789abcdef0123456789abcdef",
				Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Status: "todo"},
			},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					CreateIdempotent(gomock.Any(), gomock.Any(), "user-1", "0123456789abcdef0123456789abcdef", gomock.Any()).
					Return(domain.CreateOutput{}, errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, _ := newTaskService(t)
			test.setup(repository)

			output, err := taskService.Create(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, output)
			}
		})
	}
}

func TestList(t *testing.T) {
	tasks := []domain.Task{{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}}

	tests := []struct {
		name      string
		input     domain.ListInput
		setup     func(repository *domainmocks.MockRepository)
		wantTotal int
		wantCount int
		wantError error
	}{
		{
			name:  "success",
			input: domain.ListInput{UserID: "user-1", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					List(gomock.Any(), "user-1", "", "", "", 1, 20).
					Return(tasks, 1, nil)
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name:  "repository error",
			input: domain.ListInput{UserID: "user-1", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					List(gomock.Any(), "user-1", "", "", "", 1, 20).
					Return(nil, 0, errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, _ := newTaskService(t)
			test.setup(repository)

			gotTasks, gotTotal, err := taskService.List(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && gotTotal != test.wantTotal {
				t.Errorf("total = %d, want %d", gotTotal, test.wantTotal)
			}
			if err == nil && len(gotTasks) != test.wantCount {
				t.Errorf("tasks count = %d, want %d", len(gotTasks), test.wantCount)
			}
		})
	}
}

func TestGet(t *testing.T) {
	task := domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}

	tests := []struct {
		name      string
		setup     func(repository *domainmocks.MockRepository)
		wantTask  domain.Task
		wantError error
	}{
		{
			name: "success",
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
			},
			wantTask: task,
		},
		{
			name: "not found",
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{}, domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, _ := newTaskService(t)
			test.setup(repository)

			gotTask, err := taskService.Get(context.Background(), domain.GetInput{TaskID: "task-1", UserID: "user-1"})
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && gotTask != test.wantTask {
				t.Errorf("task = %+v, want %+v", gotTask, test.wantTask)
			}
		})
	}
}

func TestUpdate(t *testing.T) {
	existing := domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Description: "Weekly", Status: "todo"}

	tests := []struct {
		name      string
		input     domain.UpdateInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
		check     func(t *testing.T, task domain.Task)
	}{
		{
			name:  "creator updates title",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-1", Task: domain.Task{Title: "Prepare report v2"}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(existing, nil)
				repository.EXPECT().
					Update(gomock.Any(), gomock.Any(), "user-1").
					DoAndReturn(func(_ context.Context, task domain.Task, _ string) error {
						if task.Title != "Prepare report v2" {
							t.Errorf("title = %q, want %q", task.Title, "Prepare report v2")
						}
						if task.Description != "Weekly" || task.Status != "todo" {
							t.Errorf("defaults not filled: %+v", task)
						}
						return nil
					})
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report v2", Description: "Weekly", Status: "todo", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-02T00:00:00Z"}, nil)
			},
			check: func(t *testing.T, task domain.Task) {
				if task.Title != "Prepare report v2" {
					t.Errorf("title = %q, want %q", task.Title, "Prepare report v2")
				}
				if task.CreatedAt != "2026-01-01T00:00:00Z" || task.UpdatedAt != "2026-01-02T00:00:00Z" {
					t.Errorf("timestamps not from stored task: %+v", task)
				}
			},
		},
		{
			name:  "non creator cannot change title",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-2", Task: domain.Task{Title: "Hijacked"}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-2").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}, nil)
			},
			wantError: domain.ErrForbidden,
		},
		{
			name:  "assignee can change status",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-2", Task: domain.Task{Status: "done"}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-2").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Description: "Weekly", Status: "todo"}, nil)
				repository.EXPECT().
					Update(gomock.Any(), gomock.Any(), "user-2").
					DoAndReturn(func(_ context.Context, task domain.Task, _ string) error {
						if task.Status != "done" {
							t.Errorf("status = %q, want %q", task.Status, "done")
						}
						if task.Title != "Prepare report" {
							t.Errorf("title = %q, want %q", task.Title, "Prepare report")
						}
						return nil
					})
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-2").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Description: "Weekly", Status: "done", CreatedAt: "2026-01-01T00:00:00Z", UpdatedAt: "2026-01-02T00:00:00Z"}, nil)
			},
			check: func(t *testing.T, task domain.Task) {
				if task.Status != "done" {
					t.Errorf("status = %q, want %q", task.Status, "done")
				}
				if task.CreatedAt != "2026-01-01T00:00:00Z" || task.UpdatedAt != "2026-01-02T00:00:00Z" {
					t.Errorf("timestamps not from stored task: %+v", task)
				}
			},
		},
		{
			name:  "request assignee does not override stored assignee",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-1", Task: domain.Task{Title: "Prepare report v2", AssigneeID: pointer("intruder")}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(existing, nil)
				repository.EXPECT().
					Update(gomock.Any(), gomock.Any(), "user-1").
					Return(nil)
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report v2", Description: "Weekly", Status: "todo"}, nil)
			},
			check: func(t *testing.T, task domain.Task) {
				if task.AssigneeID != nil {
					t.Errorf("assignee id = %v, want nil from stored task", *task.AssigneeID)
				}
			},
		},
		{
			name:  "get error",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-1", Task: domain.Task{Title: "Prepare report v2"}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{}, domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:  "update error",
			input: domain.UpdateInput{TaskID: "task-1", UserID: "user-1", Task: domain.Task{Title: "Prepare report v2"}},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(existing, nil)
				repository.EXPECT().
					Update(gomock.Any(), gomock.Any(), "user-1").
					Return(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, _ := newTaskService(t)
			test.setup(repository)

			task, err := taskService.Update(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, task)
			}
		})
	}
}

func TestDelete(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(repository *domainmocks.MockRepository)
		wantError error
	}{
		{
			name: "success",
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Delete(gomock.Any(), "task-1", "user-1").
					Return(nil)
			},
		},
		{
			name: "error",
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Delete(gomock.Any(), "task-1", "user-1").
					Return(domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, _ := newTaskService(t)
			test.setup(repository)

			err := taskService.Delete(context.Background(), domain.DeleteInput{TaskID: "task-1", UserID: "user-1"})
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
		})
	}
}

func TestAssign(t *testing.T) {
	task := domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Status: "todo"}

	tests := []struct {
		name      string
		input     domain.AssignInput
		setup     func(repository *domainmocks.MockRepository, notificationService *notificationmocks.MockNotificationService)
		wantError error
		check     func(t *testing.T, task domain.Task)
	}{
		{
			name:  "assign success",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository, notificationService *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				notificationService.EXPECT().
					Send(gomock.Any(), notification.Notification{UserID: "user-2", TaskID: "task-1", Message: "task assigned"}).
					Return(nil)
				repository.EXPECT().
					Assign(gomock.Any(), "task-1", "user-1", gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ string, assigneeID *string, action string, notify func() error) error {
						if assigneeID == nil || *assigneeID != "user-2" {
							t.Errorf("assignee id = %v, want user-2", assigneeID)
						}
						if action != "task_assigned" {
							t.Errorf("action = %q, want %q", action, "task_assigned")
						}
						if notify == nil {
							t.Fatal("notify must not be nil")
						}
						return notify()
					})
			},
			check: func(t *testing.T, task domain.Task) {
				if task.AssigneeID == nil || *task.AssigneeID != "user-2" {
					t.Errorf("assignee id = %v, want user-2", task.AssigneeID)
				}
			},
		},
		{
			name:  "unassign success",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: ""},
			setup: func(repository *domainmocks.MockRepository, notificationService *notificationmocks.MockNotificationService) {
				assigned := "user-2"
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", AssigneeID: &assigned, Title: "Prepare report", Status: "todo"}, nil)
				notificationService.EXPECT().
					Send(gomock.Any(), notification.Notification{UserID: "user-2", TaskID: "task-1", Message: "task unassigned"}).
					Return(nil)
				repository.EXPECT().
					Assign(gomock.Any(), "task-1", "user-1", nil, "task_unassigned", gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ string, assigneeID *string, _ string, notify func() error) error {
						if assigneeID != nil {
							t.Errorf("assignee id = %v, want nil", assigneeID)
						}
						return notify()
					})
			},
			check: func(t *testing.T, task domain.Task) {
				if task.AssigneeID != nil {
					t.Errorf("assignee id = %v, want nil", task.AssigneeID)
				}
			},
		},
		{
			name:  "not creator",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-2", TargetUserID: "user-3"},
			setup: func(repository *domainmocks.MockRepository, _ *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-2").
					Return(domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1"}, nil)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:  "get error",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository, _ *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(domain.Task{}, errTest)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:  "target not member",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-9"},
			setup: func(repository *domainmocks.MockRepository, _ *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-9").
					Return(false, nil)
			},
			wantError: domain.ErrForbidden,
		},
		{
			name:  "member error",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository, _ *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-2").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "assign error",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository, _ *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				repository.EXPECT().
					Assign(gomock.Any(), "task-1", "user-1", gomock.Any(), gomock.Any(), gomock.Any()).
					Return(errTest)
			},
			wantError: errTest,
		},
		{
			name:  "notification error",
			input: domain.AssignInput{TaskID: "task-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository, notificationService *notificationmocks.MockNotificationService) {
				repository.EXPECT().
					Get(gomock.Any(), "task-1", "user-1").
					Return(task, nil)
				repository.EXPECT().
					Member(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				notificationService.EXPECT().
					Send(gomock.Any(), gomock.Any()).
					Return(errTest)
				repository.EXPECT().
					Assign(gomock.Any(), "task-1", "user-1", gomock.Any(), gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, _ string, _ string, _ *string, _ string, notify func() error) error {
						return notify()
					})
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			taskService, repository, notificationService := newTaskService(t)
			test.setup(repository, notificationService)

			task, err := taskService.Assign(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, task)
			}
		})
	}
}

type fakeIdempotencyRecord struct {
	status int
	body   []byte
}

type fakeRepository struct {
	mutex   sync.Mutex
	tasks   int
	records map[string]fakeIdempotencyRecord
}

func newFakeRepository() *fakeRepository {
	return &fakeRepository{records: map[string]fakeIdempotencyRecord{}}
}

func (repository *fakeRepository) Member(context.Context, string, string) (bool, error) {
	return true, nil
}

func (repository *fakeRepository) CreateIdempotent(_ context.Context, task domain.Task, userID, key string, body []byte) (domain.CreateOutput, error) {
	repository.mutex.Lock()
	defer repository.mutex.Unlock()
	recordKey := userID + "|" + key
	if record, ok := repository.records[recordKey]; ok {
		return domain.CreateOutput{Status: record.status, Body: record.body, Replay: true}, nil
	}
	repository.tasks++
	repository.records[recordKey] = fakeIdempotencyRecord{status: 201, body: body}
	return domain.CreateOutput{Status: 201, Body: body}, nil
}

func (repository *fakeRepository) List(context.Context, string, string, string, string, int, int) ([]domain.Task, int, error) {
	return nil, 0, nil
}

func (repository *fakeRepository) Get(context.Context, string, string) (domain.Task, error) {
	return domain.Task{}, nil
}

func (repository *fakeRepository) Update(context.Context, domain.Task, string) error {
	return nil
}

func (repository *fakeRepository) Delete(context.Context, string, string) error {
	return nil
}

func (repository *fakeRepository) Assign(context.Context, string, string, *string, string, func() error) error {
	return nil
}

func TestCreateSequentialIdempotency(t *testing.T) {
	repository := newFakeRepository()
	taskService := &taskService{Repository: repository}
	input := domain.CreateInput{
		UserID:         "user-1",
		IdempotencyKey: "0123456789abcdef0123456789abcdef",
		Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Description: "Weekly", Status: "todo"},
	}

	first, err := taskService.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}
	second, err := taskService.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}

	if first.Status != 201 || first.Replay {
		t.Errorf("first = %+v, want status 201 and replay false", first)
	}
	if second.Status != 201 || !second.Replay {
		t.Errorf("second = %+v, want status 201 and replay true", second)
	}
	if !bytes.Equal(first.Body, second.Body) {
		t.Errorf("bodies differ: %s != %s", first.Body, second.Body)
	}
	if repository.tasks != 1 {
		t.Errorf("tasks = %d, want 1", repository.tasks)
	}
}

func TestCreateConcurrentIdempotency(t *testing.T) {
	repository := newFakeRepository()
	taskService := &taskService{Repository: repository}
	const goroutines = 32
	input := domain.CreateInput{
		UserID:         "user-1",
		IdempotencyKey: "0123456789abcdef0123456789abcdef",
		Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Description: "Weekly", Status: "todo"},
	}

	results := make([]domain.CreateOutput, goroutines)
	errs := make([]error, goroutines)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for index := 0; index < goroutines; index++ {
		waitGroup.Add(1)
		go func(goroutineIndex int) {
			defer waitGroup.Done()
			<-start
			results[goroutineIndex], errs[goroutineIndex] = taskService.Create(context.Background(), input)
		}(index)
	}
	close(start)
	waitGroup.Wait()

	created := 0
	var body []byte
	for index := range results {
		if errs[index] != nil {
			t.Fatalf("goroutine %d error: %v", index, errs[index])
		}
		if results[index].Status != 201 {
			t.Errorf("goroutine %d status = %d, want 201", index, results[index].Status)
		}
		if !results[index].Replay {
			created++
			body = results[index].Body
		}
	}
	if created != 1 {
		t.Errorf("created count = %d, want 1", created)
	}
	if repository.tasks != 1 {
		t.Errorf("tasks = %d, want 1", repository.tasks)
	}
	for index := range results {
		if !bytes.Equal(results[index].Body, body) {
			t.Errorf("goroutine %d body = %s, want %s", index, results[index].Body, body)
		}
	}
}

func TestCreateSameKeyDifferentPayloadReplaysFirst(t *testing.T) {
	repository := newFakeRepository()
	taskService := &taskService{Repository: repository}

	first := domain.CreateInput{
		UserID:         "user-1",
		IdempotencyKey: "0123456789abcdef0123456789abcdef",
		Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Description: "Weekly", Status: "todo"},
	}
	second := domain.CreateInput{
		UserID:         "user-1",
		IdempotencyKey: "0123456789abcdef0123456789abcdef",
		Task:           domain.Task{TeamID: "team-1", Title: "Different title", Description: "Weekly", Status: "todo"},
	}

	firstOutput, err := taskService.Create(context.Background(), first)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}
	secondOutput, err := taskService.Create(context.Background(), second)
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}
	if !secondOutput.Replay {
		t.Error("second create must be a replay")
	}
	if !bytes.Equal(firstOutput.Body, secondOutput.Body) {
		t.Errorf("bodies differ: %s != %s", firstOutput.Body, secondOutput.Body)
	}
	if repository.tasks != 1 {
		t.Errorf("tasks = %d, want 1", repository.tasks)
	}
}

func TestCreateResponseBodyMatchesTask(t *testing.T) {
	repository := newFakeRepository()
	taskService := &taskService{Repository: repository}
	input := domain.CreateInput{
		UserID:         "user-1",
		IdempotencyKey: "0123456789abcdef0123456789abcdef",
		Task:           domain.Task{TeamID: "team-1", Title: "Prepare report", Description: "Weekly", Status: "todo"},
	}

	output, err := taskService.Create(context.Background(), input)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var decoded domain.Task
	if err := json.Unmarshal(output.Body, &decoded); err != nil {
		t.Fatalf("failed to decode response body: %v", err)
	}
	if decoded.CreatorID != "user-1" {
		t.Errorf("creator id = %q, want %q", decoded.CreatorID, "user-1")
	}
	if decoded.Title != "Prepare report" || decoded.Status != "todo" {
		t.Errorf("task = %+v, want the created task", decoded)
	}
}

func TestNewTaskService(t *testing.T) {
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)
	notificationService := notificationmocks.NewMockNotificationService(controller)

	if NewTaskService(repository, notificationService) == nil {
		t.Fatal("expected a non-nil service")
	}
}
