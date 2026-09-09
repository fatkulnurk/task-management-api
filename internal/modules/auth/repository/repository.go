package repository

import (
	"context"
	"database/sql"
	"errors"
	"taskmanagement/internal/modules/auth/domain"

	"github.com/go-sql-driver/mysql"
)

type mySQLAuthRepository struct{ Database *sql.DB }

func NewMySQLAuthRepository(database *sql.DB) domain.Repository {
	return &mySQLAuthRepository{
		Database: database,
	}
}

const insertUserQuery = "INSERT INTO users(id,name,email,password_hash,created_at,updated_at) VALUES(?,?,?, ?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))"

func (repository *mySQLAuthRepository) Create(ctx context.Context, user domain.User) error {
	_, err := repository.Database.ExecContext(ctx, insertUserQuery, user.ID, user.Name, user.Email, user.PasswordHash)
	var me *mysql.MySQLError

	if errors.As(err, &me) && me.Number == 1062 {
		return domain.ErrConflict
	}
	return err
}

const selectUserByEmailQuery = "SELECT id,name,email,password_hash FROM users WHERE email=?"

func (repository *mySQLAuthRepository) ByEmail(ctx context.Context, email string) (domain.User, error) {
	var user domain.User

	err := repository.Database.QueryRowContext(ctx, selectUserByEmailQuery, email).Scan(&user.ID, &user.Name, &user.Email, &user.PasswordHash)
	return user, err
}

const insertRefreshTokenQuery = "INSERT INTO refresh_tokens(id,user_id,token_hash,expires_at,created_at) VALUES(?,?,?,DATE_ADD(UTC_TIMESTAMP(6), INTERVAL 30 DAY),UTC_TIMESTAMP(6))"

func (repository *mySQLAuthRepository) CreateRefresh(ctx context.Context, refreshTokenID, userID, tokenHash string) error {
	_, err := repository.Database.ExecContext(ctx, insertRefreshTokenQuery, refreshTokenID, userID, tokenHash)

	return err
}

const (
	selectRefreshTokenQuery = "SELECT user_id FROM refresh_tokens WHERE token_hash=? AND expires_at>UTC_TIMESTAMP(6) FOR UPDATE"
	deleteRefreshTokenQuery = "DELETE FROM refresh_tokens WHERE token_hash=?"
)

func (repository *mySQLAuthRepository) RotateRefresh(ctx context.Context, oldTokenHash, refreshTokenID, userID, newTokenHash string) (string, error) {
	tx, err := repository.Database.BeginTx(ctx, nil)
	if err != nil {
		return "", err
	}
	var uid string
	err = tx.QueryRowContext(ctx, selectRefreshTokenQuery, oldTokenHash).Scan(&uid)
	if err != nil {
		_ = tx.Rollback()
		if errors.Is(err, sql.ErrNoRows) {
			return "", domain.ErrUnauthorized
		}
		return "", err
	}

	if err = uidCheck(uid, userID); err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if _, err = tx.ExecContext(ctx, deleteRefreshTokenQuery, oldTokenHash); err != nil {
		_ = tx.Rollback()
		return "", err
	}
	_, err = tx.ExecContext(ctx, insertRefreshTokenQuery, refreshTokenID, uid, newTokenHash)
	if err != nil {
		_ = tx.Rollback()
		return "", err
	}
	if err = tx.Commit(); err != nil {
		return "", err
	}

	return uid, nil
}

func uidCheck(existingUserID, requestedUserID string) error {
	if requestedUserID == "" {
		return nil
	}

	if existingUserID != requestedUserID {
		return domain.ErrUnauthorized
	}
	return nil
}

func (repository *mySQLAuthRepository) DeleteRefresh(ctx context.Context, tokenHash string) error {
	result, err := repository.Database.ExecContext(ctx, deleteRefreshTokenQuery, tokenHash)
	if err != nil {
		return err
	}

	affectedRows, _ := result.RowsAffected()
	if affectedRows == 0 {
		return domain.ErrUnauthorized
	}
	return nil
}
