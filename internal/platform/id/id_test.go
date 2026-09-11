package id

import "testing"

func TestNew(t *testing.T) {
	generated := New()
	if generated == "" {
		t.Fatal("expected a non-empty id")
	}
	if !IsValid(generated) {
		t.Fatalf("generated id %q is not a valid UUID", generated)
	}
}

func TestNewIsUnique(t *testing.T) {
	seen := make(map[string]struct{}, 1000)
	for index := 0; index < 1000; index++ {
		generated := New()
		if _, exists := seen[generated]; exists {
			t.Fatalf("duplicate id %q", generated)
		}
		seen[generated] = struct{}{}
	}
}

func TestIsValid(t *testing.T) {
	tests := []struct {
		name  string
		value string
		want  bool
	}{
		{name: "canonical", value: "11111111-1111-4111-8111-111111111111", want: true},
		{name: "generated", value: New(), want: true},
		{name: "dashless form accepted by the parser", value: "11111111111141118111111111111111", want: true},
		{name: "braced form accepted by the parser", value: "{11111111-1111-4111-8111-111111111111}", want: true},
		{name: "empty", value: "", want: false},
		{name: "not a uuid", value: "not-a-uuid", want: false},
		{name: "too short", value: "11111111-1111-4111-8111-11111111111", want: false},
		{name: "extra characters", value: "11111111-1111-4111-8111-111111111111ff", want: false},
		{name: "non hex", value: "zzzzzzzz-zzzz-4zzz-8zzz-zzzzzzzzzzzz", want: false},
		{name: "whitespace", value: " 11111111-1111-4111-8111-111111111111", want: false},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := IsValid(test.value); got != test.want {
				t.Errorf("IsValid(%q) = %v, want %v", test.value, got, test.want)
			}
		})
	}
}
