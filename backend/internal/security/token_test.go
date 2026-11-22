package security

import "testing"

func TestGenerateToken(t *testing.T) {
	first, err := GenerateToken()
	if err != nil {
		t.Fatalf("generate first token: %v", err)
	}
	second, err := GenerateToken()
	if err != nil {
		t.Fatalf("generate second token: %v", err)
	}
	if first == second {
		t.Fatal("expected unique tokens")
	}
	if first == "" {
		t.Fatal("expected non-empty token")
	}
	if len(HashToken(first)) != 64 {
		t.Fatalf("expected sha256 hex hash length 64, got %d", len(HashToken(first)))
	}
}
