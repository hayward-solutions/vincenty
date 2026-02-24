package auth

import (
	"bytes"
	"context"
	"testing"
)

func TestLocalEncryptor_RoundTrip(t *testing.T) {
	enc, err := NewLocalEncryptor([]byte("my-secret-jwt-key-at-least-32-chars"))
	if err != nil {
		t.Fatalf("NewLocalEncryptor() error = %v", err)
	}

	plaintext := []byte("JBSWY3DPEHPK3PXP") // example TOTP secret
	ctx := context.Background()

	ciphertext, err := enc.Encrypt(ctx, plaintext)
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}

	if bytes.Equal(ciphertext, plaintext) {
		t.Error("ciphertext should not equal plaintext")
	}

	decrypted, err := enc.Decrypt(ctx, ciphertext)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}

	if !bytes.Equal(decrypted, plaintext) {
		t.Errorf("Decrypt() = %q, want %q", decrypted, plaintext)
	}
}

func TestLocalEncryptor_DifferentCiphertexts(t *testing.T) {
	enc, _ := NewLocalEncryptor([]byte("my-secret-jwt-key-at-least-32-chars"))
	ctx := context.Background()
	plaintext := []byte("test-data")

	ct1, _ := enc.Encrypt(ctx, plaintext)
	ct2, _ := enc.Encrypt(ctx, plaintext)

	// AES-GCM with random nonce should produce different ciphertexts
	if bytes.Equal(ct1, ct2) {
		t.Error("encrypting same plaintext twice should produce different ciphertexts")
	}
}

func TestLocalEncryptor_DifferentKeys(t *testing.T) {
	enc1, _ := NewLocalEncryptor([]byte("key-one-at-least-32-chars-long!!"))
	enc2, _ := NewLocalEncryptor([]byte("key-two-at-least-32-chars-long!!"))
	ctx := context.Background()
	plaintext := []byte("test-data")

	ct1, _ := enc1.Encrypt(ctx, plaintext)
	_, err := enc2.Decrypt(ctx, ct1)
	if err == nil {
		t.Error("Decrypt() with wrong key should return error")
	}
}

func TestLocalEncryptor_DecryptTooShort(t *testing.T) {
	enc, _ := NewLocalEncryptor([]byte("my-secret-jwt-key-at-least-32-chars"))
	ctx := context.Background()

	_, err := enc.Decrypt(ctx, []byte("short"))
	if err == nil {
		t.Error("Decrypt() with too-short ciphertext should return error")
	}
}

func TestLocalEncryptor_DecryptCorrupted(t *testing.T) {
	enc, _ := NewLocalEncryptor([]byte("my-secret-jwt-key-at-least-32-chars"))
	ctx := context.Background()

	ct, _ := enc.Encrypt(ctx, []byte("test-data"))
	// Corrupt the ciphertext
	ct[len(ct)-1] ^= 0xFF

	_, err := enc.Decrypt(ctx, ct)
	if err == nil {
		t.Error("Decrypt() with corrupted ciphertext should return error")
	}
}

func TestLocalEncryptor_EmptyPlaintext(t *testing.T) {
	enc, _ := NewLocalEncryptor([]byte("my-secret-jwt-key-at-least-32-chars"))
	ctx := context.Background()

	ct, err := enc.Encrypt(ctx, []byte{})
	if err != nil {
		t.Fatalf("Encrypt() empty plaintext error = %v", err)
	}

	pt, err := enc.Decrypt(ctx, ct)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if len(pt) != 0 {
		t.Errorf("expected empty plaintext, got %q", pt)
	}
}

func TestNewLocalEncryptor_ShortKey(t *testing.T) {
	// HKDF should work with any key length
	enc, err := NewLocalEncryptor([]byte("short"))
	if err != nil {
		t.Fatalf("NewLocalEncryptor() should work with short key, got error = %v", err)
	}

	ctx := context.Background()
	ct, err := enc.Encrypt(ctx, []byte("test"))
	if err != nil {
		t.Fatalf("Encrypt() error = %v", err)
	}
	pt, err := enc.Decrypt(ctx, ct)
	if err != nil {
		t.Fatalf("Decrypt() error = %v", err)
	}
	if !bytes.Equal(pt, []byte("test")) {
		t.Errorf("round-trip failed: got %q", pt)
	}
}
