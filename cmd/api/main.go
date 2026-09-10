package main

import (
	"context"
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
	_ = godotenv.Load()
	cfg, err := config.Load()
	if err != nil {
		slog.Error("configuration", "error", err)
		os.Exit(1)
	}
	database, err := database.Open(cfg.Database)
	if err != nil {
		slog.Error("database", "error", err)
		os.Exit(1)
	}
	defer database.Close()
	tokenService := jwtservice.New(cfg.JWTSecret)
	router := chi.NewRouter()
	loggerInstance := logger.New(os.Stdout)
	notificationService := platformnotification.NewLogger(loggerInstance)
	router.Use(apphttp.Stack(tokenService, loggerInstance))
	router.Get("/health", func(responseWriter http.ResponseWriter, _ *http.Request) { responseWriter.WriteHeader(204) })
	auth.New(database, tokenService).RegisterRoutes(router)
	authenticate := apphttp.Authenticate(tokenService)
	teams.New(database).RegisterRoutes(router, authenticate)
	tasks.New(database, notificationService).RegisterRoutes(router, authenticate)
	server := apphttp.NewServer(cfg.Addr, router)
	go server.ListenAndServe()
	stop := make(chan os.Signal, 1)
	signal.Notify(stop, syscall.SIGINT, syscall.SIGTERM)
	<-stop
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(cfg.ShutdownSeconds)*time.Second)
	defer cancel()
	_ = server.Shutdown(ctx)
}
