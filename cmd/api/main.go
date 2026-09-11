package main

import (
	"context"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"taskmanagement/internal/modules/auth"
	"taskmanagement/internal/modules/tasks"
	"taskmanagement/internal/modules/teams"
	"taskmanagement/internal/platform/config"
	"taskmanagement/internal/platform/database"
	apphttp "taskmanagement/internal/platform/http"
	"taskmanagement/internal/platform/logger"
	platformnotification "taskmanagement/internal/platform/notification"
	jwtservice "taskmanagement/internal/platform/token/jwt"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/joho/godotenv"
)

func main() {
	slog.SetDefault(logger.New(os.Stdout))

	if err := godotenv.Load(); err != nil {
		if errors.Is(err, os.ErrNotExist) {
			slog.Info("dotenv file not found; using environment variables")
		} else {
			slog.Error("dotenv", "error", err)
			os.Exit(1)
		}
	}

	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(1)
	}

	databaseInstance, err := database.Open(cfg.Database)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer databaseInstance.Close()

	loggerInstance := slog.Default()
	tokenService := jwtservice.New(cfg.JWTSecret)
	notificationService := platformnotification.NewLogger(loggerInstance)

	router := chi.NewRouter()
	router.Use(apphttp.Stack(tokenService, loggerInstance))

	router.Get("/health", func(responseWriter http.ResponseWriter, _ *http.Request) {
		responseWriter.WriteHeader(http.StatusNoContent)
	})

	auth.New(databaseInstance, tokenService).RegisterRoutes(router)

	authenticate := apphttp.Authenticate(tokenService)
	teams.New(databaseInstance).RegisterRoutes(router, authenticate)
	tasks.New(databaseInstance, notificationService).RegisterRoutes(router, authenticate)

	server := apphttp.NewServer(cfg.Addr, router)
	go server.ListenAndServe()

	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop

	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownSeconds)*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
