//go:build integration

package repository

import (
	"context"
	"database/sql"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"taskmanagement/internal/modules/tasks/domain"
	"taskmanagement/internal/platform/id"

	"github.com/go-sql-driver/mysql"
)

var migrationFiles = []string{
	"001_create_users.up.sql",
	"002_create_refresh_tokens.up.sql",
	"003_create_teams.up.sql",
	"004_create_team_members.up.sql",
	"005_create_tasks.up.sql",
	"006_create_task_logs.up.sql",
	"007_create_idempotency_keys.up.sql",
}

func openIntegrationDatabase(t *testing.T) *sql.DB {
	t.Helper()
	baseDSN := os.Getenv("TEST_DATABASE_URL")
	if baseDSN == "" {
		t.Skip("TEST_DATABASE_URL is not set")
	}

	adminConfig, err := mysql.ParseDSN(baseDSN)
	if err != nil {
		t.Fatalf("failed to parse TEST_DATABASE_URL: %v", err)
	}
	adminConfig.DBName = ""
	adminConfig.MultiStatements = true
	adminConfig.ParseTime = true

	admin, err := sql.Open("mysql", adminConfig.FormatDSN())
	if err != nil {
		t.Fatalf("failed to open admin connection: %v", err)
	}
	if err = admin.Ping(); err != nil {
		_ = admin.Close()
		t.Fatalf("failed to reach database server: %v", err)
	}

	schema := fmt.Sprintf("taskmanagement_test_%d", time.Now().UnixNano())
	if _, err = admin.Exec("CREATE DATABASE `" + schema + "`"); err != nil {
		_ = admin.Close()
		t.Fatalf("failed to create schema: %v", err)
	}
	t.Cleanup(func() {
		_, _ = admin.Exec("DROP DATABASE `" + schema + "`")
		_ = admin.Close()
	})

	config := *adminConfig
	config.DBName = schema
	database, err := sql.Open("mysql", config.FormatDSN())
	if err != nil {
		t.Fatalf("failed to open schema connection: %v", err)
	}
	t.Cleanup(func() {
		_ = database.Close()
	})

	applyMigrations(t, database)
	return database
}

func applyMigrations(t *testing.T, database *sql.DB) {
	t.Helper()
	directory := filepath.Join("..", "..", "..", "..", "migrations")

	for _, name := range migrationFiles {
		statement, err := os.ReadFile(filepath.Join(directory, name))
		if err != nil {
			t.Fatalf("failed to read migration %s: %v", name, err)
		}
		if _, err = database.Exec(string(statement)); err != nil {
			t.Fatalf("failed to apply migration %s: %v", name, err)
		}
	}
}

func seedTeam(t *testing.T, database *sql.DB) (string, string) {
	t.Helper()
	userID := id.New()
	teamID := id.New()

	if _, err := database.Exec("INSERT INTO users(id,name,email,password_hash,created_at,updated_at) VALUES(?,?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", userID, "Concurrent User", "concurrent@fatkulnurk.com", "hash"); err != nil {
		t.Fatalf("failed to insert user: %v", err)
	}
	if _, err := database.Exec("INSERT INTO teams(id,owner_id,name,created_at,updated_at) VALUES(?,?,?,UTC_TIMESTAMP(6),UTC_TIMESTAMP(6))", teamID, userID, "Concurrent Team"); err != nil {
		t.Fatalf("failed to insert team: %v", err)
	}
	if _, err := database.Exec("INSERT INTO team_members(team_id,user_id) VALUES(?,?)", teamID, userID); err != nil {
		t.Fatalf("failed to insert team member: %v", err)
	}
	return userID, teamID
}

func countRows(t *testing.T, database *sql.DB, table string) int {
	t.Helper()
	var count int
	if err := database.QueryRow("SELECT COUNT(*) FROM " + table).Scan(&count); err != nil {
		t.Fatalf("failed to count %s: %v", table, err)
	}
	return count
}

func TestCreateIdempotentConcurrent(t *testing.T) {
	database := openIntegrationDatabase(t)
	userID, teamID := seedTeam(t, database)
	repository := NewMySQLTaskRepository(database)

	const goroutines = 32
	key := id.New()
	requestHash := "concurrent-request-hash"
	body := []byte(`{"id":"concurrent-task"}`)

	results := make([]domain.CreateOutput, goroutines)
	errs := make([]error, goroutines)
	start := make(chan struct{})
	var waitGroup sync.WaitGroup
	for index := 0; index < goroutines; index++ {
		waitGroup.Add(1)
		go func(goroutineIndex int) {
			defer waitGroup.Done()
			<-start
			task := domain.Task{ID: id.New(), TeamID: teamID, CreatorID: userID, Title: "Concurrent task", Status: "todo"}
			results[goroutineIndex], errs[goroutineIndex] = repository.CreateIdempotent(context.Background(), task, userID, key, requestHash, body)
		}(index)
	}
	close(start)
	waitGroup.Wait()

	created := 0
	for index := range results {
		if errs[index] != nil {
			t.Fatalf("goroutine %d error: %v", index, errs[index])
		}
		if results[index].Status != 201 {
			t.Errorf("goroutine %d status = %d, want 201", index, results[index].Status)
		}
		if !results[index].Replay {
			created++
		}
	}
	if created != 1 {
		t.Errorf("created count = %d, want 1", created)
	}
	if got := countRows(t, database, "tasks"); got != 1 {
		t.Errorf("tasks = %d, want 1", got)
	}
	if got := countRows(t, database, "idempotency_keys"); got != 1 {
		t.Errorf("idempotency_keys = %d, want 1", got)
	}
	if got := countRows(t, database, "task_logs"); got != 1 {
		t.Errorf("task_logs = %d, want 1", got)
	}
}

func TestCreateIdempotentSequentialReplay(t *testing.T) {
	database := openIntegrationDatabase(t)
	userID, teamID := seedTeam(t, database)
	repository := NewMySQLTaskRepository(database)

	key := id.New()
	requestHash := "sequential-request-hash"
	body := []byte(`{"id":"sequential-task"}`)
	task := domain.Task{ID: id.New(), TeamID: teamID, CreatorID: userID, Title: "Sequential task", Status: "todo"}

	first, err := repository.CreateIdempotent(context.Background(), task, userID, key, requestHash, body)
	if err != nil {
		t.Fatalf("first create error: %v", err)
	}
	second, err := repository.CreateIdempotent(context.Background(), task, userID, key, requestHash, body)
	if err != nil {
		t.Fatalf("second create error: %v", err)
	}

	if first.Status != 201 || first.Replay {
		t.Errorf("first = %+v, want status 201 and replay false", first)
	}
	if second.Status != 201 || !second.Replay {
		t.Errorf("second = %+v, want status 201 and replay true", second)
	}
	if got := countRows(t, database, "tasks"); got != 1 {
		t.Errorf("tasks = %d, want 1", got)
	}
}
