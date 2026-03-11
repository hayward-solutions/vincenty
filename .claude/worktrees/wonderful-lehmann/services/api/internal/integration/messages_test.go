package integration

import (
	"encoding/json"
	"fmt"
	"net/http"
	"testing"

	"github.com/vincenty/api/internal/testutil"
)

func TestMessages_SendGroupMessage(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Msg Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "msguser1", "msg1@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "msguser1", "Password123!")

	resp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
		"content":  "Hello team!",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)

	var msg struct {
		ID      string `json:"id"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&msg); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if msg.Content != "Hello team!" {
		t.Errorf("content = %q, want %q", msg.Content, "Hello team!")
	}
}

func TestMessages_ListGroupMessages(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "List Msg Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "msglistuser", "msglist@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "msglistuser", "Password123!")

	// Send a message
	sendResp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
		"content":  "List test message",
	}, userTokens.AccessToken)
	sendResp.Body.Close()

	// List messages (returns array directly, not wrapped)
	resp := e.DoJSON(t, "GET", fmt.Sprintf("/api/v1/groups/%s/messages?limit=10", groupID), nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)

	var messages []json.RawMessage
	if err := json.NewDecoder(resp.Body).Decode(&messages); err != nil {
		t.Fatalf("decode: %v", err)
	}
	if len(messages) < 1 {
		t.Errorf("expected at least 1 message, got %d", len(messages))
	}
}

func TestMessages_SendDirectMessage(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "dmsender", "dmsender@test.local", "Password123!", false)
	recipientID := e.CreateUser(t, adminTokens.AccessToken, "dmrecipient", "dmrecip@test.local", "Password123!", false)

	senderTokens := e.Login(t, "dmsender", "Password123!")

	resp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"recipient_id": recipientID.String(),
		"content":      "Hey there!",
	}, senderTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusCreated)
}

func TestMessages_ListDMConversations(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	e.CreateUser(t, adminTokens.AccessToken, "convosender", "convosender@test.local", "Password123!", false)
	recipientID := e.CreateUser(t, adminTokens.AccessToken, "convorecip", "convorecip@test.local", "Password123!", false)

	senderTokens := e.Login(t, "convosender", "Password123!")

	// Send a DM
	sendResp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"recipient_id": recipientID.String(),
		"content":      "Starting a convo",
	}, senderTokens.AccessToken)
	sendResp.Body.Close()

	// List conversations
	resp := e.DoJSON(t, "GET", "/api/v1/messages/conversations", nil, senderTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestMessages_GetMessage(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Get Msg Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "getmsguser", "getmsg@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "getmsguser", "Password123!")

	// Send
	sendResp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
		"content":  "Get me",
	}, userTokens.AccessToken)
	var sendBody struct {
		ID string `json:"id"`
	}
	json.NewDecoder(sendResp.Body).Decode(&sendBody)
	sendResp.Body.Close()

	if sendBody.ID == "" {
		t.Fatal("failed to send message — no ID returned")
	}

	// Get by ID
	resp := e.DoJSON(t, "GET", "/api/v1/messages/"+sendBody.ID, nil, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusOK)
}

func TestMessages_DeleteMessage(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Del Msg Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "delmsguser", "delmsg@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "delmsguser", "Password123!")

	// Send
	sendResp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
		"content":  "Delete me",
	}, userTokens.AccessToken)
	var sendBody struct {
		ID string `json:"id"`
	}
	json.NewDecoder(sendResp.Body).Decode(&sendBody)
	sendResp.Body.Close()

	if sendBody.ID == "" {
		t.Fatal("failed to send message — no ID returned")
	}

	// Delete
	resp := e.DoJSON(t, "DELETE", "/api/v1/messages/"+sendBody.ID, nil, userTokens.AccessToken)
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		t.Errorf("delete message status = %d", resp.StatusCode)
	}
}

func TestMessages_NonMemberCannotSendToGroup(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Restricted Msg Group")
	e.CreateUser(t, adminTokens.AccessToken, "nonmembermsg", "nonmembermsg@test.local", "Password123!", false)

	userTokens := e.Login(t, "nonmembermsg", "Password123!")

	resp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
		"content":  "I shouldn't be able to send this",
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusForbidden)
}

func TestMessages_EmptyContent(t *testing.T) {
	e := getEnv(t)
	adminTokens := e.LoginAdmin(t)

	groupID := e.CreateGroup(t, adminTokens.AccessToken, "Empty Msg Group")
	userID := e.CreateUser(t, adminTokens.AccessToken, "emptymsguser", "emptymsg@test.local", "Password123!", false)
	e.AddGroupMember(t, adminTokens.AccessToken, groupID, userID, "member")

	userTokens := e.Login(t, "emptymsguser", "Password123!")

	resp := e.DoMultipartForm(t, "POST", "/api/v1/messages", map[string]string{
		"group_id": groupID.String(),
	}, userTokens.AccessToken)
	defer resp.Body.Close()
	testutil.RequireStatus(t, resp, http.StatusBadRequest)
}
