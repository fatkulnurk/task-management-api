package service

import (
	"context"
	"strings"
	"taskmanagement/internal/modules/teams/domain"
	"taskmanagement/internal/platform/id"
)

type teamService struct{ Repository domain.Repository }

func NewTeamService(teamRepository domain.Repository) domain.Service {
	return &teamService{
		Repository: teamRepository,
	}
}

func (teamService *teamService) Create(ctx context.Context, input domain.CreateInput) (domain.Team, error) {
	normalizedName := strings.TrimSpace(input.Name)
	team := domain.Team{
		ID:      id.New(),
		OwnerID: input.UserID,
		Name:    normalizedName,
	}
	if err := teamService.Repository.Create(ctx, team); err != nil {
		return team, err
	}

	return team, nil
}

func (teamService *teamService) List(ctx context.Context, input domain.ListInput) ([]domain.Team, int, error) {
	return teamService.Repository.List(ctx, input.UserID, input.Page, input.Limit)
}

func (teamService *teamService) Get(ctx context.Context, input domain.GetInput) (domain.Team, error) {
	return teamService.Repository.Get(ctx, input.TeamID, input.UserID)
}

func (teamService *teamService) Members(ctx context.Context, input domain.MembersInput) ([]domain.Member, int, error) {
	return teamService.Repository.Members(ctx, input.TeamID, input.UserID, input.Page, input.Limit)
}

func (teamService *teamService) Add(ctx context.Context, input domain.AddInput) (domain.Member, error) {
	isOwner, err := teamService.Repository.IsOwner(ctx, input.TeamID, input.UserID)
	if err != nil {
		return domain.Member{}, err
	}

	if !isOwner {
		return domain.Member{}, domain.ErrForbidden
	}

	member, err := teamService.Repository.User(ctx, input.TargetUserID, input.Email)
	if err != nil {
		return member, err
	}

	isMember, err := teamService.Repository.IsMember(ctx, input.TeamID, member.ID)
	if err != nil {
		return member, err
	}

	if isMember {
		return member, domain.ErrConflict
	}

	return member, teamService.Repository.Add(ctx, input.TeamID, member.ID)
}

func (teamService *teamService) Remove(ctx context.Context, input domain.RemoveInput) error {
	isOwner, err := teamService.Repository.IsOwner(ctx, input.TeamID, input.UserID)
	if err != nil {
		return err
	}

	if !isOwner {
		return domain.ErrForbidden
	}

	isTargetOwner, err := teamService.Repository.IsOwner(ctx, input.TeamID, input.TargetUserID)
	if err != nil {
		return err
	}

	if isTargetOwner {
		return domain.ErrOwner
	}

	isMember, err := teamService.Repository.IsMember(ctx, input.TeamID, input.TargetUserID)
	if err != nil {
		return err
	}

	if !isMember {
		return domain.ErrNotFound
	}

	hasActiveAssignments, err := teamService.Repository.ActiveAssignments(ctx, input.TeamID, input.TargetUserID)
	if err != nil {
		return err
	}

	if hasActiveAssignments {
		return domain.ErrAssignments
	}

	return teamService.Repository.Remove(ctx, input.TeamID, input.TargetUserID)
}
