package auth

import (
	"testing"
)

func TestHashPassword(t *testing.T) {
	hash, err := HashPassword("testpassword123")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	if hash == "" {
		t.Error("HashPassword() returned empty string")
	}
	if hash == "testpassword123" {
		t.Error("HashPassword() returned plaintext password")
	}
}

func TestHashPassword_DifferentHashesForSamePassword(t *testing.T) {
	hash1, _ := HashPassword("testpassword123")
	hash2, _ := HashPassword("testpassword123")

	// bcrypt produces different hashes due to random salt
	if hash1 == hash2 {
		t.Error("HashPassword() should produce different hashes (different salts)")
	}
}

func TestCheckPassword_Correct(t *testing.T) {
	hash, _ := HashPassword("testpassword123")
	err := CheckPassword("testpassword123", hash)
	if err != nil {
		t.Errorf("CheckPassword() error = %v, want nil for correct password", err)
	}
}

func TestCheckPassword_Incorrect(t *testing.T) {
	hash, _ := HashPassword("testpassword123")
	err := CheckPassword("wrongpassword", hash)
	if err == nil {
		t.Error("CheckPassword() expected error for incorrect password, got nil")
	}
}

func TestCheckPassword_EmptyPassword(t *testing.T) {
	hash, _ := HashPassword("testpassword123")
	err := CheckPassword("", hash)
	if err == nil {
		t.Error("CheckPassword() expected error for empty password, got nil")
	}
}

func TestHashPassword_EmptyInput(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword() error = %v", err)
	}
	// bcrypt allows empty passwords
	err = CheckPassword("", hash)
	if err != nil {
		t.Errorf("CheckPassword() for empty password round-trip failed: %v", err)
	}
}

func TestCheckPassword_InvalidHash(t *testing.T) {
	err := CheckPassword("testpassword123", "not-a-bcrypt-hash")
	if err == nil {
		t.Error("CheckPassword() expected error for invalid hash, got nil")
	}
}
