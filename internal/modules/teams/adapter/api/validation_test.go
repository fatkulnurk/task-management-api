package api

import (
	"strings"
	"testing"

	phttp "taskmanagement/internal/platform/http"
)

func TestCreateRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      CreateRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{name: "valid", request: CreateRequest{Name: "Platform Engineering"}, wantNoErrors: true},
		{name: "empty name", request: CreateRequest{Name: ""}, wantFields: []string{"name"}},
		{name: "name too long", request: CreateRequest{Name: strings.Repeat("a", 121)}, wantFields: []string{"name"}},
		{name: "name at max length", request: CreateRequest{Name: strings.Repeat("a", 120)}, wantNoErrors: true},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestAddRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      AddRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{name: "valid by user id", request: AddRequest{UserID: "0123456789abcdef0123456789abcdef"}, wantNoErrors: true},
		{name: "valid by email", request: AddRequest{Email: "bob@fatkulnurk.com"}, wantNoErrors: true},
		{name: "both empty", request: AddRequest{}, wantFields: []string{"user_id"}},
		{name: "both provided", request: AddRequest{UserID: "0123456789abcdef0123456789abcdef", Email: "bob@fatkulnurk.com"}, wantFields: []string{"user_id"}},
		{name: "user id not uuid", request: AddRequest{UserID: "not-a-uuid"}, wantFields: []string{"user_id"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestPaginationRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      PaginationRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{name: "valid", request: PaginationRequest{Page: 1, Limit: 20}, wantNoErrors: true},
		{name: "limit at max", request: PaginationRequest{Page: 1, Limit: 100}, wantNoErrors: true},
		{name: "page zero", request: PaginationRequest{Page: 0, Limit: 20}, wantFields: []string{"page"}},
		{name: "page negative", request: PaginationRequest{Page: -1, Limit: 20}, wantFields: []string{"page"}},
		{name: "limit zero", request: PaginationRequest{Page: 1, Limit: 0}, wantFields: []string{"limit"}},
		{name: "limit over max", request: PaginationRequest{Page: 1, Limit: 101}, wantFields: []string{"limit"}},
		{name: "page and limit invalid", request: PaginationRequest{Page: 0, Limit: 0}, wantFields: []string{"page", "limit"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

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
