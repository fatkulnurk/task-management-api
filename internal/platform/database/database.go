package database

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	_ "github.com/go-sql-driver/mysql"
	"time"
)

type Config struct {
	DSN             string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	ConnMaxIdleTime time.Duration
	PingTimeout     time.Duration
}

func Open(config Config) (*sql.DB, error) {
	if config.DSN == "" {
		return nil, errors.New("database DSN is required")
	}
	if config.MaxOpenConns < 1 {
		return nil, errors.New("database max open connections must be positive")
	}
	if config.MaxIdleConns < 0 || config.MaxIdleConns > config.MaxOpenConns {
		return nil, errors.New("database max idle connections must be between zero and max open connections")
	}
	if config.ConnMaxLifetime <= 0 {
		return nil, errors.New("database connection max lifetime must be positive")
	}
	if config.ConnMaxIdleTime <= 0 {
		return nil, errors.New("database connection max idle time must be positive")
	}
	if config.PingTimeout <= 0 {
		return nil, errors.New("database ping timeout must be positive")
	}

	database, err := sql.Open("mysql", config.DSN)
	if err != nil {
		return nil, fmt.Errorf("open database: %w", err)
	}
	database.SetMaxOpenConns(config.MaxOpenConns)
	database.SetMaxIdleConns(config.MaxIdleConns)
	database.SetConnMaxLifetime(config.ConnMaxLifetime)
	database.SetConnMaxIdleTime(config.ConnMaxIdleTime)

	ctx, cancel := context.WithTimeout(context.Background(), config.PingTimeout)
	defer cancel()
	if err = database.PingContext(ctx); err != nil {
		_ = database.Close()
		return nil, fmt.Errorf("ping database: %w", err)
	}
	return database, nil
}
