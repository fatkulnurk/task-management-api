package database

import (
	"strings"
	"testing"
	"time"
)

func validConfig() Config {
	return Config{
		DSN:             "user:pass@tcp(127.0.0.1:3306)/db?parseTime=true",
		MaxOpenConns:    10,
		MaxIdleConns:    5,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 5 * time.Minute,
		PingTimeout:     time.Second,
	}
}

func TestOpenValidatesConfig(t *testing.T) {
	tests := []struct {
		name    string
		mutate  func(*Config)
		wantErr string
	}{
		{
			name:    "empty dsn",
			mutate:  func(config *Config) { config.DSN = "" },
			wantErr: "database DSN is required",
		},
		{
			name:    "max open below one",
			mutate:  func(config *Config) { config.MaxOpenConns = 0 },
			wantErr: "database max open connections must be positive",
		},
		{
			name:    "max idle negative",
			mutate:  func(config *Config) { config.MaxIdleConns = -1 },
			wantErr: "database max idle connections must be between zero and max open connections",
		},
		{
			name:    "max idle exceeds max open",
			mutate:  func(config *Config) { config.MaxIdleConns = config.MaxOpenConns + 1 },
			wantErr: "database max idle connections must be between zero and max open connections",
		},
		{
			name:    "connection max lifetime zero",
			mutate:  func(config *Config) { config.ConnMaxLifetime = 0 },
			wantErr: "database connection max lifetime must be positive",
		},
		{
			name:    "connection max idle time negative",
			mutate:  func(config *Config) { config.ConnMaxIdleTime = -time.Second },
			wantErr: "database connection max idle time must be positive",
		},
		{
			name:    "ping timeout zero",
			mutate:  func(config *Config) { config.PingTimeout = 0 },
			wantErr: "database ping timeout must be positive",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			config := validConfig()
			test.mutate(&config)

			database, err := Open(config)
			if err == nil {
				_ = database.Close()
				t.Fatalf("expected an error, got nil")
			}
			if database != nil {
				_ = database.Close()
				t.Fatal("expected a nil database on error")
			}
			if !strings.Contains(err.Error(), test.wantErr) {
				t.Errorf("error = %q, want it to contain %q", err.Error(), test.wantErr)
			}
		})
	}
}
