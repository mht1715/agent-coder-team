package security

import "testing"

func TestPasswordHasher(t *testing.T) {
	hasher := NewPasswordHasher()

	hash, err := hasher.Hash("correct horse battery staple")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}

	ok, err := hasher.Verify("correct horse battery staple", hash)
	if err != nil {
		t.Fatalf("verify password: %v", err)
	}
	if !ok {
		t.Fatal("expected password to match")
	}

	ok, err = hasher.Verify("wrong password", hash)
	if err != nil {
		t.Fatalf("verify wrong password: %v", err)
	}
	if ok {
		t.Fatal("expected password not to match")
	}
}
