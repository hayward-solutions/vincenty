package model

import (
	"testing"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// LoginRequest
// ---------------------------------------------------------------------------

func TestLoginRequest_Validate_Valid(t *testing.T) {
	r := &LoginRequest{Username: "admin", Password: "password123"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v, want nil", err)
	}
}

func TestLoginRequest_Validate_EmptyUsername(t *testing.T) {
	r := &LoginRequest{Username: "", Password: "password123"}
	err := r.Validate()
	if err == nil {
		t.Error("Validate() expected error for empty username")
	}
	assertValidationError(t, err, "username is required")
}

func TestLoginRequest_Validate_EmptyPassword(t *testing.T) {
	r := &LoginRequest{Username: "admin", Password: ""}
	err := r.Validate()
	if err == nil {
		t.Error("Validate() expected error for empty password")
	}
	assertValidationError(t, err, "password is required")
}

// ---------------------------------------------------------------------------
// RefreshRequest
// ---------------------------------------------------------------------------

func TestRefreshRequest_Validate_Valid(t *testing.T) {
	r := &RefreshRequest{RefreshToken: "some-token"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestRefreshRequest_Validate_Empty(t *testing.T) {
	r := &RefreshRequest{RefreshToken: ""}
	if err := r.Validate(); err == nil {
		t.Error("Validate() expected error for empty refresh_token")
	}
}

// ---------------------------------------------------------------------------
// CreateUserRequest
// ---------------------------------------------------------------------------

func TestCreateUserRequest_Validate_Valid(t *testing.T) {
	r := &CreateUserRequest{Username: "testuser", Email: "test@example.com", Password: "password123"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCreateUserRequest_Validate_EmptyUsername(t *testing.T) {
	r := &CreateUserRequest{Username: "", Email: "test@example.com", Password: "password123"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty username")
	}
}

func TestCreateUserRequest_Validate_EmptyEmail(t *testing.T) {
	r := &CreateUserRequest{Username: "test", Email: "", Password: "password123"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty email")
	}
}

func TestCreateUserRequest_Validate_EmptyPassword(t *testing.T) {
	r := &CreateUserRequest{Username: "test", Email: "test@example.com", Password: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty password")
	}
}

func TestCreateUserRequest_Validate_ShortPassword(t *testing.T) {
	r := &CreateUserRequest{Username: "test", Email: "test@example.com", Password: "short"}
	err := r.Validate()
	if err == nil {
		t.Error("expected error for short password")
	}
	assertValidationError(t, err, "password must be at least 8 characters")
}

func TestCreateUserRequest_Validate_ExactlyMinPassword(t *testing.T) {
	r := &CreateUserRequest{Username: "test", Email: "test@example.com", Password: "12345678"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v, password with exactly 8 chars should be valid", err)
	}
}

// ---------------------------------------------------------------------------
// ChangePasswordRequest
// ---------------------------------------------------------------------------

func TestChangePasswordRequest_Validate_Valid(t *testing.T) {
	r := &ChangePasswordRequest{CurrentPassword: "oldpass", NewPassword: "newpassword123"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestChangePasswordRequest_Validate_EmptyCurrent(t *testing.T) {
	r := &ChangePasswordRequest{CurrentPassword: "", NewPassword: "newpassword123"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty current password")
	}
}

func TestChangePasswordRequest_Validate_EmptyNew(t *testing.T) {
	r := &ChangePasswordRequest{CurrentPassword: "oldpass", NewPassword: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty new password")
	}
}

func TestChangePasswordRequest_Validate_ShortNew(t *testing.T) {
	r := &ChangePasswordRequest{CurrentPassword: "oldpass", NewPassword: "short"}
	err := r.Validate()
	if err == nil {
		t.Error("expected error for short new password")
	}
	assertValidationError(t, err, "new password must be at least 8 characters")
}

// ---------------------------------------------------------------------------
// CreateGroupRequest
// ---------------------------------------------------------------------------

func TestCreateGroupRequest_Validate_Valid(t *testing.T) {
	r := &CreateGroupRequest{Name: "Test Group"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCreateGroupRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateGroupRequest{Name: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateGroupRequest_Validate_LongName(t *testing.T) {
	longName := make([]byte, 256)
	for i := range longName {
		longName[i] = 'a'
	}
	r := &CreateGroupRequest{Name: string(longName)}
	if err := r.Validate(); err == nil {
		t.Error("expected error for name > 255 chars")
	}
}

func TestCreateGroupRequest_Validate_ValidMarkerIcon(t *testing.T) {
	icon := "circle"
	r := &CreateGroupRequest{Name: "Test", MarkerIcon: &icon}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v for valid marker icon", err)
	}
}

func TestCreateGroupRequest_Validate_InvalidMarkerIcon(t *testing.T) {
	icon := "invalid-icon"
	r := &CreateGroupRequest{Name: "Test", MarkerIcon: &icon}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid marker icon")
	}
}

func TestCreateGroupRequest_Validate_ValidMarkerColor(t *testing.T) {
	color := "#ff0000"
	r := &CreateGroupRequest{Name: "Test", MarkerColor: &color}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v for valid marker color", err)
	}
}

func TestCreateGroupRequest_Validate_InvalidMarkerColor(t *testing.T) {
	color := "red"
	r := &CreateGroupRequest{Name: "Test", MarkerColor: &color}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid marker color")
	}
}

// ---------------------------------------------------------------------------
// UpdateGroupMarkerRequest
// ---------------------------------------------------------------------------

func TestUpdateGroupMarkerRequest_Validate_BothNil(t *testing.T) {
	r := &UpdateGroupMarkerRequest{}
	if err := r.Validate(); err == nil {
		t.Error("expected error when both fields are nil")
	}
}

func TestUpdateGroupMarkerRequest_Validate_ValidIcon(t *testing.T) {
	icon := "star"
	r := &UpdateGroupMarkerRequest{MarkerIcon: &icon}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestUpdateGroupMarkerRequest_Validate_InvalidIcon(t *testing.T) {
	icon := "unicorn"
	r := &UpdateGroupMarkerRequest{MarkerIcon: &icon}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid icon")
	}
}

func TestUpdateGroupMarkerRequest_Validate_InvalidColor(t *testing.T) {
	color := "notahexcolor"
	r := &UpdateGroupMarkerRequest{MarkerColor: &color}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid color")
	}
}

// ---------------------------------------------------------------------------
// AddGroupMemberRequest
// ---------------------------------------------------------------------------

func TestAddGroupMemberRequest_Validate_Valid(t *testing.T) {
	r := &AddGroupMemberRequest{UserID: uuid.New().String()}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestAddGroupMemberRequest_Validate_EmptyUserID(t *testing.T) {
	r := &AddGroupMemberRequest{UserID: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty user_id")
	}
}

func TestAddGroupMemberRequest_Validate_InvalidUUID(t *testing.T) {
	r := &AddGroupMemberRequest{UserID: "not-a-uuid"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid UUID")
	}
}

// ---------------------------------------------------------------------------
// CreateDeviceRequest
// ---------------------------------------------------------------------------

func TestCreateDeviceRequest_Validate_Valid(t *testing.T) {
	r := &CreateDeviceRequest{Name: "My Phone", DeviceType: "ios"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCreateDeviceRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateDeviceRequest{Name: "", DeviceType: "web"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateDeviceRequest_Validate_InvalidType(t *testing.T) {
	r := &CreateDeviceRequest{Name: "Device", DeviceType: "windows"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid device type")
	}
}

func TestCreateDeviceRequest_Validate_DefaultType(t *testing.T) {
	r := &CreateDeviceRequest{Name: "Device", DeviceType: ""}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v, empty device type should default to web", err)
	}
	if r.DeviceType != "web" {
		t.Errorf("DeviceType = %q, want %q", r.DeviceType, "web")
	}
}

// ---------------------------------------------------------------------------
// UpdateDeviceRequest
// ---------------------------------------------------------------------------

func TestUpdateDeviceRequest_Validate_Valid(t *testing.T) {
	name := "New Name"
	r := &UpdateDeviceRequest{Name: &name}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestUpdateDeviceRequest_Validate_EmptyName(t *testing.T) {
	name := "   "
	r := &UpdateDeviceRequest{Name: &name}
	if err := r.Validate(); err == nil {
		t.Error("expected error for whitespace-only name")
	}
}

func TestUpdateDeviceRequest_Validate_LongName(t *testing.T) {
	name := "abcdefghijklmnopqrstuvwxyz-abcdefghijklmnopqrstuvwxyz"
	r := &UpdateDeviceRequest{Name: &name}
	if err := r.Validate(); err == nil {
		t.Error("expected error for name > 50 chars")
	}
}

// ---------------------------------------------------------------------------
// CreateMapConfigRequest
// ---------------------------------------------------------------------------

func TestCreateMapConfigRequest_Validate_Valid(t *testing.T) {
	r := &CreateMapConfigRequest{Name: "OSM", TileURL: "https://tile.openstreetmap.org/{z}/{x}/{y}.png"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCreateMapConfigRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateMapConfigRequest{Name: "", TileURL: "https://example.com"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateMapConfigRequest_Validate_InvalidSourceType(t *testing.T) {
	r := &CreateMapConfigRequest{Name: "Test", SourceType: "invalid", TileURL: "https://example.com"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid source type")
	}
}

func TestCreateMapConfigRequest_Validate_NoTileURLForRemote(t *testing.T) {
	r := &CreateMapConfigRequest{Name: "Test", SourceType: "remote", TileURL: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for missing tile_url with remote source")
	}
}

func TestCreateMapConfigRequest_Validate_ZoomRange(t *testing.T) {
	min, max := 5, 3
	r := &CreateMapConfigRequest{Name: "Test", TileURL: "https://example.com", MinZoom: &min, MaxZoom: &max}
	if err := r.Validate(); err == nil {
		t.Error("expected error for min_zoom > max_zoom")
	}
}

func TestCreateMapConfigRequest_Validate_ZoomOutOfRange(t *testing.T) {
	min := -1
	r := &CreateMapConfigRequest{Name: "Test", TileURL: "https://example.com", MinZoom: &min}
	if err := r.Validate(); err == nil {
		t.Error("expected error for min_zoom < 0")
	}

	max := 25
	r = &CreateMapConfigRequest{Name: "Test", TileURL: "https://example.com", MaxZoom: &max}
	if err := r.Validate(); err == nil {
		t.Error("expected error for max_zoom > 24")
	}
}

// ---------------------------------------------------------------------------
// CreateTerrainConfigRequest
// ---------------------------------------------------------------------------

func TestCreateTerrainConfigRequest_Validate_Valid(t *testing.T) {
	r := &CreateTerrainConfigRequest{Name: "AWS Terrain", TerrainURL: "https://s3.amazonaws.com/terrain/{z}/{x}/{y}.png"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
}

func TestCreateTerrainConfigRequest_Validate_EmptyName(t *testing.T) {
	r := &CreateTerrainConfigRequest{Name: "", TerrainURL: "https://example.com"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestCreateTerrainConfigRequest_Validate_InvalidSourceType(t *testing.T) {
	r := &CreateTerrainConfigRequest{Name: "Test", SourceType: "cloud", TerrainURL: "https://example.com"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid source type")
	}
}

func TestCreateTerrainConfigRequest_Validate_InvalidEncoding(t *testing.T) {
	r := &CreateTerrainConfigRequest{Name: "Test", TerrainURL: "https://example.com", TerrainEncoding: "png"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for invalid encoding")
	}
}

func TestCreateTerrainConfigRequest_Validate_Defaults(t *testing.T) {
	r := &CreateTerrainConfigRequest{Name: "Test", TerrainURL: "https://example.com"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}
	if r.SourceType != "remote" {
		t.Errorf("SourceType default = %q, want %q", r.SourceType, "remote")
	}
	if r.TerrainEncoding != "terrarium" {
		t.Errorf("TerrainEncoding default = %q, want %q", r.TerrainEncoding, "terrarium")
	}
}

// ---------------------------------------------------------------------------
// MFA validation
// ---------------------------------------------------------------------------

func TestTOTPSetupRequest_Validate(t *testing.T) {
	r := &TOTPSetupRequest{Name: "My Authenticator"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	r = &TOTPSetupRequest{Name: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

func TestTOTPVerifyRequest_Validate(t *testing.T) {
	r := &TOTPVerifyRequest{MethodID: uuid.New(), Code: "123456"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	r = &TOTPVerifyRequest{MethodID: uuid.Nil, Code: "123456"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for nil method_id")
	}

	r = &TOTPVerifyRequest{MethodID: uuid.New(), Code: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty code")
	}

	r = &TOTPVerifyRequest{MethodID: uuid.New(), Code: "12345"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for code != 6 digits")
	}
}

func TestMFAVerifyTOTPRequest_Validate(t *testing.T) {
	r := &MFAVerifyTOTPRequest{MFAToken: "tok", Code: "123456"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	r = &MFAVerifyTOTPRequest{MFAToken: "", Code: "123456"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty mfa_token")
	}
}

func TestMFARecoveryRequest_Validate(t *testing.T) {
	r := &MFARecoveryRequest{MFAToken: "tok", Code: "RECOVERY-CODE"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	r = &MFARecoveryRequest{MFAToken: "", Code: "code"}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty mfa_token")
	}
}

func TestWebAuthnRegisterRequest_Validate(t *testing.T) {
	r := &WebAuthnRegisterRequest{Name: "My Key"}
	if err := r.Validate(); err != nil {
		t.Errorf("Validate() error = %v", err)
	}

	r = &WebAuthnRegisterRequest{Name: ""}
	if err := r.Validate(); err == nil {
		t.Error("expected error for empty name")
	}
}

// ---------------------------------------------------------------------------
// HexColorRegex
// ---------------------------------------------------------------------------

func TestHexColorRegex(t *testing.T) {
	valid := []string{"#ff0000", "#AABBCC", "#123abc", "#000000", "#FFFFFF"}
	for _, c := range valid {
		if !HexColorRegex.MatchString(c) {
			t.Errorf("HexColorRegex should match %q", c)
		}
	}

	invalid := []string{"ff0000", "#fff", "#GGGGGG", "red", "#ff000", "#ff00000", ""}
	for _, c := range invalid {
		if HexColorRegex.MatchString(c) {
			t.Errorf("HexColorRegex should not match %q", c)
		}
	}
}

// ---------------------------------------------------------------------------
// AllowedMarkerIcons
// ---------------------------------------------------------------------------

func TestAllowedMarkerIcons(t *testing.T) {
	expected := []string{"circle", "square", "triangle", "diamond", "star", "crosshair", "pentagon", "hexagon", "arrow", "plus"}
	for _, icon := range expected {
		if !AllowedMarkerIcons[icon] {
			t.Errorf("AllowedMarkerIcons should contain %q", icon)
		}
	}
	if AllowedMarkerIcons["invalid"] {
		t.Error("AllowedMarkerIcons should not contain 'invalid'")
	}
}

// ---------------------------------------------------------------------------
// Helper
// ---------------------------------------------------------------------------

func assertValidationError(t *testing.T, err error, expectedMsg string) {
	t.Helper()
	ve, ok := err.(*ValidationError)
	if !ok {
		t.Fatalf("expected *ValidationError, got %T", err)
	}
	if ve.Message != expectedMsg {
		t.Errorf("error message = %q, want %q", ve.Message, expectedMsg)
	}
}
