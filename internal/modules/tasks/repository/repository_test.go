package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"taskmanagement/internal/modules/tasks/domain"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/go-sql-driver/mysql"
)

var errTest = errors.New("test error")

func newRepository(t *testing.T) (domain.Repository, sqlmock.Sqlmock) {
	t.Helper()
	database, mock, err := sqlmock.New(sqlmock.QueryMatcherOption(sqlmock.QueryMatcherEqual))
	if err != nil {
		t.Fatalf("failed to create sqlmock: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})
	return NewMySQLTaskRepository(database), mock
}

func TestMySQLDateTime(t *testing.T) {
	if got := mysqlDateTime("2026-09-10T10:00:00Z"); got != "2026-09-10 10:00:00" {
		t.Errorf("got %v, want 2026-09-10 10:00:00", got)
	}
	if got := mysqlDateTime(""); got != "" {
		t.Errorf("got %v, want empty string", got)
	}
	if got := mysqlDateTime("not-a-date"); got != "not-a-date" {
		t.Errorf("got %v, want the input unchanged", got)
	}
}

func TestRepositoryMember(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		want      bool
		wantError error
	}{
		{
			name: "member",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isTeamMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
			},
			want: true,
		},
		{
			name: "not member",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isTeamMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(sql.ErrNoRows)
			},
			want: false,
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isTeamMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			got, err := repository.Member(context.Background(), "team-1", "user-1")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && got != test.want {
				t.Errorf("got = %v, want %v", got, test.want)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryCreateIdempotent(t *testing.T) {
	task := domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Description: "Weekly", Status: "todo", CreatedAt: "2026-09-10T10:00:00Z", UpdatedAt: "2026-09-10T10:00:00Z"}
	body := []byte(`{"id":"task-1"}`)

	lookupNoRows := func(mock sqlmock.Sqlmock) {
		mock.ExpectQuery(selectIdempotencyQuery).
			WithArgs("user-1", "key-1").
			WillReturnError(sql.ErrNoRows)
	}
	lookupStored := func(mock sqlmock.Sqlmock, expired int) {
		mock.ExpectQuery(selectIdempotencyQuery).
			WithArgs("user-1", "key-1").
			WillReturnRows(sqlmock.NewRows([]string{"response_status", "response_body", "expired"}).
				AddRow(201, body, expired))
	}
	insertTaskAndLog := func(mock sqlmock.Sqlmock) {
		mock.ExpectExec(insertTaskQuery).
			WithArgs(task.ID, task.TeamID, task.CreatorID, task.Title, task.Description, task.Status, "2026-09-10 10:00:00", "2026-09-10 10:00:00").
			WillReturnResult(sqlmock.NewResult(1, 1))
		mock.ExpectExec(insertTaskLogQuery).
			WithArgs(task.ID, task.CreatorID, "task_created").
			WillReturnResult(sqlmock.NewResult(1, 1))
	}
	insertIdempotency := func(mock sqlmock.Sqlmock) {
		mock.ExpectExec(insertIdempotencyQuery).
			WithArgs("user-1", "key-1", 201, body).
			WillReturnResult(sqlmock.NewResult(1, 1))
	}

	tests := []struct {
		name         string
		key          string
		setup        func(mock sqlmock.Sqlmock)
		wantStatus   int
		wantReplay   bool
		wantError    error
		wantAnyError bool
	}{
		{
			name: "success",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				insertIdempotency(mock)
				mock.ExpectCommit()
			},
			wantStatus: 201,
		},
		{
			name: "replay existing",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupStored(mock, 0)
			},
			wantStatus: 201,
			wantReplay: true,
		},
		{
			name: "expired key",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupStored(mock, 1)
				mock.ExpectExec(deleteIdempotencyQuery).
					WithArgs("user-1", "key-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				insertIdempotency(mock)
				mock.ExpectCommit()
			},
			wantStatus: 201,
		},
		{
			name: "lookup fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectIdempotencyQuery).
					WithArgs("user-1", "key-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "delete expired fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupStored(mock, 1)
				mock.ExpectExec(deleteIdempotencyQuery).
					WithArgs("user-1", "key-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "begin fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "insert task fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				mock.ExpectExec(insertTaskQuery).
					WithArgs(task.ID, task.TeamID, task.CreatorID, task.Title, task.Description, task.Status, "2026-09-10 10:00:00", "2026-09-10 10:00:00").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "insert log fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				mock.ExpectExec(insertTaskQuery).
					WithArgs(task.ID, task.TeamID, task.CreatorID, task.Title, task.Description, task.Status, "2026-09-10 10:00:00", "2026-09-10 10:00:00").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs(task.ID, task.CreatorID, "task_created").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "insert idempotency fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				mock.ExpectExec(insertIdempotencyQuery).
					WithArgs("user-1", "key-1", 201, body).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "duplicate on insert idempotency",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				mock.ExpectExec(insertIdempotencyQuery).
					WithArgs("user-1", "key-1", 201, body).
					WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate entry"})
				mock.ExpectRollback()
				lookupStored(mock, 0)
			},
			wantStatus: 201,
			wantReplay: true,
		},
		{
			name: "commit fails",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				insertIdempotency(mock)
				mock.ExpectCommit().WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "deadlock then replay",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				mock.ExpectExec(insertIdempotencyQuery).
					WithArgs("user-1", "key-1", 201, body).
					WillReturnError(&mysql.MySQLError{Number: 1213, Message: "deadlock"})
				mock.ExpectRollback()
				lookupStored(mock, 0)
			},
			wantStatus: 201,
			wantReplay: true,
		},
		{
			name: "lock wait timeout then retry succeeds",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				mock.ExpectExec(insertIdempotencyQuery).
					WithArgs("user-1", "key-1", 201, body).
					WillReturnError(&mysql.MySQLError{Number: 1205, Message: "lock wait timeout"})
				mock.ExpectRollback()
				lookupNoRows(mock)
				mock.ExpectBegin()
				insertTaskAndLog(mock)
				insertIdempotency(mock)
				mock.ExpectCommit()
			},
			wantStatus: 201,
		},
		{
			name: "duplicate errors exhausted",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				for attempt := 0; attempt < maxCreateAttempts; attempt++ {
					lookupNoRows(mock)
					mock.ExpectBegin()
					insertTaskAndLog(mock)
					mock.ExpectExec(insertIdempotencyQuery).
						WithArgs("user-1", "key-1", 201, body).
						WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate entry"})
					mock.ExpectRollback()
				}
			},
			wantError: errDuplicateKey,
		},
		{
			name: "retryable errors exhausted",
			key:  "key-1",
			setup: func(mock sqlmock.Sqlmock) {
				for attempt := 0; attempt < maxCreateAttempts; attempt++ {
					lookupNoRows(mock)
					mock.ExpectBegin()
					mock.ExpectExec(insertTaskQuery).
						WithArgs(task.ID, task.TeamID, task.CreatorID, task.Title, task.Description, task.Status, "2026-09-10 10:00:00", "2026-09-10 10:00:00").
						WillReturnError(&mysql.MySQLError{Number: 1213, Message: "deadlock"})
					mock.ExpectRollback()
				}
			},
			wantAnyError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			output, err := repository.CreateIdempotent(context.Background(), task, "user-1", test.key, body)
			if test.wantAnyError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
			} else if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil {
				if output.Status != test.wantStatus {
					t.Errorf("status = %d, want %d", output.Status, test.wantStatus)
				}
				if output.Replay != test.wantReplay {
					t.Errorf("replay = %v, want %v", output.Replay, test.wantReplay)
				}
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryCreateIdempotentReplayReturnsStoredBody(t *testing.T) {
	repository, mock := newRepository(t)
	storedBody := []byte(`{"id":"task-original"}`)
	mock.ExpectQuery(selectIdempotencyQuery).
		WithArgs("user-1", "key-1").
		WillReturnRows(sqlmock.NewRows([]string{"response_status", "response_body", "expired"}).
			AddRow(201, storedBody, 0))

	task := domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Different title", Status: "todo"}
	output, err := repository.CreateIdempotent(context.Background(), task, "user-1", "key-1", []byte(`{"id":"a new body"}`))
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if !output.Replay {
		t.Error("expected a replay")
	}
	if string(output.Body) != string(storedBody) {
		t.Errorf("body = %s, want the stored body %s", output.Body, storedBody)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Errorf("unmet expectations: %v", err)
	}
}

func TestRepositoryList(t *testing.T) {
	newTaskRows := func() *sqlmock.Rows {
		return sqlmock.NewRows([]string{"id", "team_id", "creator_id", "assignee_id", "title", "description", "status", "created_at", "updated_at"}).
			AddRow("task-1", "team-1", "user-1", nil, "Prepare report", "Weekly", "todo", "2026-09-10T10:00:00Z", "2026-09-10T10:00:00Z")
	}

	tests := []struct {
		name         string
		teamID       string
		status       string
		search       string
		setup        func(mock sqlmock.Sqlmock)
		wantTotal    int
		wantCount    int
		wantError    error
		wantAnyError bool
	}{
		{
			name: "no filter",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTasksPrefix+taskListBaseQuery).
					WithArgs("user-1", "user-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(selectTasksPrefix+taskListBaseQuery+selectTasksSuffix).
					WithArgs("user-1", "user-1", "user-1", 20, 0).
					WillReturnRows(newTaskRows())
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name:   "team and status and search filter",
			teamID: "team-1",
			status: "todo",
			search: "report",
			setup: func(mock sqlmock.Sqlmock) {
				query := taskListBaseQuery + taskListTeamFilter + taskListStatusFilter + taskListSearchFilter
				mock.ExpectQuery(countTasksPrefix+query).
					WithArgs("user-1", "user-1", "user-1", "team-1", "todo", "%report%").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(selectTasksPrefix+query+selectTasksSuffix).
					WithArgs("user-1", "user-1", "user-1", "team-1", "todo", "%report%", 20, 0).
					WillReturnRows(newTaskRows())
			},
			wantTotal: 1,
			wantCount: 1,
		},
		{
			name: "count error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTasksPrefix+taskListBaseQuery).
					WithArgs("user-1", "user-1", "user-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "rows error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTasksPrefix+taskListBaseQuery).
					WithArgs("user-1", "user-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(selectTasksPrefix+taskListBaseQuery+selectTasksSuffix).
					WithArgs("user-1", "user-1", "user-1", 20, 0).
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "scan error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTasksPrefix+taskListBaseQuery).
					WithArgs("user-1", "user-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(1))
				mock.ExpectQuery(selectTasksPrefix+taskListBaseQuery+selectTasksSuffix).
					WithArgs("user-1", "user-1", "user-1", 20, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("task-1"))
			},
			wantAnyError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			tasks, total, err := repository.List(context.Background(), "user-1", test.teamID, test.status, test.search, 1, 20)
			if test.wantAnyError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
			} else if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && total != test.wantTotal {
				t.Errorf("total = %d, want %d", total, test.wantTotal)
			}
			if err == nil && len(tasks) != test.wantCount {
				t.Errorf("tasks count = %d, want %d", len(tasks), test.wantCount)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryGet(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantTask  domain.Task
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectTaskQuery).
					WithArgs("task-1", "user-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"id", "team_id", "creator_id", "assignee_id", "title", "description", "status", "created_at", "updated_at"}).
						AddRow("task-1", "team-1", "user-1", nil, "Prepare report", "Weekly", "todo", "2026-09-10T10:00:00Z", "2026-09-10T10:00:00Z"))
			},
			wantTask: domain.Task{ID: "task-1", TeamID: "team-1", CreatorID: "user-1", Title: "Prepare report", Description: "Weekly", Status: "todo", CreatedAt: "2026-09-10T10:00:00Z", UpdatedAt: "2026-09-10T10:00:00Z"},
		},
		{
			name: "not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectTaskQuery).
					WithArgs("task-1", "user-1", "user-1").
					WillReturnError(sql.ErrNoRows)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			task, err := repository.Get(context.Background(), "task-1", "user-1")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && task != test.wantTask {
				t.Errorf("task = %+v, want %+v", task, test.wantTask)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryUpdate(t *testing.T) {
	task := domain.Task{ID: "task-1", Title: "Prepare report v2", Description: "Weekly", Status: "done"}

	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "status changed",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectTaskStatusForUpdateQuery).
					WithArgs("task-1").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))
				mock.ExpectExec(updateTaskQuery).
					WithArgs(task.Title, task.Description, task.Status, task.ID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs(task.ID, "user-1", "task_status_changed").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectTaskStatusForUpdateQuery).
					WithArgs("task-1").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantError: domain.ErrNotFound,
		},
		{
			name: "no rows affected",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectTaskStatusForUpdateQuery).
					WithArgs("task-1").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))
				mock.ExpectExec(updateTaskQuery).
					WithArgs(task.Title, task.Description, task.Status, task.ID).
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			wantError: domain.ErrNotFound,
		},
		{
			name: "rows affected error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectTaskStatusForUpdateQuery).
					WithArgs("task-1").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("todo"))
				mock.ExpectExec(updateTaskQuery).
					WithArgs(task.Title, task.Description, task.Status, task.ID).
					WillReturnResult(sqlmock.NewErrorResult(errTest))
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "log insert fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectTaskStatusForUpdateQuery).
					WithArgs("task-1").
					WillReturnRows(sqlmock.NewRows([]string{"status"}).AddRow("done"))
				mock.ExpectExec(updateTaskQuery).
					WithArgs(task.Title, task.Description, task.Status, task.ID).
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs(task.ID, "user-1", "task_updated").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "begin fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.Update(context.Background(), task, "user-1")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryDelete(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(deleteTaskQuery).
					WithArgs("task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs("task-1", "user-1", "task_deleted").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(deleteTaskQuery).
					WithArgs("task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			wantError: domain.ErrNotFound,
		},
		{
			name: "rows affected error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(deleteTaskQuery).
					WithArgs("task-1", "user-1").
					WillReturnResult(sqlmock.NewErrorResult(errTest))
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "log insert fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(deleteTaskQuery).
					WithArgs("task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs("task-1", "user-1", "task_deleted").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "begin fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.Delete(context.Background(), "task-1", "user-1")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryAssign(t *testing.T) {
	assignee := "user-2"

	tests := []struct {
		name       string
		assigneeID *string
		action     string
		notify     func() error
		setup      func(mock sqlmock.Sqlmock)
		wantNotify bool
		wantError  error
	}{
		{
			name:       "assign success",
			assigneeID: &assignee,
			action:     "task_assigned",
			notify: func() error {
				return nil
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignTaskQuery).
					WithArgs(&assignee, "task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs("task-1", "user-1", "task_assigned").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantNotify: true,
		},
		{
			name:       "unassign success",
			assigneeID: nil,
			action:     "task_unassigned",
			notify: func() error {
				return nil
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignTaskQuery).
					WithArgs(nil, "task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs("task-1", "user-1", "task_unassigned").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantNotify: true,
		},
		{
			name:       "not found",
			assigneeID: &assignee,
			action:     "task_assigned",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignTaskQuery).
					WithArgs(&assignee, "task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 0))
				mock.ExpectRollback()
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:       "rows affected error",
			assigneeID: &assignee,
			action:     "task_assigned",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignTaskQuery).
					WithArgs(&assignee, "task-1", "user-1").
					WillReturnResult(sqlmock.NewErrorResult(errTest))
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name:       "notify error rolls back",
			assigneeID: &assignee,
			action:     "task_assigned",
			notify: func() error {
				return errTest
			},
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(assignTaskQuery).
					WithArgs(&assignee, "task-1", "user-1").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertTaskLogQuery).
					WithArgs("task-1", "user-1", "task_assigned").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectRollback()
			},
			wantNotify: true,
			wantError:  errTest,
		},
		{
			name:       "begin fails",
			assigneeID: &assignee,
			action:     "task_assigned",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			notified := false
			notify := test.notify
			if notify != nil {
				original := notify
				notify = func() error {
					notified = true
					return original()
				}
			}

			err := repository.Assign(context.Background(), "task-1", "user-1", test.assigneeID, test.action, notify)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if notified != test.wantNotify {
				t.Errorf("notified = %v, want %v", notified, test.wantNotify)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}
