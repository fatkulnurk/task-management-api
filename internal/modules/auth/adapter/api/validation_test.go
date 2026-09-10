package api

import (
	"strings"
	"testing"

	phttp "taskmanagement/internal/platform/http"
)

func assertValidation(t *testing.T, validationErrors []phttp.ValidationError, wantNoErrors bool, wantFields []string) {
	t.Helper()
	if wantNoErrors {
		if len(validationErrors) != 0 {
			t.Fatalf("expected no validation errors, got %+v", validationErrors)
		}
		return
	}
	if len(validationErrors) == 0 {
		t.Fatal("expected at least one validation error, got none")
	}
	if len(wantFields) == 0 {
		return
	}
	gotFields := make([]string, 0, len(validationErrors))
	for _, validationError := range validationErrors {
		gotFields = append(gotFields, validationError.Field)
	}
	if len(gotFields) != len(wantFields) {
		t.Fatalf("fields = %v, want %v", gotFields, wantFields)
	}
	for index := range wantFields {
		if gotFields[index] != wantFields[index] {
			t.Fatalf("fields = %v, want %v", gotFields, wantFields)
		}
	}
}

func TestRegisterRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      RegisterRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{
			name:         "valid",
			request:      RegisterRequest{Name: "Alice Smith", Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantNoErrors: true,
		},
		{
			name:       "empty name",
			request:    RegisterRequest{Name: "   ", Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantFields: []string{"name"},
		},
		{
			name:       "name too long",
			request:    RegisterRequest{Name: strings.Repeat("a", 121), Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantFields: []string{"name"},
		},
		{
			name:       "name with control character",
			request:    RegisterRequest{Name: "Alice\nAdmin", Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantFields: []string{"name"},
		},
		{
			name:         "unicode name accepted",
			request:      RegisterRequest{Name: "Ångström Ünïcode", Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantNoErrors: true,
		},
		{
			name:       "empty email",
			request:    RegisterRequest{Name: "Alice", Email: "", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email",
			request:    RegisterRequest{Name: "Alice", Email: "not-an-email", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "display name email rejected",
			request:    RegisterRequest{Name: "Alice", Email: "Alice <alice@fatkulnurk.com>", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "email with internal space rejected",
			request:    RegisterRequest{Name: "Alice", Email: "ali ce@fatkulnurk.com", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "email too long",
			request:    RegisterRequest{Name: "Alice", Email: strings.Repeat("a", 250) + "@fatkulnurk.com", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "empty password",
			request:    RegisterRequest{Name: "Alice", Email: "alice@fatkulnurk.com", Password: ""},
			wantFields: []string{"password"},
		},
		{
			name:       "password too short",
			request:    RegisterRequest{Name: "Alice", Email: "alice@fatkulnurk.com", Password: "short"},
			wantFields: []string{"password"},
		},
		{
			name:       "password too long",
			request:    RegisterRequest{Name: "Alice", Email: "alice@fatkulnurk.com", Password: strings.Repeat("a", 73)},
			wantFields: []string{"password"},
		},
		{
			name:       "all fields invalid",
			request:    RegisterRequest{Name: "", Email: "bad", Password: ""},
			wantFields: []string{"name", "email", "password"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestLoginRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      LoginRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{
			name:         "valid",
			request:      LoginRequest{Email: "alice@fatkulnurk.com", Password: "correct-horse"},
			wantNoErrors: true,
		},
		{
			name:       "empty email",
			request:    LoginRequest{Email: "", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "invalid email",
			request:    LoginRequest{Email: "bad", Password: "correct-horse"},
			wantFields: []string{"email"},
		},
		{
			name:       "empty password",
			request:    LoginRequest{Email: "alice@fatkulnurk.com", Password: ""},
			wantFields: []string{"password"},
		},
		{
			name:       "password too long",
			request:    LoginRequest{Email: "alice@fatkulnurk.com", Password: strings.Repeat("a", 73)},
			wantFields: []string{"password"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestRefreshAndLogoutRequestValidate(t *testing.T) {
	valid := "0123456789abcdef0123456789abcdef"
	tests := []struct {
		name         string
		token        string
		wantNoErrors bool
	}{
		{name: "valid", token: valid, wantNoErrors: true},
		{name: "empty", token: ""},
		{name: "whitespace only", token: "   "},
		{name: "with internal space", token: "token with space"},
		{name: "with newline", token: "token\nline"},
		{name: "too long", token: strings.Repeat("a", 129)},
	}

	for _, test := range tests {
		t.Run("refresh/"+test.name, func(t *testing.T) {
			assertValidation(t, RefreshRequest{RefreshToken: test.token}.Validate(), test.wantNoErrors, nil)
		})
		t.Run("logout/"+test.name, func(t *testing.T) {
			assertValidation(t, LogoutRequest{RefreshToken: test.token}.Validate(), test.wantNoErrors, nil)
		})
	}
}

func TestValidEmail(t *testing.T) {
	tests := []struct {
		name  string
		email string
		want  bool
	}{
		{name: "plain", email: "alice@fatkulnurk.com", want: true},
		{name: "uppercase", email: "ALICE@FATKULNURK.COM", want: true},
		{name: "padded", email: "  alice@fatkulnurk.com  ", want: true},
		{name: "plus tag", email: "alice+tag@fatkulnurk.com", want: true},
		{name: "not an email", email: "not-an-email", want: false},
		{name: "missing domain", email: "alice@", want: false},
		{name: "missing local part", email: "@fatkulnurk.com", want: false},
		{name: "display name", email: "Alice <alice@fatkulnurk.com>", want: false},
		{name: "internal space", email: "ali ce@fatkulnurk.com", want: false},
		{name: "trailing newline", email: "alice@fatkulnurk.com\n", want: true},
		{name: "empty", email: "", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := validEmail(test.email); got != test.want {
				t.Errorf("validEmail(%q) = %v, want %v", test.email, got, test.want)
			}
		})
	}
}

func TestInvalidToken(t *testing.T) {
	tests := []struct {
		name  string
		token string
		want  bool
	}{
		{name: "empty", token: "", want: true},
		{name: "whitespace only", token: "   ", want: true},
		{name: "internal space", token: "token with space", want: true},
		{name: "newline", token: "token\nline", want: true},
		{name: "too long", token: strings.Repeat("a", 129), want: true},
		{name: "max length", token: strings.Repeat("a", 128), want: false},
		{name: "valid", token: "0123456789abcdef", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := invalidToken(test.token); got != test.want {
				t.Errorf("invalidToken(%q) = %v, want %v", test.token, got, test.want)
			}
		})
	}
}

func TestContainsControl(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "plain", value: "Alice Smith", want: false},
		{name: "tab", value: "Alice\tSmith", want: true},
		{name: "newline", value: "Alice\nSmith", want: true},
		{name: "carriage return", value: "Alice\rSmith", want: true},
		{name: "null byte", value: "\x00", want: true},
		{name: "unicode", value: "Ångström", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := containsControl(test.value); got != test.want {
				t.Errorf("containsControl(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
