package repository

import (
	"context"
	"database/sql"
	"errors"
	"testing"

	"taskmanagement/internal/modules/teams/domain"

	"github.com/DATA-DOG/go-sqlmock"
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
	return NewMySQLTeamRepository(database), mock
}

func TestRepositoryCreate(t *testing.T) {
	team := domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}

	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(insertTeamQuery).
					WithArgs(team.ID, team.OwnerID, team.Name).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(insertTeamMemberQuery).
					WithArgs(team.ID, team.OwnerID).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectCommit()
			},
		},
		{
			name: "insert team fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(insertTeamQuery).
					WithArgs(team.ID, team.OwnerID, team.Name).
					WillReturnError(errTest)
				mock.ExpectRollback()
			},
			wantError: errTest,
		},
		{
			name: "insert member fails",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectBegin()
				mock.ExpectExec(insertTeamQuery).
					WithArgs(team.ID, team.OwnerID, team.Name).
					WillReturnResult(sqlmock.NewResult(1, 1))
				mock.ExpectExec(insertTeamMemberQuery).
					WithArgs(team.ID, team.OwnerID).
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

			err := repository.Create(context.Background(), team)
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

func TestRepositoryAdd(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertTeamMemberQuery).
					WithArgs("team-1", "user-2").
					WillReturnResult(sqlmock.NewResult(1, 1))
			},
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(insertTeamMemberQuery).
					WithArgs("team-1", "user-2").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.Add(context.Background(), "team-1", "user-2")
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

func TestRepositoryList(t *testing.T) {
	rows := sqlmock.NewRows([]string{"id", "owner_id", "name", "created_at", "updated_at"}).
		AddRow("team-1", "user-1", "Platform Engineering", "2026-09-10T10:00:00Z", "2026-09-10T10:00:00Z").
		AddRow("team-2", "user-1", "Data Platform", "2026-09-09T10:00:00Z", "2026-09-09T10:00:00Z")

	tests := []struct {
		name         string
		setup        func(mock sqlmock.Sqlmock)
		wantTotal    int
		wantCount    int
		wantError    error
		wantAnyError bool
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTeamsQuery).
					WithArgs("user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectTeamsQuery).
					WithArgs("user-1", 20, 0).
					WillReturnRows(rows)
			},
			wantTotal: 2,
			wantCount: 2,
		},
		{
			name: "count error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTeamsQuery).
					WithArgs("user-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "rows error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTeamsQuery).
					WithArgs("user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectTeamsQuery).
					WithArgs("user-1", 20, 0).
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name: "scan error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(countTeamsQuery).
					WithArgs("user-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectTeamsQuery).
					WithArgs("user-1", 20, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("team-1"))
			},
			wantAnyError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			teams, total, err := repository.List(context.Background(), "user-1", 1, 20)
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
			if err == nil && len(teams) != test.wantCount {
				t.Errorf("teams count = %d, want %d", len(teams), test.wantCount)
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
		wantTeam  domain.Team
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectTeamQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"id", "owner_id", "name", "created_at", "updated_at"}).
						AddRow("team-1", "user-1", "Platform Engineering", "2026-09-10T10:00:00Z", "2026-09-10T10:00:00Z"))
			},
			wantTeam: domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering", CreatedAt: "2026-09-10T10:00:00Z", UpdatedAt: "2026-09-10T10:00:00Z"},
		},
		{
			name: "not found",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectTeamQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(sql.ErrNoRows)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			team, err := repository.Get(context.Background(), "team-1", "user-1")
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && team != test.wantTeam {
				t.Errorf("team = %+v, want %+v", team, test.wantTeam)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryMembers(t *testing.T) {
	rows := sqlmock.NewRows([]string{"id", "name", "email", "is_owner"}).
		AddRow("user-1", "Alice", "alice@fatkulnurk.com", true).
		AddRow("user-2", "Bob", "bob@fatkulnurk.com", false)

	tests := []struct {
		name         string
		userID       string
		setup        func(mock sqlmock.Sqlmock)
		wantTotal    int
		wantCount    int
		wantError    error
		wantAnyError bool
	}{
		{
			name:   "success",
			userID: "user-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
				mock.ExpectQuery(countMembersQuery).
					WithArgs("team-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectMembersQuery).
					WithArgs("team-1", 20, 0).
					WillReturnRows(rows)
			},
			wantTotal: 2,
			wantCount: 2,
		},
		{
			name:   "not member",
			userID: "user-9",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-9").
					WillReturnError(sql.ErrNoRows)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:   "is member error",
			userID: "user-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name:   "count error",
			userID: "user-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
				mock.ExpectQuery(countMembersQuery).
					WithArgs("team-1").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name:   "rows error",
			userID: "user-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
				mock.ExpectQuery(countMembersQuery).
					WithArgs("team-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectMembersQuery).
					WithArgs("team-1", 20, 0).
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
		{
			name:   "scan error",
			userID: "user-1",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
				mock.ExpectQuery(countMembersQuery).
					WithArgs("team-1").
					WillReturnRows(sqlmock.NewRows([]string{"count"}).AddRow(2))
				mock.ExpectQuery(selectMembersQuery).
					WithArgs("team-1", 20, 0).
					WillReturnRows(sqlmock.NewRows([]string{"id"}).AddRow("user-1"))
			},
			wantAnyError: true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			members, total, err := repository.Members(context.Background(), "team-1", test.userID, 1, 20)
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
			if err == nil && len(members) != test.wantCount {
				t.Errorf("members count = %d, want %d", len(members), test.wantCount)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryUser(t *testing.T) {
	tests := []struct {
		name       string
		userID     string
		email      string
		setup      func(mock sqlmock.Sqlmock)
		wantMember domain.Member
		wantError  error
	}{
		{
			name:   "by id",
			userID: "user-2",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectUserByIDQuery).
					WithArgs("user-2").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
						AddRow("user-2", "Bob", "bob@fatkulnurk.com"))
			},
			wantMember: domain.Member{ID: "user-2", Name: "Bob", Email: "bob@fatkulnurk.com"},
		},
		{
			name:  "by email",
			email: "bob@fatkulnurk.com",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectUserByEmailQuery).
					WithArgs("bob@fatkulnurk.com").
					WillReturnRows(sqlmock.NewRows([]string{"id", "name", "email"}).
						AddRow("user-2", "Bob", "bob@fatkulnurk.com"))
			},
			wantMember: domain.Member{ID: "user-2", Name: "Bob", Email: "bob@fatkulnurk.com"},
		},
		{
			name:   "not found",
			userID: "user-9",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(selectUserByIDQuery).
					WithArgs("user-9").
					WillReturnError(sql.ErrNoRows)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			member, err := repository.User(context.Background(), test.userID, test.email)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && member != test.wantMember {
				t.Errorf("member = %+v, want %+v", member, test.wantMember)
			}
			if err := mock.ExpectationsWereMet(); err != nil {
				t.Errorf("unmet expectations: %v", err)
			}
		})
	}
}

func TestRepositoryRemove(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		wantError error
	}{
		{
			name: "success",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(deleteTeamMemberQuery).
					WithArgs("team-1", "user-2").
					WillReturnResult(sqlmock.NewResult(0, 1))
			},
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectExec(deleteTeamMemberQuery).
					WithArgs("team-1", "user-2").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			err := repository.Remove(context.Background(), "team-1", "user-2")
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

func TestRepositoryIsMember(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		want      bool
		wantError error
	}{
		{
			name: "member",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
			},
			want: true,
		},
		{
			name: "not member",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(sql.ErrNoRows)
			},
			want: false,
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isMemberQuery).
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

			got, err := repository.IsMember(context.Background(), "team-1", "user-1")
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

func TestRepositoryIsOwner(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		want      bool
		wantError error
	}{
		{
			name: "owner",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isOwnerQuery).
					WithArgs("team-1", "user-1").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
			},
			want: true,
		},
		{
			name: "not owner",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isOwnerQuery).
					WithArgs("team-1", "user-1").
					WillReturnError(sql.ErrNoRows)
			},
			want: false,
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(isOwnerQuery).
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

			got, err := repository.IsOwner(context.Background(), "team-1", "user-1")
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

func TestRepositoryActiveAssignments(t *testing.T) {
	tests := []struct {
		name      string
		setup     func(mock sqlmock.Sqlmock)
		want      bool
		wantError error
	}{
		{
			name: "has active assignments",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(activeAssignmentsQuery).
					WithArgs("team-1", "user-2").
					WillReturnRows(sqlmock.NewRows([]string{"exists"}).AddRow(1))
			},
			want: true,
		},
		{
			name: "no active assignments",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(activeAssignmentsQuery).
					WithArgs("team-1", "user-2").
					WillReturnError(sql.ErrNoRows)
			},
			want: false,
		},
		{
			name: "error",
			setup: func(mock sqlmock.Sqlmock) {
				mock.ExpectQuery(activeAssignmentsQuery).
					WithArgs("team-1", "user-2").
					WillReturnError(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			repository, mock := newRepository(t)
			test.setup(mock)

			got, err := repository.ActiveAssignments(context.Background(), "team-1", "user-2")
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
