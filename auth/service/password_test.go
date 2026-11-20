package service

import (
	"strings"
	"testing"
)

func TestHashPasswordAndVerify(t *testing.T) {
	hash, err := HashPassword("correct-password")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if !strings.HasPrefix(hash, "$2") {
		t.Fatalf("HashPassword() did not return a bcrypt hash: %q", hash)
	}
	if hash == "correct-password" {
		t.Fatal("HashPassword() returned the plaintext password")
	}

	valid, needsRehash, err := VerifyPassword(hash, "correct-password")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !valid || needsRehash {
		t.Fatalf("VerifyPassword() valid=%v needsRehash=%v, want true/false", valid, needsRehash)
	}

	valid, _, err = VerifyPassword(hash, "wrong-password")
	if err != nil {
		t.Fatalf("VerifyPassword() wrong password error = %v", err)
	}
	if valid {
		t.Fatal("VerifyPassword() accepted an incorrect password")
	}
}

func TestVerifyPasswordLegacyPlaintext(t *testing.T) {
	valid, needsRehash, err := VerifyPassword("legacy-password", "legacy-password")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !valid || !needsRehash {
		t.Fatalf("VerifyPassword() valid=%v needsRehash=%v, want true/true", valid, needsRehash)
	}

	valid, needsRehash, err = VerifyPassword("legacy-password", "wrong-password")
	if err != nil {
		t.Fatalf("VerifyPassword() wrong password error = %v", err)
	}
	if valid || needsRehash {
		t.Fatalf("VerifyPassword() valid=%v needsRehash=%v, want false/false", valid, needsRehash)
	}
}

func TestVerifyPasswordAcceptsLegacyBcryptPrefix(t *testing.T) {
	hash, err := HashPassword("legacy-bcrypt")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	legacyHash := "$2y$" + hash[4:]

	valid, needsRehash, err := VerifyPassword(legacyHash, "legacy-bcrypt")
	if err != nil {
		t.Fatalf("VerifyPassword() error = %v", err)
	}
	if !valid || !needsRehash {
		t.Fatalf("VerifyPassword() valid=%v needsRehash=%v, want true/true", valid, needsRehash)
	}
}
