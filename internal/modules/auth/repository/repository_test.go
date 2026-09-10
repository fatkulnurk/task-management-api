package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"taskmanagement/internal/modules/auth/domain"

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
	return NewMySQLAuthRepository(database), mock
}

func TestRepositoryCreate(t *testing.T) {
	user := domain.User{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", PasswordHash: "hash"}

	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertUserQuery).
					WithArgs(user.ID, user.Name, user.Email, user.PasswordHash).
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "duplicate email",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertUserQuery).
					WithArgs(user.ID, user.Name, user.Email, user.PasswordHash).
					WillReturnError(&mysql.MySQLError{Number: 1062, Message: "duplicate entry"})
			},
			wantError: domain.ErrConflict,
		},
		{
			name: "generic error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertUserQuery).
					WithArgs(user.ID, user.Name, user.Email, user.PasswordHash).
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.Create(context.Background(), user)
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

func TestRepositoryByEmail(t *testing.T) {
	tests := []struct {
		name      string
		email     string
		setup     func(mock sqlmock.Sqlmock)
		wantUser  domain.User
		wantError error
	}{
		{
			name:  "success",
			email: "alice@fatkulnurk.com",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectUserByEmailQuery).
					WithArgs("alice@fatkulnurk.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email", "password_hash"}).
						AddRow("user-1", "Alice", "alice@fatkulnurk.com", "hash"))
			},
			wantUser: domain.User{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", PasswordHash: "hash"},
		},
		{
			name:  "not found",
			email: "missing@fatkulnurk.com",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectUserByEmailQuery).
					WithArgs("missing@fatkulnurk.com").
					WillReturnError(sql.ErrNoRows)
			},
			wantError: sql.ErrNoRows,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			user, err := repository.ByEmail(context.Background(), test.email)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && user != test.wantUser {
				t.Errorf("user = %+v, want %+v", user, test.wantUser)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryCreateRefresh(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertRefreshTokenQuery).
					WithArgs("token-1", "user-1", "hash").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertRefreshTokenQuery).
					WithArgs("token-1", "user-1", "hash").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.CreateRefresh(context.Background(), "token-1", "user-1", "hash")
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

func TestRepositoryRotateRefresh(t *testing.T) {
	tests := []struct {
		name       string
		oldHash    string
		userID     string
		setup      func(mock sqlmock.Sqlmock)
		wantUserID string
		wantError  error
	}{
		{
			name:    "success",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertRefreshTokenQuery).
					WithArgs("token-2", "user-1", "new-hash").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
			wantUserID: "user-1",
		},
		{
			name:    "no rows",
			oldHash: "stale-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("stale-hash").
					WillReturnError(sql.ErrNoRows)
				mock.ExpectRollback()
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name:    "user mismatch",
			oldHash: "old-hash",
			userID:  "user-2",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
				mock.ExpectRollback()
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name:    "delete fails",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name:    "insert fails",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertRefreshTokenQuery).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name:    "begin fails",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin().WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name:    "select fails",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name:    "commit fails",
			oldHash: "old-hash",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectQuery(selectRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnRows(sqlmock.NewRows([]string{"user_id"}).AddRow("user-1"))
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("old-hash").
					WillReturnResult(sqlmock.NewResult(0, 1))
				mock.ExpectExec(insertRefreshTokenQuery).
					WithArgs("token-2", "user-1", "new-hash").
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit().WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			userID, err := repository.RotateRefresh(context.Background(), test.oldHash, "token-2", test.userID, "new-hash")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && userID != test.wantUserID {
				t.Errorf("user id = %q, want %q", userID, test.wantUserID)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryDeleteRefresh(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("hash").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("hash").
					WillReturnResult(sqlmock.NewResult(0, 0))
			},
			wantError: domain.ErrUnauthorized,
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(deleteRefreshTokenQuery).
					WithArgs("hash").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.DeleteRefresh(context.Background(), "hash")
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

func TestUidCheck(t *testing.T) {
	tests := []struct {
		name             string
		existingUserID   string
		requestedUserID  string
		wantUnauthorized bool
	}{
		{name: "empty requested", existingUserID: "user-1", requestedUserID: "", wantUnauthorized: false},
		{name: "matching", existingUserID: "user-1", requestedUserID: "user-1", wantUnauthorized: false},
		{name: "mismatch", existingUserID: "user-1", requestedUserID: "user-2", wantUnauthorized: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := uidCheck(test.existingUserID, test.requestedUserID)
			if test.wantUnauthorized && !errors.Is(err, domain.ErrUnauthorized) {
				t.Fatalf("error = %v, want ErrUnauthorized", err)
			}
			if !test.wantUnauthorized && err != nil {
				t.Fatalf("error = %v, want nil", err)
			}
		})
	}
}
