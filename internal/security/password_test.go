package security_test

import (
	"testing"

	"agnos-assignment/internal/security"
)

func TestBcryptHasherHashesAndMatchesPassword(t *testing.T) {
	hasher := security.NewBcryptHasher()
	hash, err := hasher.Hash("password123")
	if err != nil {
		t.Fatalf("hash password: %v", err)
	}
	if hash == "password123" {
		t.Fatal("password must not be stored as plaintext")
	}
	if !hasher.Matches("password123", hash) {
		t.Fatal("expected matching password to be accepted")
	}
	if hasher.Matches("incorrect-password", hash) {
		t.Fatal("expected incorrect password to be rejected")
	}
}
