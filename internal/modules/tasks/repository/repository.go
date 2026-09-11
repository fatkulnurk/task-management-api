package repository

import (
	"context"
	"database/sql"
	"errors"
	"taskmanagement/internal/application/errorcode"
	"taskmanagement/internal/modules/tasks/domain"
	"time"

	"github.com/go-sql-driver/mysql"
)

func mysqlDateTime(value string) any {
	parsed, err := time.Parse("2006-01-02T15:04:05Z", value)
	if err != nil {
		return value
	}
	return parsed.Format("2006-01-02 15:04:05")
}

type mySQLTaskRepository struct{ Database *sql.DB }

func NewMySQLTaskRepository(database *sql.DB) domain.Repository {
	return &mySQLTaskRepository{Database: database}
}

const isTeamMemberQuery = "SELECT 1 FROM team_members WHERE team_id=? AND user_id=?"

func (taskRepository *mySQLTaskRepository) Member(ctx context.Context, teamID, userID string) (bool, error) {
	var exists int
	err := taskRepository.Database.QueryRowContext(ctx, isTeamMemberQuery, teamID, userID).Scan(&exists)
	if errors.Is(err, sql.ErrNoRows) {
		return false, nil
	}

	return err == nil, err
}

const (
	deleteIdempotencyQuery = "DELETE FROM idempotency_keys WHERE user_id=? AND idempotency_key=?"
	insertTaskQuery        = "INSERT INTO tasks(id,team_id,creator_id,title,description,status,created_at,updated_at) VALUES(?,?,?,?,?,?,?,?)"
	insertTaskLogQuery     = "INSERT INTO task_logs(task_id,actor_id,action,changes,created_at) VALUES(?,?,?,JSON_OBJECT(),UTC_TIMESTAMP(6))"
	insertIdempotencyQuery = "INSERT INTO idempotency_keys(user_id,idempotency_key,response_status,response_body,expires_at,created_at) VALUES(?,?,?,?,DATE_ADD(UTC_TIMESTAMP(6), INTERVAL 24 HOUR),UTC_TIMESTAMP(6))"
	selectIdempotencyQuery = "SELECT response_status,response_body,expires_at<=UTC_TIMESTAMP(6) FROM idempotency_keys WHERE user_id=? AND idempotency_key=?"
)

const maxCreateAttempts = 5

var errDuplicateKey = errors.New("duplicate idempotency key")

type storedIdempotency struct {
	Status  int
	Body    []byte
	Expired bool
}

func (taskRepository *mySQLTaskRepository) lookupIdempotency(ctx context.Context, userID, key string) (storedIdempotency, bool, error) {
	var stored storedIdempotency
	var expired int
	err := taskRepository.Database.QueryRowContext(ctx, selectIdempotencyQuery, userID, key).Scan(&stored.Status, &stored.Body, &expired)
	if errors.Is(err, sql.ErrNoRows) {
		return stored, false, nil
	}
	if err != nil {
		return stored, false, err
	}
	stored.Expired = expired == 1
	return stored, true, nil
}

func (taskRepository *mySQLTaskRepository) CreateIdempotent(ctx context.Context, task domain.Task, userID, key string, body []byte) (domain.CreateOutput, error) {
	var lastErr error
	for attempt := 0; attempt < maxCreateAttempts; attempt++ {
		result, found, err := taskRepository.replayIdempotent(ctx, userID, key)
		if err != nil {
			return domain.CreateOutput{}, err
		}
		if found {
			return result, nil
		}

		result, err = taskRepository.insertTaskIdempotent(ctx, task, userID, key, body)
		if err == nil {
			return result, nil
		}
		if isRetryableLockError(err) || errors.Is(err, errDuplicateKey) {
			lastErr = err
			continue
		}
		return domain.CreateOutput{}, err
	}
	if lastErr != nil {
		return domain.CreateOutput{}, lastErr
	}
	return domain.CreateOutput{}, errDuplicateKey
}

func (taskRepository *mySQLTaskRepository) replayIdempotent(ctx context.Context, userID, key string) (domain.CreateOutput, bool, error) {
	stored, found, err := taskRepository.lookupIdempotency(ctx, userID, key)
	if err != nil {
		return domain.CreateOutput{}, false, err
	}
	if !found {
		return domain.CreateOutput{}, false, nil
	}
	if stored.Expired {
		if _, err = taskRepository.Database.ExecContext(ctx, deleteIdempotencyQuery, userID, key); err != nil {
			return domain.CreateOutput{}, false, err
		}
		return domain.CreateOutput{}, false, nil
	}
	return domain.CreateOutput{
		Status: stored.Status,
		Body:   stored.Body,
		Replay: true,
	}, true, nil
}

func isRetryableLockError(err error) bool {
	var mysqlErr *mysql.MySQLError
	if !errors.As(err, &mysqlErr) {
		return false
	}
	return mysqlErr.Number == 1213 || mysqlErr.Number == 1205
}

func (taskRepository *mySQLTaskRepository) insertTaskIdempotent(ctx context.Context, task domain.Task, userID, key string, body []byte) (domain.CreateOutput, error) {
	tx, err := taskRepository.Database.BeginTx(ctx, nil)
	if err != nil {
		return domain.CreateOutput{}, err
	}
	defer tx.Rollback()

	if _, err = tx.ExecContext(ctx, insertTaskQuery, task.ID, task.TeamID, task.CreatorID, task.Title, task.Description, task.Status, mysqlDateTime(task.CreatedAt), mysqlDateTime(task.UpdatedAt)); err != nil {
		return domain.CreateOutput{}, err
	}
	if _, err = tx.ExecContext(ctx, insertTaskLogQuery, task.ID, task.CreatorID, errorcode.TaskCreated); err != nil {
		return domain.CreateOutput{}, err
	}
	if _, err = tx.ExecContext(ctx, insertIdempotencyQuery, userID, key, 201, body); err != nil {
		var mysqlErr *mysql.MySQLError
		if errors.As(err, &mysqlErr) && mysqlErr.Number == 1062 {
			return domain.CreateOutput{}, errDuplicateKey
		}
		return domain.CreateOutput{}, err
	}

	if err = tx.Commit(); err != nil {
		return domain.CreateOutput{}, err
	}
	return domain.CreateOutput{
		Status: 201,
		Body:   body,
	}, nil
}

const (
	taskListBaseQuery    = " FROM tasks t JOIN team_members tm ON tm.team_id=t.team_id AND tm.user_id=? WHERE t.deleted_at IS NULL AND (t.creator_id=? OR t.assignee_id=?)"
	taskListTeamFilter   = " AND t.team_id=?"
	taskListStatusFilter = " AND t.status=?"
	taskListSearchFilter = " AND t.title LIKE ?"
	countTasksPrefix     = "SELECT COUNT(*)"
	selectTasksPrefix    = "SELECT t.id,t.team_id,t.creator_id,t.assignee_id,t.title,t.description,t.status,DATE_FORMAT(t.created_at,'%Y-%m-%dT%H:%i:%sZ'),DATE_FORMAT(t.updated_at,'%Y-%m-%dT%H:%i:%sZ')"
	selectTasksSuffix    = " ORDER BY t.created_at DESC LIMIT ? OFFSET ?"
)

func (taskRepository *mySQLTaskRepository) List(ctx context.Context, userID, teamID, taskStatus, searchTerm string, page, limit int) ([]domain.Task, int, error) {
	q := taskListBaseQuery
	args := []any{userID, userID, userID}
	if teamID != "" {
		q += taskListTeamFilter
		args = append(args, teamID)
	}
	if taskStatus != "" {
		q += taskListStatusFilter
		args = append(args, taskStatus)
	}
	if searchTerm != "" {
		q += taskListSearchFilter
		args = append(args, "%"+searchTerm+"%")
	}
	var total int
	if err := taskRepository.Database.QueryRowContext(ctx, countTasksPrefix+q, args...).Scan(&total); err != nil {
		return nil, 0, err
	}

	args = append(args, limit, (page-1)*limit)
	rows, err := taskRepository.Database.QueryContext(ctx, selectTasksPrefix+q+selectTasksSuffix, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()
	tasks := []domain.Task{}

	for rows.Next() {
		var task domain.Task
		if err = rows.Scan(&task.ID, &task.TeamID, &task.CreatorID, &task.AssigneeID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt); err != nil {
			return nil, 0, err
		}
		tasks = append(tasks, task)
	}
	return tasks, total, rows.Err()
}

const selectTaskQuery = "SELECT id,team_id,creator_id,assignee_id,title,description,status,DATE_FORMAT(created_at,'%Y-%m-%dT%H:%i:%sZ'),DATE_FORMAT(updated_at,'%Y-%m-%dT%H:%i:%sZ') FROM tasks WHERE id=? AND deleted_at IS NULL AND (creator_id=? OR assignee_id=?)"

func (taskRepository *mySQLTaskRepository) Get(ctx context.Context, taskID, userID string) (domain.Task, error) {
	var task domain.Task
	err := taskRepository.Database.QueryRowContext(ctx, selectTaskQuery, taskID, userID, userID).Scan(&task.ID, &task.TeamID, &task.CreatorID, &task.AssigneeID, &task.Title, &task.Description, &task.Status, &task.CreatedAt, &task.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return task, domain.ErrNotFound
	}

	return task, err
}

const (
	selectTaskStatusForUpdateQuery = "SELECT status FROM tasks WHERE id=? AND deleted_at IS NULL FOR UPDATE"
	updateTaskQuery                = "UPDATE tasks SET title=?,description=?,status=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND deleted_at IS NULL"
)

func (taskRepository *mySQLTaskRepository) Update(ctx context.Context, task domain.Task, userID string) error {
	tx, err := taskRepository.Database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	defer tx.Rollback()
	var oldStatus string
	err = tx.QueryRowContext(ctx, selectTaskStatusForUpdateQuery, task.ID).Scan(&oldStatus)
	if errors.Is(err, sql.ErrNoRows) {
		err = domain.ErrNotFound
	}
	if err == nil {
		var result sql.Result
		result, err = tx.ExecContext(ctx, updateTaskQuery, task.Title, task.Description, task.Status, task.ID)
		if err == nil {
			if affectedRows, rowsErr := result.RowsAffected(); rowsErr != nil {
				err = rowsErr
			} else if affectedRows == 0 {
				err = domain.ErrNotFound
			}
		}
	}
	if err == nil {
		action := errorcode.TaskUpdated
		if oldStatus != "" && oldStatus != task.Status {
			action = errorcode.TaskStatusChanged
		}
		_, err = tx.ExecContext(ctx, insertTaskLogQuery, task.ID, userID, action)
	}
	if err != nil {
		return err
	}
	return tx.Commit()
}

const deleteTaskQuery = "UPDATE tasks SET deleted_at=UTC_TIMESTAMP(6),updated_at=UTC_TIMESTAMP(6) WHERE id=? AND creator_id=? AND deleted_at IS NULL"

func (taskRepository *mySQLTaskRepository) Delete(ctx context.Context, taskID, userID string) error {
	tx, err := taskRepository.Database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var result sql.Result
	result, err = tx.ExecContext(ctx, deleteTaskQuery, taskID, userID)
	if err == nil {
		if affectedRows, rowsErr := result.RowsAffected(); rowsErr != nil {
			err = rowsErr
		} else if affectedRows == 0 {
			err = domain.ErrNotFound
		}
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, insertTaskLogQuery, taskID, userID, errorcode.TaskDeleted)
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}

const assignTaskQuery = "UPDATE tasks SET assignee_id=?,updated_at=UTC_TIMESTAMP(6) WHERE id=? AND creator_id=? AND deleted_at IS NULL"

func (taskRepository *mySQLTaskRepository) Assign(ctx context.Context, taskID, userID string, assigneeID *string, action string, notify func() error) error {
	tx, err := taskRepository.Database.BeginTx(ctx, nil)
	if err != nil {
		return err
	}
	var result sql.Result
	result, err = tx.ExecContext(ctx, assignTaskQuery, assigneeID, taskID, userID)
	if err == nil {
		if affectedRows, rowsErr := result.RowsAffected(); rowsErr != nil {
			err = rowsErr
		} else if affectedRows == 0 {
			err = domain.ErrNotFound
		}
	}
	if err == nil {
		_, err = tx.ExecContext(ctx, insertTaskLogQuery, taskID, userID, action)
	}
	if err == nil && notify != nil {
		err = notify()
	}
	if err != nil {
		_ = tx.Rollback()
		return err
	}
	return tx.Commit()
}
