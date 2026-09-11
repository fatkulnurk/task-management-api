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
		{
			name:         "valid",
			request:      CreateRequest{TeamID: "team-1", Title: "Prepare report", Status: "todo"},
			wantNoErrors: true,
		},
		{
			name:       "missing team id",
			request:    CreateRequest{Title: "Prepare report", Status: "todo"},
			wantFields: []string{"team_id"},
		},
		{
			name:       "missing title",
			request:    CreateRequest{TeamID: "team-1", Status: "todo"},
			wantFields: []string{"title"},
		},
		{
			name:       "title too long",
			request:    CreateRequest{TeamID: "team-1", Title: strings.Repeat("a", 256), Status: "todo"},
			wantFields: []string{"title"},
		},
		{
			name:       "missing status",
			request:    CreateRequest{TeamID: "team-1", Title: "Prepare report"},
			wantFields: []string{"status"},
		},
		{
			name:       "invalid status",
			request:    CreateRequest{TeamID: "team-1", Title: "Prepare report", Status: "blocked"},
			wantFields: []string{"status"},
		},
		{
			name:       "all invalid",
			request:    CreateRequest{Status: "blocked"},
			wantFields: []string{"team_id", "title", "status"},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestUpdateRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      UpdateRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{name: "empty is valid", request: UpdateRequest{}, wantNoErrors: true},
		{name: "valid status", request: UpdateRequest{Status: "done"}, wantNoErrors: true},
		{name: "valid title", request: UpdateRequest{Title: "Prepare report v2"}, wantNoErrors: true},
		{name: "title too long", request: UpdateRequest{Title: strings.Repeat("a", 256)}, wantFields: []string{"title"}},
		{name: "invalid status", request: UpdateRequest{Status: "blocked"}, wantFields: []string{"status"}},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertValidation(t, test.request.Validate(), test.wantNoErrors, test.wantFields)
		})
	}
}

func TestAssignRequestValidate(t *testing.T) {
	tests := []struct {
		name         string
		request      AssignRequest
		wantFields   []string
		wantNoErrors bool
	}{
		{name: "valid uuid", request: AssignRequest{AssigneeID: "0123456789abcdef0123456789abcdef"}, wantNoErrors: true},
		{name: "empty is allowed for unassign", request: AssignRequest{}, wantNoErrors: true},
		{name: "not uuid", request: AssignRequest{AssigneeID: "not-a-uuid"}, wantFields: []string{"assignee_id"}},
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
		{name: "valid with filters", request: PaginationRequest{Page: 1, Limit: 20, Status: "todo", TeamID: "0123456789abcdef0123456789abcdef"}, wantNoErrors: true},
		{name: "page zero", request: PaginationRequest{Page: 0, Limit: 20}, wantFields: []string{"page"}},
		{name: "limit over max", request: PaginationRequest{Page: 1, Limit: 101}, wantFields: []string{"limit"}},
		{name: "invalid status", request: PaginationRequest{Page: 1, Limit: 20, Status: "blocked"}, wantFields: []string{"status"}},
		{name: "invalid team id", request: PaginationRequest{Page: 1, Limit: 20, TeamID: "not-a-uuid"}, wantFields: []string{"team_id"}},
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
