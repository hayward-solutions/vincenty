package model

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
)

// ---------------------------------------------------------------------------
// User.ToResponse
// ---------------------------------------------------------------------------

func TestUser_ToResponse(t *testing.T) {
	displayName := "John Doe"
	avatarURL := "avatars/123/avatar.jpg"
	id := uuid.New()
	now := time.Now()

	u := &User{
		ID:           id,
		Username:     "johndoe",
		Email:        "john@example.com",
		PasswordHash: "should-not-appear",
		DisplayName:  &displayName,
		AvatarURL:    &avatarURL,
		MarkerIcon:   "circle",
		MarkerColor:  "#ff0000",
		IsAdmin:      true,
		IsActive:     true,
		MFAEnabled:   false,
		CreatedAt:    now,
		UpdatedAt:    now,
	}

	resp := u.ToResponse()

	if resp.ID != id {
		t.Errorf("ID = %v, want %v", resp.ID, id)
	}
	if resp.Username != "johndoe" {
		t.Errorf("Username = %q, want %q", resp.Username, "johndoe")
	}
	if resp.DisplayName != "John Doe" {
		t.Errorf("DisplayName = %q, want %q", resp.DisplayName, "John Doe")
	}
	if resp.AvatarURL != "avatars/123/avatar.jpg" {
		t.Errorf("AvatarURL = %q, want %q", resp.AvatarURL, "avatars/123/avatar.jpg")
	}
	if resp.MarkerIcon != "circle" {
		t.Errorf("MarkerIcon = %q, want %q", resp.MarkerIcon, "circle")
	}
	if !resp.IsAdmin {
		t.Error("IsAdmin should be true")
	}

	// Ensure JSON marshal works and doesn't include password hash
	data, _ := json.Marshal(resp)
	jsonStr := string(data)
	if containsStr(jsonStr, "should-not-appear") {
		t.Error("JSON should not contain password hash")
	}
}

func TestUser_ToResponse_NilOptionals(t *testing.T) {
	u := &User{
		ID:          uuid.New(),
		Username:    "minimal",
		Email:       "min@example.com",
		DisplayName: nil,
		AvatarURL:   nil,
	}

	resp := u.ToResponse()
	if resp.DisplayName != "" {
		t.Errorf("DisplayName = %q, want empty string for nil", resp.DisplayName)
	}
	if resp.AvatarURL != "" {
		t.Errorf("AvatarURL = %q, want empty string for nil", resp.AvatarURL)
	}
}

// ---------------------------------------------------------------------------
// Group.ToResponse
// ---------------------------------------------------------------------------

func TestGroup_ToResponse(t *testing.T) {
	desc := "A test group"
	creatorID := uuid.New()
	g := &Group{
		ID:          uuid.New(),
		Name:        "Test Group",
		Description: &desc,
		MarkerIcon:  "star",
		MarkerColor: "#00ff00",
		CreatedBy:   &creatorID,
	}

	resp := g.ToResponse(5)

	if resp.Name != "Test Group" {
		t.Errorf("Name = %q, want %q", resp.Name, "Test Group")
	}
	if resp.Description != "A test group" {
		t.Errorf("Description = %q, want %q", resp.Description, "A test group")
	}
	if resp.MemberCount != 5 {
		t.Errorf("MemberCount = %d, want %d", resp.MemberCount, 5)
	}
}

func TestGroup_ToResponse_NilDescription(t *testing.T) {
	g := &Group{ID: uuid.New(), Name: "Test", Description: nil}
	resp := g.ToResponse(0)
	if resp.Description != "" {
		t.Errorf("Description = %q, want empty string", resp.Description)
	}
}

// ---------------------------------------------------------------------------
// MessageWithUser.ToResponse
// ---------------------------------------------------------------------------

func TestMessageWithUser_ToResponse(t *testing.T) {
	content := "Hello world"
	displayName := "John"
	groupID := uuid.New()

	m := &MessageWithUser{
		Message: Message{
			ID:          uuid.New(),
			SenderID:    uuid.New(),
			GroupID:     &groupID,
			Content:     &content,
			MessageType: "text",
			CreatedAt:   time.Now(),
		},
		Username:    "johndoe",
		DisplayName: &displayName,
		Attachments: []Attachment{
			{ID: uuid.New(), Filename: "photo.jpg", ContentType: "image/jpeg", SizeBytes: 1024},
		},
	}

	resp := m.ToResponse()
	if resp.Content != "Hello world" {
		t.Errorf("Content = %q, want %q", resp.Content, "Hello world")
	}
	if resp.DisplayName != "John" {
		t.Errorf("DisplayName = %q, want %q", resp.DisplayName, "John")
	}
	if len(resp.Attachments) != 1 {
		t.Fatalf("Attachments len = %d, want 1", len(resp.Attachments))
	}
	if resp.Attachments[0].Filename != "photo.jpg" {
		t.Errorf("attachment filename = %q", resp.Attachments[0].Filename)
	}
}

func TestMessageWithUser_ToResponse_NilOptionals(t *testing.T) {
	m := &MessageWithUser{
		Message:     Message{ID: uuid.New(), SenderID: uuid.New(), Content: nil, MessageType: "text"},
		Username:    "test",
		DisplayName: nil,
	}

	resp := m.ToResponse()
	if resp.Content != "" {
		t.Errorf("Content = %q, want empty", resp.Content)
	}
	if resp.DisplayName != "" {
		t.Errorf("DisplayName = %q, want empty", resp.DisplayName)
	}
	if resp.Attachments == nil {
		t.Error("Attachments should be empty slice, not nil")
	}
}

// ---------------------------------------------------------------------------
// Device.ToResponse
// ---------------------------------------------------------------------------

func TestDevice_ToResponse(t *testing.T) {
	ua := "Mozilla/5.0"
	uid := "device-unique-id"
	now := time.Now()

	d := &Device{
		ID:         uuid.New(),
		UserID:     uuid.New(),
		Name:       "My Browser",
		DeviceType: "web",
		DeviceUID:  &uid,
		UserAgent:  &ua,
		IsPrimary:  true,
		LastSeenAt: &now,
	}

	resp := d.ToResponse()
	if resp.Name != "My Browser" {
		t.Errorf("Name = %q", resp.Name)
	}
	if resp.DeviceUID != "device-unique-id" {
		t.Errorf("DeviceUID = %q", resp.DeviceUID)
	}
	if resp.UserAgent != "Mozilla/5.0" {
		t.Errorf("UserAgent = %q", resp.UserAgent)
	}
	if !resp.IsPrimary {
		t.Error("IsPrimary should be true")
	}
}

func TestDevice_ToResponse_NilOptionals(t *testing.T) {
	d := &Device{ID: uuid.New(), UserID: uuid.New(), Name: "Test", DeviceType: "web"}
	resp := d.ToResponse()
	if resp.DeviceUID != "" {
		t.Errorf("DeviceUID = %q, want empty", resp.DeviceUID)
	}
	if resp.UserAgent != "" {
		t.Errorf("UserAgent = %q, want empty", resp.UserAgent)
	}
	if resp.LastSeenAt != nil {
		t.Error("LastSeenAt should be nil")
	}
}

// ---------------------------------------------------------------------------
// CotEvent.ToResponse
// ---------------------------------------------------------------------------

func TestCotEvent_ToResponse(t *testing.T) {
	callsign := "Alpha1"
	hae := 50.5
	e := &CotEvent{
		ID:        uuid.New(),
		EventUID:  "TEST-UID",
		EventType: "a-f-G-U-C",
		How:       "m-g",
		Callsign:  &callsign,
		Lat:       -33.8688,
		Lng:       151.2093,
		HAE:       &hae,
	}

	resp := e.ToResponse()
	if resp.Callsign != "Alpha1" {
		t.Errorf("Callsign = %q, want %q", resp.Callsign, "Alpha1")
	}
	if resp.EventUID != "TEST-UID" {
		t.Errorf("EventUID = %q", resp.EventUID)
	}
}

func TestCotEvent_ToResponse_NilCallsign(t *testing.T) {
	e := &CotEvent{ID: uuid.New(), EventUID: "test", Callsign: nil}
	resp := e.ToResponse()
	if resp.Callsign != "" {
		t.Errorf("Callsign = %q, want empty", resp.Callsign)
	}
}

// ---------------------------------------------------------------------------
// AuditLogWithUser.ToResponse
// ---------------------------------------------------------------------------

func TestAuditLogWithUser_ToResponse(t *testing.T) {
	dn := "Admin User"
	a := &AuditLogWithUser{
		AuditLog: AuditLog{
			ID:        uuid.New(),
			UserID:    uuid.New(),
			Action:    "login",
			IPAddress: "192.168.1.1",
		},
		Username:    "admin",
		DisplayName: &dn,
	}

	resp := a.ToResponse()
	if resp.Username != "admin" {
		t.Errorf("Username = %q", resp.Username)
	}
	if resp.DisplayName != "Admin User" {
		t.Errorf("DisplayName = %q", resp.DisplayName)
	}
}

func TestAuditLogWithUser_ToResponse_NilDisplayName(t *testing.T) {
	a := &AuditLogWithUser{
		AuditLog:    AuditLog{ID: uuid.New(), UserID: uuid.New()},
		Username:    "test",
		DisplayName: nil,
	}
	resp := a.ToResponse()
	if resp.DisplayName != "" {
		t.Errorf("DisplayName = %q, want empty", resp.DisplayName)
	}
}

// ---------------------------------------------------------------------------
// Error types
// ---------------------------------------------------------------------------

func TestValidationError(t *testing.T) {
	err := ErrValidation("bad input")
	if err.Error() != "bad input" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestNotFoundError(t *testing.T) {
	err := ErrNotFound("user")
	if err.Error() != "user not found" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestConflictError(t *testing.T) {
	err := ErrConflict("duplicate")
	if err.Error() != "duplicate" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestForbiddenError(t *testing.T) {
	err := ErrForbidden("access denied")
	if err.Error() != "access denied" {
		t.Errorf("Error() = %q", err.Error())
	}
}

func TestMFARequiredError(t *testing.T) {
	err := ErrMFARequired(MFAChallengeResponse{MFARequired: true, MFAToken: "tok"})
	if err.Error() != "MFA verification required" {
		t.Errorf("Error() = %q", err.Error())
	}
	if err.Challenge.MFAToken != "tok" {
		t.Errorf("Challenge.MFAToken = %q", err.Challenge.MFAToken)
	}
}

func TestMFASetupRequiredError(t *testing.T) {
	err := ErrMFASetupRequired("setup needed")
	if err.Error() != "setup needed" {
		t.Errorf("Error() = %q", err.Error())
	}
}

// ---------------------------------------------------------------------------
// MapConfig/TerrainConfig ToResponse
// ---------------------------------------------------------------------------

func TestMapConfig_ToResponse(t *testing.T) {
	tileURL := "https://tile.example.com/{z}/{x}/{y}.png"
	mc := &MapConfig{
		ID:        uuid.New(),
		Name:      "Test Map",
		TileURL:   &tileURL,
		IsDefault: true,
		IsEnabled: true,
	}

	resp := mc.ToResponse()
	if resp.TileURL != tileURL {
		t.Errorf("TileURL = %q", resp.TileURL)
	}
	if !resp.IsDefault {
		t.Error("IsDefault should be true")
	}
}

func TestMapConfig_ToResponse_NilTileURL(t *testing.T) {
	mc := &MapConfig{ID: uuid.New(), Name: "Test", TileURL: nil}
	resp := mc.ToResponse()
	if resp.TileURL != "" {
		t.Errorf("TileURL = %q, want empty", resp.TileURL)
	}
}

func TestTerrainConfig_ToResponse(t *testing.T) {
	tc := &TerrainConfig{
		ID:              uuid.New(),
		Name:            "AWS Terrain",
		TerrainURL:      "https://example.com/terrain",
		TerrainEncoding: "terrarium",
		IsDefault:       true,
	}

	resp := tc.ToResponse()
	if resp.Name != "AWS Terrain" {
		t.Errorf("Name = %q", resp.Name)
	}
	if resp.TerrainEncoding != "terrarium" {
		t.Errorf("TerrainEncoding = %q", resp.TerrainEncoding)
	}
}

// ---------------------------------------------------------------------------
// GroupMemberWithUser.ToResponse
// ---------------------------------------------------------------------------

func TestGroupMemberWithUser_ToResponse(t *testing.T) {
	dn := "Alice"
	m := &GroupMemberWithUser{
		GroupMember: GroupMember{
			ID:           uuid.New(),
			GroupID:      uuid.New(),
			UserID:       uuid.New(),
			CanRead:      true,
			CanWrite:     false,
			IsGroupAdmin: true,
		},
		Username:    "alice",
		DisplayName: &dn,
	}

	resp := m.ToResponse()
	if resp.Username != "alice" {
		t.Errorf("Username = %q", resp.Username)
	}
	if !resp.CanRead {
		t.Error("CanRead should be true")
	}
	if resp.CanWrite {
		t.Error("CanWrite should be false")
	}
	if !resp.IsGroupAdmin {
		t.Error("IsGroupAdmin should be true")
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func containsStr(s, substr string) bool {
	return len(s) >= len(substr) && searchStr(s, substr)
}

func searchStr(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
