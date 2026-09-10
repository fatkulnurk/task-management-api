package config

import (
	"strings"
	"testing"
)

func TestLoadJWTSecret(t *testing.T) {
	tests := []struct {
		name      string
		secret    string
		wantError bool
	}{
		{name: "missing", secret: "", wantError: true},
		{name: "too short", secret: strings.Repeat("a", 31), wantError: true},
		{name: "minimum length", secret: strings.Repeat("a", 32)},
		{name: "long", secret: strings.Repeat("a", 64)},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv("JWT_SECRET", test.secret)

			configuration, err := Load()
			if test.wantError {
				if err == nil {
					t.Fatal("expected an error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("error = %v, want nil", err)
			}
			if configuration.JWTSecret != test.secret {
				t.Errorf("secret = %q, want %q", configuration.JWTSecret, test.secret)
			}
		})
	}
}

func TestLoadDefaults(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))

	configuration, err := Load()
	if err != nil {
		t.Fatalf("error = %v, want nil", err)
	}
	if configuration.Addr != ":8080" {
		t.Errorf("addr = %q, want %q", configuration.Addr, ":8080")
	}
	if configuration.ShutdownSeconds != 10 {
		t.Errorf("shutdown seconds = %d, want 10", configuration.ShutdownSeconds)
	}
	if configuration.Database.MaxOpenConns != 25 {
		t.Errorf("max open conns = %d, want 25", configuration.Database.MaxOpenConns)
	}
	if configuration.Database.MaxIdleConns != 10 {
		t.Errorf("max idle conns = %d, want 10", configuration.Database.MaxIdleConns)
	}
}

func TestLoadInvalidDatabaseSettings(t *testing.T) {
	t.Setenv("JWT_SECRET", strings.Repeat("a", 32))

	tests := []struct {
		name  string
		key   string
		value string
	}{
		{name: "zero open conns", key: "DB_MAX_OPEN_CONNS", value: "0"},
		{name: "negative idle conns", key: "DB_MAX_IDLE_CONNS", value: "-1"},
		{name: "idle exceeds open", key: "DB_MAX_IDLE_CONNS", value: "50"},
		{name: "invalid duration", key: "DB_CONN_MAX_LIFETIME", value: "soon"},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			t.Setenv(test.key, test.value)

			if _, err := Load(); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}
