package auth

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"fmt"
	"io"

	"golang.org/x/crypto/hkdf"

	"github.com/aws/aws-sdk-go-v2/service/kms"
	kmsTypes "github.com/aws/aws-sdk-go-v2/service/kms/types"
)

// SecretEncryptor encrypts and decrypts TOTP secrets at rest.
type SecretEncryptor interface {
	Encrypt(ctx context.Context, plaintext []byte) ([]byte, error)
	Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error)
}

// ---------------------------------------------------------------------------
// Local encryptor: HKDF-derived AES-256-GCM from JWT secret
// ---------------------------------------------------------------------------

// LocalEncryptor uses an AES-256-GCM key derived from the JWT secret via HKDF.
type LocalEncryptor struct {
	aead cipher.AEAD
}

// NewLocalEncryptor derives a 256-bit AES key from the JWT secret using HKDF
// with SHA-256 and a fixed context label to ensure key separation.
func NewLocalEncryptor(jwtSecret []byte) (*LocalEncryptor, error) {
	// HKDF: extract-then-expand with a context label to separate this key
	// from the JWT signing key material.
	info := []byte("sitaware-mfa-totp-v1")
	hkdfReader := hkdf.New(sha256.New, jwtSecret, nil, info)

	derivedKey := make([]byte, 32) // AES-256
	if _, err := io.ReadFull(hkdfReader, derivedKey); err != nil {
		return nil, fmt.Errorf("hkdf derive key: %w", err)
	}

	block, err := aes.NewCipher(derivedKey)
	if err != nil {
		return nil, fmt.Errorf("aes new cipher: %w", err)
	}

	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("gcm new: %w", err)
	}

	return &LocalEncryptor{aead: aead}, nil
}

// Encrypt encrypts plaintext using AES-256-GCM with a random nonce.
// Output format: nonce || ciphertext (nonce is prepended).
func (e *LocalEncryptor) Encrypt(_ context.Context, plaintext []byte) ([]byte, error) {
	nonce := make([]byte, e.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("generate nonce: %w", err)
	}
	return e.aead.Seal(nonce, nonce, plaintext, nil), nil
}

// Decrypt decrypts ciphertext produced by Encrypt.
func (e *LocalEncryptor) Decrypt(_ context.Context, ciphertext []byte) ([]byte, error) {
	nonceSize := e.aead.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	nonce, ct := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.aead.Open(nil, nonce, ct, nil)
	if err != nil {
		return nil, fmt.Errorf("decrypt: %w", err)
	}
	return plaintext, nil
}

// ---------------------------------------------------------------------------
// KMS encryptor: AWS KMS Encrypt/Decrypt
// ---------------------------------------------------------------------------

// KMSEncryptor uses AWS KMS to encrypt and decrypt secrets.
// The encryption key never leaves AWS.
type KMSEncryptor struct {
	client *kms.Client
	keyARN string
}

// NewKMSEncryptor creates a KMS-backed encryptor.
func NewKMSEncryptor(client *kms.Client, keyARN string) *KMSEncryptor {
	return &KMSEncryptor{client: client, keyARN: keyARN}
}

// Encrypt encrypts plaintext using the KMS key.
func (e *KMSEncryptor) Encrypt(ctx context.Context, plaintext []byte) ([]byte, error) {
	out, err := e.client.Encrypt(ctx, &kms.EncryptInput{
		KeyId:               &e.keyARN,
		Plaintext:           plaintext,
		EncryptionAlgorithm: kmsTypes.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("kms encrypt: %w", err)
	}
	return out.CiphertextBlob, nil
}

// Decrypt decrypts ciphertext that was encrypted with the KMS key.
func (e *KMSEncryptor) Decrypt(ctx context.Context, ciphertext []byte) ([]byte, error) {
	out, err := e.client.Decrypt(ctx, &kms.DecryptInput{
		CiphertextBlob:      ciphertext,
		KeyId:               &e.keyARN,
		EncryptionAlgorithm: kmsTypes.EncryptionAlgorithmSpecSymmetricDefault,
	})
	if err != nil {
		return nil, fmt.Errorf("kms decrypt: %w", err)
	}
	return out.Plaintext, nil
}
