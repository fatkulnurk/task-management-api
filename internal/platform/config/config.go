package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"taskmanagement/internal/platform/database"
	"time"
)

type Config struct {
	Addr, JWTSecret string
	Database        database.Config
	ShutdownSeconds int
}

func Load() (Config, error) {
	maxOpenConns, err := positiveInt("DB_MAX_OPEN_CONNS", 25)
	if err != nil {
		return Config{}, err
	}
	maxIdleConns, err := nonNegativeInt("DB_MAX_IDLE_CONNS", 10)
	if err != nil {
		return Config{}, err
	}
	if maxIdleConns > maxOpenConns {
		return Config{}, errors.New("DB_MAX_IDLE_CONNS must not exceed DB_MAX_OPEN_CONNS")
	}
	connMaxLifetime, err := duration("DB_CONN_MAX_LIFETIME", 30*time.Minute)
	if err != nil {
		return Config{}, err
	}
	connMaxIdleTime, err := duration("DB_CONN_MAX_IDLE_TIME", 5*time.Minute)
	if err != nil {
		return Config{}, err
	}
	pingTimeout, err := duration("DB_PING_TIMEOUT", 5*time.Second)
	if err != nil {
		return Config{}, err
	}
	jwtSecret := os.Getenv("JWT_SECRET")
	if len(jwtSecret) < 32 {
		return Config{}, errors.New("JWT_SECRET must be set and at least 32 characters")
	}

	return Config{
		Addr:      env("APP_ADDR", ":8080"),
		JWTSecret: jwtSecret,
		Database: database.Config{
			DSN:             os.Getenv("DATABASE_URL"),
			MaxOpenConns:    maxOpenConns,
			MaxIdleConns:    maxIdleConns,
			ConnMaxLifetime: connMaxLifetime,
			ConnMaxIdleTime: connMaxIdleTime,
			PingTimeout:     pingTimeout,
		},
		ShutdownSeconds: 10,
	}, nil
}

func env(k, fallback string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return fallback
}

func positiveInt(key string, fallback int) (int, error) {
	value := env(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 1 {
		return 0, fmt.Errorf("%s must be a positive integer", key)
	}
	return parsed, nil
}

func nonNegativeInt(key string, fallback int) (int, error) {
	value := env(key, strconv.Itoa(fallback))
	parsed, err := strconv.Atoi(value)
	if err != nil || parsed < 0 {
		return 0, fmt.Errorf("%s must be a non-negative integer", key)
	}
	return parsed, nil
}

func duration(key string, fallback time.Duration) (time.Duration, error) {
	value := env(key, fallback.String())
	parsed, err := time.ParseDuration(value)
	if err != nil || parsed <= 0 {
		return 0, fmt.Errorf("%s must be a positive duration", key)
	}
	return parsed, nil
}
