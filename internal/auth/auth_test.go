package auth_test

import (
	"testing"

	"github.com/yourname/cli-login-system/internal/auth"
)

func TestHashPassword(t *testing.T) {
	plain := "securePass123"

	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}
	if hash == "" {
		t.Fatal("expected non-empty hash")
	}
	if hash == plain {
		t.Fatal("hash must not equal plain text")
	}
}

func TestCheckPassword(t *testing.T) {
	plain := "securePass123"

	hash, err := auth.HashPassword(plain)
	if err != nil {
		t.Fatalf("HashPassword error: %v", err)
	}

	if !auth.CheckPassword(plain, hash) {
		t.Error("CheckPassword should return true for correct password")
	}
	if auth.CheckPassword("wrongpassword", hash) {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestHashPasswordTooShort(t *testing.T) {
	_, err := auth.HashPassword("short")
	if err == nil {
		t.Error("expected error for password shorter than 8 chars")
	}
}

func TestHashPasswordUniqueness(t *testing.T) {
	plain := "securePass123"

	h1, _ := auth.HashPassword(plain)
	h2, _ := auth.HashPassword(plain)

	// bcrypt uses a random salt so two hashes must differ
	if h1 == h2 {
		t.Error("two hashes of the same password must differ (different salts)")
	}
}
