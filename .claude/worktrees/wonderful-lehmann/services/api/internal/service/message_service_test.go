package service

import (
	"bytes"
	"context"
	"io"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/vincenty/api/internal/model"
	"github.com/vincenty/api/internal/pubsub"
	"github.com/vincenty/api/internal/repository"
	mockrepo "github.com/vincenty/api/internal/repository/mock"
	"github.com/vincenty/api/internal/storage"
)

func TestMessageService_Send_TextMessage(t *testing.T) {
	groupID := uuid.New()
	senderID := uuid.New()
	messageID := uuid.New()

	msgRepo := &mockrepo.MessageRepo{
		CreateFn: func(ctx context.Context, msg *model.Message) error {
			msg.ID = messageID
			return nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			return &model.MessageWithUser{
				Message:  model.Message{ID: messageID, SenderID: senderID, GroupID: &groupID, MessageType: "text", CreatedAt: time.Now()},
				Username: "bob",
			}, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanWrite: true}, nil
		},
	}
	ps := pubsub.NewMockPubSub()

	svc := NewMessageService(msgRepo, groupRepo, nil, ps, newTestPermSvc())

	content := "Hello world"
	msg, err := svc.Send(context.Background(), SendMessageRequest{
		SenderID: senderID,
		GroupID:  &groupID,
		Content:  &content,
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if msg.ID != messageID {
		t.Errorf("ID = %v, want %v", msg.ID, messageID)
	}
	if len(ps.Published()) != 1 {
		t.Errorf("published = %d, want 1", len(ps.Published()))
	}
}

func TestMessageService_Send_NoTarget(t *testing.T) {
	svc := NewMessageService(nil, nil, nil, nil, nil)
	content := "test"
	_, err := svc.Send(context.Background(), SendMessageRequest{Content: &content})
	if err == nil {
		t.Fatal("expected error for missing target")
	}
}

func TestMessageService_Send_BothTargets(t *testing.T) {
	svc := NewMessageService(nil, nil, nil, nil, nil)
	gid := uuid.New()
	rid := uuid.New()
	content := "test"
	_, err := svc.Send(context.Background(), SendMessageRequest{
		GroupID:     &gid,
		RecipientID: &rid,
		Content:     &content,
	})
	if err == nil {
		t.Fatal("expected error for both targets")
	}
}

func TestMessageService_Send_NoContentOrFiles(t *testing.T) {
	svc := NewMessageService(nil, nil, nil, nil, nil)
	gid := uuid.New()
	_, err := svc.Send(context.Background(), SendMessageRequest{GroupID: &gid})
	if err == nil {
		t.Fatal("expected error for no content or files")
	}
}

func TestMessageService_Send_NoWriteAccess(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanWrite: false}, nil
		},
	}
	svc := NewMessageService(nil, groupRepo, nil, nil, newTestPermSvc())

	content := "test"
	_, err := svc.Send(context.Background(), SendMessageRequest{
		SenderID: uuid.New(),
		GroupID:  &groupID,
		Content:  &content,
	})
	if err == nil {
		t.Fatal("expected forbidden error for no write access")
	}
}

func TestMessageService_Send_FileTooLarge(t *testing.T) {
	groupID := uuid.New()
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanWrite: true}, nil
		},
	}
	svc := NewMessageService(nil, groupRepo, nil, nil, newTestPermSvc())

	_, err := svc.Send(context.Background(), SendMessageRequest{
		SenderID: uuid.New(),
		GroupID:  &groupID,
		Files:    []FileUpload{{Filename: "big.zip", Size: 30 * 1024 * 1024}},
	})
	if err == nil {
		t.Fatal("expected error for file exceeding size limit")
	}
}

func TestMessageService_Send_WithAttachment(t *testing.T) {
	groupID := uuid.New()
	senderID := uuid.New()
	messageID := uuid.New()

	msgRepo := &mockrepo.MessageRepo{
		CreateFn: func(ctx context.Context, msg *model.Message) error {
			msg.ID = messageID
			return nil
		},
		CreateAttachmentFn: func(ctx context.Context, att *model.Attachment) error {
			att.ID = uuid.New()
			return nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			return &model.MessageWithUser{
				Message: model.Message{ID: messageID, MessageType: "file", CreatedAt: time.Now()},
			}, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanWrite: true}, nil
		},
	}
	storageMock := &storage.MockStorage{
		UploadFn: func(ctx context.Context, key string, body io.Reader, contentType string, size int64) error {
			return nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewMessageService(msgRepo, groupRepo, storageMock, ps, newTestPermSvc())

	msg, err := svc.Send(context.Background(), SendMessageRequest{
		SenderID: senderID,
		GroupID:  &groupID,
		Files: []FileUpload{
			{Filename: "photo.jpg", ContentType: "image/jpeg", Size: 1024, Body: bytes.NewReader([]byte("fake image"))},
		},
	})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if msg == nil {
		t.Fatal("expected non-nil message")
	}
}

func TestMessageService_ListGroupMessages_NonMember(t *testing.T) {
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return nil, model.ErrNotFound("group member")
		},
	}
	svc := NewMessageService(nil, groupRepo, nil, nil, newTestPermSvc())

	_, err := svc.ListGroupMessages(context.Background(), uuid.New(), uuid.New(), false, nil, 50)
	if err == nil {
		t.Fatal("expected forbidden error for non-member")
	}
}

func TestMessageService_ListGroupMessages_NoReadAccess(t *testing.T) {
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanRead: false}, nil
		},
	}
	svc := NewMessageService(nil, groupRepo, nil, nil, newTestPermSvc())

	_, err := svc.ListGroupMessages(context.Background(), uuid.New(), uuid.New(), false, nil, 50)
	if err == nil {
		t.Fatal("expected forbidden error for no read access")
	}
}

func TestMessageService_ListGroupMessages_Admin(t *testing.T) {
	groupID := uuid.New()
	expected := []model.MessageWithUser{{Message: model.Message{Content: strPtr("hi")}}}
	msgRepo := &mockrepo.MessageRepo{
		ListByGroupFn: func(ctx context.Context, gid uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
			return expected, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanRead: true}, nil
		},
	}
	svc := NewMessageService(msgRepo, groupRepo, nil, nil, newTestPermSvc())

	msgs, err := svc.ListGroupMessages(context.Background(), groupID, uuid.New(), true, nil, 50)
	if err != nil {
		t.Fatalf("ListGroupMessages() error = %v", err)
	}
	if len(msgs) != 1 {
		t.Errorf("len(msgs) = %d, want 1", len(msgs))
	}
}

func TestMessageService_ListGroupMessages_LimitClamped(t *testing.T) {
	var capturedLimit int
	msgRepo := &mockrepo.MessageRepo{
		ListByGroupFn: func(ctx context.Context, gid uuid.UUID, before *uuid.UUID, limit int) ([]model.MessageWithUser, error) {
			capturedLimit = limit
			return nil, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanRead: true}, nil
		},
	}
	svc := NewMessageService(msgRepo, groupRepo, nil, nil, newTestPermSvc())

	svc.ListGroupMessages(context.Background(), uuid.New(), uuid.New(), true, nil, 0)
	if capturedLimit != 50 {
		t.Errorf("limit = %d, want 50 (default)", capturedLimit)
	}

	svc.ListGroupMessages(context.Background(), uuid.New(), uuid.New(), true, nil, 200)
	if capturedLimit != 50 {
		t.Errorf("limit = %d, want 50 (clamped)", capturedLimit)
	}
}

func TestMessageService_DeleteMessage_Sender(t *testing.T) {
	senderID := uuid.New()
	messageID := uuid.New()
	deleted := false

	msgRepo := &mockrepo.MessageRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			return &model.MessageWithUser{Message: model.Message{ID: messageID, SenderID: senderID}}, nil
		},
		GetAttachmentObjectKeysFn: func(ctx context.Context, mid uuid.UUID) ([]string, error) {
			return []string{"attachments/key1"}, nil
		},
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	storageMock := &storage.MockStorage{
		DeleteFn: func(ctx context.Context, key string) error { return nil },
	}
	svc := NewMessageService(msgRepo, nil, storageMock, nil, nil)

	err := svc.DeleteMessage(context.Background(), messageID, senderID, false)
	if err != nil {
		t.Fatalf("DeleteMessage() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestMessageService_DeleteMessage_NonSender(t *testing.T) {
	msgRepo := &mockrepo.MessageRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			return &model.MessageWithUser{Message: model.Message{SenderID: uuid.New()}}, nil
		},
	}
	svc := NewMessageService(msgRepo, nil, nil, nil, nil)

	err := svc.DeleteMessage(context.Background(), uuid.New(), uuid.New(), false)
	if err == nil {
		t.Fatal("expected forbidden error for non-sender")
	}
}

func TestMessageService_ListDMConversations(t *testing.T) {
	dn := "Alice"
	msgRepo := &mockrepo.MessageRepo{
		ListDMPartnersFn: func(ctx context.Context, uid uuid.UUID) ([]repository.DMPartner, error) {
			return []repository.DMPartner{
				{UserID: uuid.New(), Username: "alice", DisplayName: &dn},
			}, nil
		},
	}
	svc := NewMessageService(msgRepo, nil, nil, nil, nil)

	convos, err := svc.ListDMConversations(context.Background(), uuid.New())
	if err != nil {
		t.Fatalf("ListDMConversations() error = %v", err)
	}
	if len(convos) != 1 || convos[0].DisplayName != "Alice" {
		t.Errorf("unexpected conversations: %v", convos)
	}
}

func TestIsGPXFile(t *testing.T) {
	tests := []struct {
		filename    string
		contentType string
		want        bool
	}{
		{"track.gpx", "application/octet-stream", true},
		{"TRACK.GPX", "text/xml", true},
		{"track.xml", "application/gpx+xml", true},
		{"photo.jpg", "image/jpeg", false},
		{"data.json", "application/json", false},
	}

	for _, tt := range tests {
		got := isGPXFile(tt.filename, tt.contentType)
		if got != tt.want {
			t.Errorf("isGPXFile(%q, %q) = %v, want %v", tt.filename, tt.contentType, got, tt.want)
		}
	}
}

func strPtr(s string) *string { return &s }
