package model

import (
	"errors"
	"testing"
	"time"
)

func TestCreateAPITokenRequest_Validate_EmptyName(t *testing.T) {
	req := CreateAPITokenRequest{Name: ""}
	err := req.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if ve.Message != "name is required" {
		t.Errorf("message = %q", ve.Message)
	}
}

func TestCreateAPITokenRequest_Validate_WhitespaceOnlyName(t *testing.T) {
	req := CreateAPITokenRequest{Name: "   "}
	err := req.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
}

func TestCreateAPITokenRequest_Validate_NameTooLong(t *testing.T) {
	long := make([]byte, 101)
	for i := range long {
		long[i] = 'a'
	}
	req := CreateAPITokenRequest{Name: string(long)}
	err := req.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if ve.Message != "name must not exceed 100 characters" {
		t.Errorf("message = %q", ve.Message)
	}
}

func TestCreateAPITokenRequest_Validate_PastExpiry(t *testing.T) {
	past := time.Now().Add(-1 * time.Hour)
	req := CreateAPITokenRequest{Name: "test", ExpiresAt: &past}
	err := req.Validate()
	var ve *ValidationError
	if !errors.As(err, &ve) {
		t.Fatalf("expected ValidationError, got %v", err)
	}
	if ve.Message != "expires_at must be in the future" {
		t.Errorf("message = %q", ve.Message)
	}
}

func TestCreateAPITokenRequest_Validate_ValidNoExpiry(t *testing.T) {
	req := CreateAPITokenRequest{Name: "my-token"}
	if err := req.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if req.Name != "my-token" {
		t.Errorf("name = %q after trim", req.Name)
	}
}

func TestCreateAPITokenRequest_Validate_ValidWithExpiry(t *testing.T) {
	future := time.Now().Add(24 * time.Hour)
	req := CreateAPITokenRequest{Name: "  load-test  ", ExpiresAt: &future}
	if err := req.Validate(); err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if req.Name != "load-test" {
		t.Errorf("name should be trimmed, got %q", req.Name)
	}
}

func TestAPIToken_ToResponse(t *testing.T) {
	now := time.Now()
	token := APIToken{
		Name:      "test",
		CreatedAt: now,
	}
	resp := token.ToResponse()
	if resp.Name != "test" {
		t.Errorf("Name = %q", resp.Name)
	}
	if resp.ExpiresAt != nil {
		t.Error("ExpiresAt should be nil")
	}
	if resp.LastUsedAt != nil {
		t.Error("LastUsedAt should be nil")
	}
}
