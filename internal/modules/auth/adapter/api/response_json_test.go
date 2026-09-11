package api

import (
	"encoding/json"
	"testing"
	"time"

	"taskmanagement/internal/modules/auth/domain"
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

func TestUserOutputJSONKeys(t *testing.T) {
	keys := jsonKeys(t, domain.UserOutput{ID: "user-1", Name: "Alice", Email: "alice@fatkulnurk.com"})
	assertKeys(t, keys, []string{"id", "name", "email"}, []string{"ID", "Name", "Email"})
}

func TestTokenPairJSONKeys(t *testing.T) {
	keys := jsonKeys(t, domain.TokenPair{
		AccessToken:           "access",
		RefreshToken:          "refresh",
		TokenType:             "Bearer",
		ExpiresIn:             900,
		AccessTokenExpiresAt:  time.Now(),
		RefreshTokenExpiresAt: time.Now(),
	})
	assertKeys(t, keys,
		[]string{"access_token", "refresh_token", "token_type", "expires_in", "access_token_expires_at", "refresh_token_expires_at"},
		[]string{"AccessToken", "RefreshToken", "TokenType", "ExpiresIn", "AccessTokenExpiresAt", "RefreshTokenExpiresAt"},
	)
}
