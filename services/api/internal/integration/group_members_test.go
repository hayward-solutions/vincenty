package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/vincenty/api/internal/testutil"
)

func TestGroupMembers_AddAndList(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Member Test Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "memberuser1", "member1@test.local", "Password123!", false)

	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	// List members
	resp := e.DoJSON(t, "GET", fmt.Sprintf("/api/v1/groups/%s/members", groupID), nil, adminTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var members []struct {
		UserID       string `json:"user_id"`
		CanRead      bool   `json:"can_read"`
		CanWrite     bool   `json:"can_write"`
		IsGroupAdmin bool   `json:"is_group_admin"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&members); err != nil {
		t.Fatalf("decode: %v", err)
	}
	found := false
	for _, m := range members {
		if m.UserID == userID.String() {
			found = true
			if !m.CanRead {
				t.Error("expected can_read=true")
			}
			if !m.CanWrite {
				t.Error("expected can_write=true")
			}
		}
	}
	if !found {
		t.Error("added member not found in member list")
	}
}

func TestGroupMembers_UpdateRole(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Role Update Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "roleuser", "role@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	resp := e.DoJSON(t, "PUT", fmt.Sprintf("/api/v1/groups/%s/members/%s", groupID, userID), map[string]any{
		"is_group_admin": true,
	}, adminTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestGroupMembers_Remove(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Remove Member Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "removeuser", "remove@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	resp := e.DoJSON(t, "DELETE", fmt.Sprintf("/api/v1/groups/%s/members/%s", groupID, userID), nil, adminTokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("remove member status = %d", resp.StatusCode)
	}
}

func TestGroupMembers_NonMemberCannotListMembers(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Private Group")
	e.CreateUser(t, adminTokens.AccessToken, "outsider", "outsider@test.local", "Password123!", false)
	outsiderTokens := e.Login(t, "outsider", "Password123!")

	resp := e.DoJSON(t, "GET", fmt.Sprintf("/api/v1/groups/%s/members", groupID), nil, outsiderTokens.AccessToken)
	defer resp.Body.Close()
	// Should be forbidden since user is not a member
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestGroupMembers_DuplicateAdd(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Dup Add Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "dupadduser", "dupadd@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	// Second add should fail with conflict
	resp := e.DoJSON(t, "POST", fmt.Sprintf("/api/v1/groups/%s/members", groupID), map[string]any{
		"user_id":  userID.String(),
		"can_read": true,
	}, adminTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusConflict)
}
