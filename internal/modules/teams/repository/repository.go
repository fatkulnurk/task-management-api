package repository

import (
	"context"
	"database/sql"
	"errors"
	"taskmanagement/internal/modules/teams/domain"
	"time"
)

func mysqlDateTime(value string) any {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05")
}

type mySQLTeamRepository struct{ Database *sql.DB }

func NewMySQLTeamRepository(database *sql.DB) domain.Repository {
	return &mySQLTeamRepository{Database: database}
}

const (
	insertTeamQuery       = "INSERT INTO teams(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,?,?)"
	insertTeamMemberQuery = "INSERT INTO team_members(team_id,user_id) VALUES(?,?)"
)

func (teamRepository *mySQLTeamRepository) Create(ctx context.Context, team domain.Team) error {
	tx, err := teamRepository.Database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	if _, err = tx.ExecContext(ctx, insertTeamQuery, team.ID, team.OwnerID, team.Name, mysqlDateTime(team.CreatedAt), mysqlDateTime(team.UpdatedAt)); err == nil {
		_, err = tx.ExecContext(ctx, insertTeamMemberQuery, team.ID, team.OwnerID)
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}

	return tx.Commit()
}

func (teamRepository *mySQLTeamRepository) Add(ctx context.Context, teamID, userID string) error {
	_, err := teamRepository.Database.ExecContext(ctx, insertTeamMemberQuery, teamID, userID)
	return err
}

const (
	countTeamsQuery  = "SELECT COUNT(*) FROM teams t JOIN team_members m ON m.team_id=t.id WHERE m.user_id=?"
	selectTeamsQuery = "SELECT t.id,t.owner_id,t.name,DATE_FORMAT(t.created_at,'%Y-%m-%dT%H:%i:%sZ'),DATE_FORMAT(t.updated_at,'%Y-%m-%dT%H:%i:%sZ') FROM teams t JOIN team_members m ON m.team_id=t.id WHERE m.user_id=? ORDER BY t.created_at DESC LIMIT ? OFFSET ?"
)

func (teamRepository *mySQLTeamRepository) List(ctx context.Context, userID string, page, limit int) ([]domain.Team, int, error) {
	var total int
	err := teamRepository.Database.QueryRowContext(ctx, countTeamsQuery, userID).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	rows, err := teamRepository.Database.QueryContext(ctx, selectTeamsQuery, userID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Team{}

	for rows.Next() {
		var team domain.Team
		if err = rows.Scan(&team.ID, &team.OwnerID, &team.Name, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, 0, err
		}
		out = append(out, team)
	}
	return out, total, rows.Err()
}

const selectTeamQuery = "SELECT t.id,t.owner_id,t.name,DATE_FORMAT(t.created_at,'%Y-%m-%dT%H:%i:%sZ'),DATE_FORMAT(t.updated_at,'%Y-%m-%dT%H:%i:%sZ') FROM teams t JOIN team_members m ON m.team_id=t.id WHERE t.id=? AND m.user_id=?"

func (teamRepository *mySQLTeamRepository) Get(ctx context.Context, teamID, userID string) (domain.Team, error) {
	var team domain.Team
	err := teamRepository.Database.QueryRowContext(ctx, selectTeamQuery, teamID, userID).Scan(&team.ID, &team.OwnerID, &team.Name, &team.CreatedAt, &team.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return team, domain.ErrNotFound
	}
	return team, err
}

const (
	countMembersQuery  = "SELECT COUNT(*) FROM team_members WHERE team_id=?"
	selectMembersQuery = "SELECT u.id,u.name,u.email,(u.id=x.owner_id) FROM users u JOIN team_members m ON m.user_id=u.id JOIN teams x ON x.id=m.team_id WHERE m.team_id=? ORDER BY u.name LIMIT ? OFFSET ?"
)

func (teamRepository *mySQLTeamRepository) Members(ctx context.Context, teamID, userID string, page, limit int) ([]domain.Member, int, error) {
	ok, err := teamRepository.IsMember(ctx, teamID, userID)
	if err != nil {
		return nil, 0, err
	}

	if !ok {
		return nil, 0, domain.ErrNotFound
	}
	var total int
	if err = teamRepository.Database.QueryRowContext(ctx, countMembersQuery, teamID).Scan(&total); err != nil {
		return nil, 0, err
	}

	rows, err := teamRepository.Database.QueryContext(ctx, selectMembersQuery, teamID, limit, (page-1)*limit)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	out := []domain.Member{}

	for rows.Next() {
		var m domain.Member
		if err = rows.Scan(&m.ID, &m.Name, &m.Email, &m.IsOwner); err != nil {
			return nil, 0, err
		}
		out = append(out, m)
	}
	return out, total, rows.Err()
}

const (
	selectUserByIDQuery    = "SELECT id,name,email FROM users WHERE id=?"
	selectUserByEmailQuery = "SELECT id,name,email FROM users WHERE email=?"
)

func (teamRepository *mySQLTeamRepository) User(ctx context.Context, userID, email string) (domain.Member, error) {
	var m domain.Member
	var err error
	if userID != "" {
		err = teamRepository.Database.QueryRowContext(ctx, selectUserByIDQuery, userID).Scan(&m.ID, &m.Name, &m.Email)
	} else {
		err = teamRepository.Database.QueryRowContext(ctx, selectUserByEmailQuery, email).Scan(&m.ID, &m.Name, &m.Email)
	}

	if errors.Is(err, sql.ErrNoRows) {
		return m, domain.ErrNotFound
	}
	return m, err
}

const deleteTeamMemberQuery = "DELETE FROM team_members WHERE team_id=? AND user_id=?"

func (teamRepository *mySQLTeamRepository) Remove(ctx context.Context, teamID, userID string) error {
	_, err := teamRepository.Database.ExecContext(ctx, deleteTeamMemberQuery, teamID, userID)
	return err
}

const isMemberQuery = "SELECT 1 FROM team_members WHERE team_id=? AND user_id=?"

func (teamRepository *mySQLTeamRepository) IsMember(ctx context.Context, teamID, userID string) (bool, error) {
	var exists int
	err := teamRepository.Database.QueryRowContext(ctx, isMemberQuery, teamID, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

const isOwnerQuery = "SELECT 1 FROM teams WHERE id=? AND owner_id=?"

func (teamRepository *mySQLTeamRepository) IsOwner(ctx context.Context, teamID, userID string) (bool, error) {
	var exists int
	err := teamRepository.Database.QueryRowContext(ctx, isOwnerQuery, teamID, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

const activeAssignmentsQuery = "SELECT 1 FROM tasks WHERE team_id=? AND assignee_id=? AND deleted_at IS NULL LIMIT 1"

func (teamRepository *mySQLTeamRepository) ActiveAssignments(ctx context.Context, teamID, userID string) (bool, error) {
	var exists int
	err := teamRepository.Database.QueryRowContext(ctx, activeAssignmentsQuery, teamID, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}
