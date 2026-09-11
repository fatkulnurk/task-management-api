package password

import (
	"strings"
	"testing"
)

func TestHash(t *testing.T) {
	raw := "correct-horse-battery-staple"

	hashed, err := Hash(raw)
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}
	if hashed == raw {
		t.Fatal("hash must not equal the raw password")
	}
	if !strings.HasPrefix(hashed, "$2") {
		t.Errorf("hash %q is not a bcrypt hash", hashed)
	}
}

func TestHashIsSalted(t *testing.T) {
	first, err := Hash("same-password")
	if err != nil {
		t.Fatalf("first hash error: %v", err)
	}
	second, err := Hash("same-password")
	if err != nil {
		t.Fatalf("second hash error: %v", err)
	}
	if first == second {
		t.Error("two hashes of the same password must differ (salt)")
	}
}

func TestCompare(t *testing.T) {
	hashed, err := Hash("correct-horse")
	if err != nil {
		t.Fatalf("hash error: %v", err)
	}

	tests := []struct {
		name string
		hash string
		raw  string
		want bool
	}{
		{name: "match", hash: hashed, raw: "correct-horse", want: true},
		{name: "wrong password", hash: hashed, raw: "wrong-horse", want: false},
		{name: "empty password", hash: hashed, raw: "", want: false},
		{name: "malformed hash", hash: "not-a-bcrypt-hash", raw: "correct-horse", want: false},
		{name: "empty hash", hash: "", raw: "correct-horse", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := Compare(test.hash, test.raw); got != test.want {
				t.Errorf("Compare(%q, %q) = %v, want %v", test.hash, test.raw, got, test.want)
			}
		})
	}
}
