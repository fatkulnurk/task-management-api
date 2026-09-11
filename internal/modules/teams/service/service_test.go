package service

import (
	"context"
	"errors"
	"testing"

	"taskmanagement/internal/modules/teams/domain"
	domainmocks "taskmanagement/internal/modules/teams/domain/mocks"

	"go.uber.org/mock/gomock"
)

var errTest = errors.New("test error")

func newTeamService(t *testing.T) (*teamService, *domainmocks.MockRepository) {
	t.Helper()
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)
	return &teamService{Repository: repository}, repository
}

func TestCreate(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.CreateInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
		check     func(t *testing.T, team domain.Team)
	}{
		{
			name:  "success",
			input: domain.CreateInput{UserID: "user-1", Name: "  Platform Engineering  "},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					DoAndReturn(func(_ context.Context, team domain.Team) error {
						if team.ID == "" {
							t.Error("team id must not be empty")
						}
						if team.OwnerID != "user-1" {
							t.Errorf("owner id = %q, want %q", team.OwnerID, "user-1")
						}
						if team.Name != "Platform Engineering" {
							t.Errorf("name = %q, want %q", team.Name, "Platform Engineering")
						}
						if team.CreatedAt == "" || team.UpdatedAt == "" {
							t.Errorf("timestamps must not be empty: created_at=%q updated_at=%q", team.CreatedAt, team.UpdatedAt)
						}
						return nil
					})
			},
			check: func(t *testing.T, team domain.Team) {
				if team.Name != "Platform Engineering" {
					t.Errorf("name = %q, want %q", team.Name, "Platform Engineering")
				}
				if team.ID == "" {
					t.Error("team id must not be empty")
				}
				if team.CreatedAt == "" || team.UpdatedAt == "" {
					t.Errorf("timestamps must not be empty: created_at=%q updated_at=%q", team.CreatedAt, team.UpdatedAt)
				}
			},
		},
		{
			name:  "repository error",
			input: domain.CreateInput{UserID: "user-1", Name: "Platform Engineering"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Create(gomock.Any(), gomock.Any()).
					Return(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			team, err := teamService.Create(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if test.check != nil {
				test.check(t, team)
			}
		})
	}
}

func TestList(t *testing.T) {
	teams := []domain.Team{{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}}

	tests := []struct {
		name      string
		input     domain.ListInput
		setup     func(repository *domainmocks.MockRepository)
		wantTeams []domain.Team
		wantTotal int
		wantError error
	}{
		{
			name:  "success",
			input: domain.ListInput{UserID: "user-1", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					List(gomock.Any(), "user-1", 1, 20).
					Return(teams, 1, nil)
			},
			wantTeams: teams,
			wantTotal: 1,
		},
		{
			name:  "repository error",
			input: domain.ListInput{UserID: "user-1", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					List(gomock.Any(), "user-1", 1, 20).
					Return(nil, 0, errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			gotTeams, gotTotal, err := teamService.List(context.Background(), test.input)
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
			if err == nil && len(gotTeams) != len(test.wantTeams) {
				t.Errorf("teams = %+v, want %+v", gotTeams, test.wantTeams)
			}
		})
	}
}

func TestGet(t *testing.T) {
	team := domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering"}

	tests := []struct {
		name      string
		input     domain.GetInput
		setup     func(repository *domainmocks.MockRepository)
		wantTeam  domain.Team
		wantError error
	}{
		{
			name:  "success",
			input: domain.GetInput{TeamID: "team-1", UserID: "user-1"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "team-1", "user-1").
					Return(team, nil)
			},
			wantTeam: team,
		},
		{
			name:  "not found",
			input: domain.GetInput{TeamID: "team-1", UserID: "user-1"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Get(gomock.Any(), "team-1", "user-1").
					Return(domain.Team{}, domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			gotTeam, err := teamService.Get(context.Background(), test.input)
			if test.wantError == nil {
				if err != nil {
					t.Fatalf("error = %v, want nil", err)
				}
			} else if !errors.Is(err, test.wantError) {
				t.Fatalf("error = %v, want %v", err, test.wantError)
			}
			if err == nil && gotTeam != test.wantTeam {
				t.Errorf("team = %+v, want %+v", gotTeam, test.wantTeam)
			}
		})
	}
}

func TestMembers(t *testing.T) {
	members := []domain.Member{{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", IsOwner: true}}

	tests := []struct {
		name        string
		input       domain.MembersInput
		setup       func(repository *domainmocks.MockRepository)
		wantMembers []domain.Member
		wantTotal   int
		wantError   error
	}{
		{
			name:  "success",
			input: domain.MembersInput{TeamID: "team-1", UserID: "user-1", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Members(gomock.Any(), "team-1", "user-1", 1, 20).
					Return(members, 1, nil)
			},
			wantMembers: members,
			wantTotal:   1,
		},
		{
			name:  "not found",
			input: domain.MembersInput{TeamID: "team-1", UserID: "user-2", Page: 1, Limit: 20},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					Members(gomock.Any(), "team-1", "user-2", 1, 20).
					Return(nil, 0, domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			gotMembers, gotTotal, err := teamService.Members(context.Background(), test.input)
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
			if err == nil && len(gotMembers) != len(test.wantMembers) {
				t.Errorf("members = %+v, want %+v", gotMembers, test.wantMembers)
			}
		})
	}
}

func TestAdd(t *testing.T) {
	member := domain.Member{ID: "user-2", Name: "Bob", Email: "bob@fatkulnurk.com"}

	tests := []struct {
		name      string
		input     domain.AddInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
	}{
		{
			name:  "success",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					User(gomock.Any(), "", "bob@fatkulnurk.com").
					Return(member, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					Add(gomock.Any(), "team-1", "user-2").
					Return(nil)
			},
		},
		{
			name:  "not owner",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-2", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
			},
			wantError: domain.ErrForbidden,
		},
		{
			name:  "is owner error",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "user error",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "missing@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					User(gomock.Any(), "", "missing@fatkulnurk.com").
					Return(domain.Member{}, domain.ErrNotFound)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:  "is member error",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					User(gomock.Any(), "", "bob@fatkulnurk.com").
					Return(member, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "already member",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					User(gomock.Any(), "", "bob@fatkulnurk.com").
					Return(member, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
			},
			wantError: domain.ErrConflict,
		},
		{
			name:  "add error",
			input: domain.AddInput{TeamID: "team-1", UserID: "user-1", Email: "bob@fatkulnurk.com"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					User(gomock.Any(), "", "bob@fatkulnurk.com").
					Return(member, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					Add(gomock.Any(), "team-1", "user-2").
					Return(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			_, err := teamService.Add(context.Background(), test.input)
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

func TestRemove(t *testing.T) {
	tests := []struct {
		name      string
		input     domain.RemoveInput
		setup     func(repository *domainmocks.MockRepository)
		wantError error
	}{
		{
			name:  "success",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				repository.EXPECT().
					ActiveAssignments(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					Remove(gomock.Any(), "team-1", "user-2").
					Return(nil)
			},
		},
		{
			name:  "not owner",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-2", TargetUserID: "user-3"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
			},
			wantError: domain.ErrForbidden,
		},
		{
			name:  "is owner error",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "target is owner",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-1"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
			},
			wantError: domain.ErrOwner,
		},
		{
			name:  "target is owner check error",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "not member",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-3"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-3").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-3").
					Return(false, nil)
			},
			wantError: domain.ErrNotFound,
		},
		{
			name:  "is member error",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "has active assignments",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				repository.EXPECT().
					ActiveAssignments(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
			},
			wantError: domain.ErrAssignments,
		},
		{
			name:  "active assignments error",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				repository.EXPECT().
					ActiveAssignments(gomock.Any(), "team-1", "user-2").
					Return(false, errTest)
			},
			wantError: errTest,
		},
		{
			name:  "remove error",
			input: domain.RemoveInput{TeamID: "team-1", UserID: "user-1", TargetUserID: "user-2"},
			setup: func(repository *domainmocks.MockRepository) {
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-1").
					Return(true, nil)
				repository.EXPECT().
					IsOwner(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					IsMember(gomock.Any(), "team-1", "user-2").
					Return(true, nil)
				repository.EXPECT().
					ActiveAssignments(gomock.Any(), "team-1", "user-2").
					Return(false, nil)
				repository.EXPECT().
					Remove(gomock.Any(), "team-1", "user-2").
					Return(errTest)
			},
			wantError: errTest,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			teamService, repository := newTeamService(t)
			test.setup(repository)

			err := teamService.Remove(context.Background(), test.input)
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

func TestNewTeamService(t *testing.T) {
	controller := gomock.NewController(t)
	repository := domainmocks.NewMockRepository(controller)

	if NewTeamService(repository) == nil {
		t.Fatal("expected a non-nil service")
	}
}
