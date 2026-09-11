package api

import (
	"encoding/json"
	"testing"

	"taskmanagement/internal/modules/teams/domain"
)

func jsonKeys(t *testing.T, value any) map[string]json.RawMessage {
	t.Helper()
	body, err := json.Marshal(value)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var keys map[string]json.RawMessage
	if err = json.Unmarshal(body, &keys); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	return keys
}

func assertKeys(t *testing.T, keys map[string]json.RawMessage, want []string, reject []string) {
	t.Helper()
	for _, key := range want {
		if _, ok := keys[key]; !ok {
			t.Errorf("missing key %q in %v", key, keys)
		}
	}
	for _, key := range reject {
		if _, ok := keys[key]; ok {
			t.Errorf("unexpected key %q in %v", key, keys)
		}
	}
}

func TestTeamJSONKeys(t *testing.T) {
	keys := jsonKeys(t, domain.Team{ID: "team-1", OwnerID: "user-1", Name: "Platform Engineering", CreatedAt: "2026-09-10T10:00:00Z", UpdatedAt: "2026-09-10T10:00:00Z"})
	assertKeys(t, keys, []string{"id", "owner_id", "name", "created_at", "updated_at"}, []string{"ID", "OwnerID", "Name", "CreatedAt", "UpdatedAt"})
}

func TestMemberJSONKeys(t *testing.T) {
	keys := jsonKeys(t, domain.Member{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com", IsOwner: true})
	assertKeys(t, keys, []string{"id", "name", "email", "is_owner"}, []string{"ID", "Name", "Email", "IsOwner"})
}
