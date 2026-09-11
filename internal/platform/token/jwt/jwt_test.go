package jwt

import (
	"context"
	applicationtoken "taskmanagement/internal/application/token"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const testSecret = "0123456789abcdef0123456789abcdef"

func TestIssueVerify(t *testing.T) {
	tokenService := New(testSecret)

	rawToken, ttl, err := tokenService.Issue(context.Background(), "user-1")
	if err != nil {
		t.Fatalf("issue error: %v", err)
	}
	if ttl != int64((15 * time.Minute).Seconds()) {
		t.Errorf("ttl = %d, want %d", ttl, int64((15 * time.Minute).Seconds()))
	}

	userID, err := tokenService.Verify(context.Background(), rawToken)
	if err != nil {
		t.Fatalf("verify error: %v", err)
	}
	if userID != "user-1" {
		t.Errorf("user id = %q, want %q", userID, "user-1")
	}
}

func TestVerifyRejects(t *testing.T) {
	now := time.Now()

	noneToken := jwt.NewWithClaims(jwt.SigningMethodNone, jwt.MapClaims{"sub": "user-1", "exp": now.Add(time.Hour).Unix()})
	noneRaw, err := noneToken.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("failed to sign none token: %v", err)
	}

	hs384Token := jwt.NewWithClaims(jwt.SigningMethodHS384, jwt.MapClaims{"sub": "user-1", "exp": now.Add(time.Hour).Unix()})
	hs384Raw, err := hs384Token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign hs384 token: %v", err)
	}

	noExpiryToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-1"})
	noExpiryRaw, err := noExpiryToken.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign no-expiry token: %v", err)
	}

	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"sub": "user-1", "exp": now.Add(-time.Hour).Unix()})
	expiredRaw, err := expiredToken.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("failed to sign expired token: %v", err)
	}

	tests := []struct {
		name     string
		token    string
		verifier applicationtoken.TokenService
	}{
		{name: "alg none", token: noneRaw, verifier: New(testSecret)},
		{name: "alg hs384", token: hs384Raw, verifier: New(testSecret)},
		{name: "missing exp", token: noExpiryRaw, verifier: New(testSecret)},
		{name: "expired", token: expiredRaw, verifier: New(testSecret)},
		{name: "wrong secret", token: hs384Raw, verifier: New("ffffffffffffffffffffffffffffffff")},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if _, err := test.verifier.Verify(context.Background(), test.token); err == nil {
				t.Fatal("expected an error, got nil")
			}
		})
	}
}
