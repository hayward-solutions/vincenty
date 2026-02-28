package service

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/sitaware/api/internal/model"
	"github.com/sitaware/api/internal/pubsub"
	mockrepo "github.com/sitaware/api/internal/repository/mock"
)

func TestDrawingService_Create(t *testing.T) {
	ownerID := uuid.New()
	drawingID := uuid.New()

	drawingRepo := &mockrepo.DrawingRepo{
		CreateFn: func(ctx context.Context, d *model.Drawing) error {
			d.ID = drawingID
			return nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{
				Drawing:  model.Drawing{ID: drawingID, Name: "My Drawing", OwnerID: ownerID},
				Username: "bob",
			}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	req := CreateDrawingRequest{
		Name:    "My Drawing",
		GeoJSON: json.RawMessage(`{"type":"Feature"}`),
	}
	dwu, err := svc.Create(context.Background(), ownerID, req)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if dwu.Name != "My Drawing" {
		t.Errorf("Name = %q, want %q", dwu.Name, "My Drawing")
	}
}

func TestDrawingService_Create_EmptyName(t *testing.T) {
	svc := NewDrawingService(nil, nil, nil, nil, nil)
	_, err := svc.Create(context.Background(), uuid.New(), CreateDrawingRequest{
		Name:    "",
		GeoJSON: json.RawMessage(`{"type":"Feature"}`),
	})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestDrawingService_Create_EmptyGeoJSON(t *testing.T) {
	svc := NewDrawingService(nil, nil, nil, nil, nil)
	_, err := svc.Create(context.Background(), uuid.New(), CreateDrawingRequest{
		Name: "Test",
	})
	if err == nil {
		t.Fatal("expected error for empty geojson")
	}
}

func TestDrawingService_Create_InvalidGeoJSON(t *testing.T) {
	svc := NewDrawingService(nil, nil, nil, nil, nil)
	_, err := svc.Create(context.Background(), uuid.New(), CreateDrawingRequest{
		Name:    "Test",
		GeoJSON: json.RawMessage(`{"not_type":"invalid"}`),
	})
	if err == nil {
		t.Fatal("expected error for geojson without type field")
	}
}

func TestDrawingService_Get_Owner(t *testing.T) {
	ownerID := uuid.New()
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{
				Drawing: model.Drawing{OwnerID: ownerID, Name: "Test"},
			}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	dwu, err := svc.Get(context.Background(), uuid.New(), ownerID, false)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if dwu.Name != "Test" {
		t.Errorf("Name = %q, want %q", dwu.Name, "Test")
	}
}

func TestDrawingService_Get_Admin(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{
				Drawing: model.Drawing{OwnerID: uuid.New(), Name: "Test"},
			}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	dwu, err := svc.Get(context.Background(), uuid.New(), uuid.New(), true)
	if err != nil {
		t.Fatalf("Get() error = %v", err)
	}
	if dwu.Name != "Test" {
		t.Errorf("Name = %q, want %q", dwu.Name, "Test")
	}
}

func TestDrawingService_Get_NonOwnerNoAccess(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{
				Drawing: model.Drawing{OwnerID: uuid.New()},
			}, nil
		},
		GetShareTargetsFn: func(ctx context.Context, drawingID uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
			return nil, nil, nil // no shares
		},
	}
	groupRepo := &mockrepo.GroupRepo{}
	svc := NewDrawingService(drawingRepo, nil, groupRepo, nil, nil)

	_, err := svc.Get(context.Background(), uuid.New(), uuid.New(), false)
	if err == nil {
		t.Fatal("expected forbidden error for non-owner without access")
	}
}

func TestDrawingService_ListOwn(t *testing.T) {
	ownerID := uuid.New()
	drawingRepo := &mockrepo.DrawingRepo{
		ListByOwnerFn: func(ctx context.Context, oid uuid.UUID) ([]model.DrawingWithUser, error) {
			if oid != ownerID {
				t.Errorf("ListByOwner called with %v, want %v", oid, ownerID)
			}
			return []model.DrawingWithUser{{Drawing: model.Drawing{Name: "D1"}}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	drawings, err := svc.ListOwn(context.Background(), ownerID)
	if err != nil {
		t.Fatalf("ListOwn() error = %v", err)
	}
	if len(drawings) != 1 {
		t.Errorf("len(drawings) = %d, want 1", len(drawings))
	}
}

func TestDrawingService_Delete_Owner(t *testing.T) {
	ownerID := uuid.New()
	deleted := false
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: ownerID}}, nil
		},
		DeleteFn: func(ctx context.Context, id uuid.UUID) error {
			deleted = true
			return nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), uuid.New(), ownerID, false)
	if err != nil {
		t.Fatalf("Delete() error = %v", err)
	}
	if !deleted {
		t.Error("expected Delete to be called")
	}
}

func TestDrawingService_Delete_NonOwner_Forbidden(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: uuid.New()}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	err := svc.Delete(context.Background(), uuid.New(), uuid.New(), false)
	if err == nil {
		t.Fatal("expected forbidden error for non-owner")
	}
}

func TestDrawingService_Update_Owner(t *testing.T) {
	ownerID := uuid.New()
	drawingID := uuid.New()

	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{
				Drawing: model.Drawing{
					ID:      drawingID,
					OwnerID: ownerID,
					Name:    "Old",
					GeoJSON: json.RawMessage(`{"type":"Feature"}`),
				},
			}, nil
		},
		UpdateFn: func(ctx context.Context, d *model.Drawing) error {
			return nil
		},
		GetShareTargetsFn: func(ctx context.Context, did uuid.UUID) ([]uuid.UUID, []uuid.UUID, error) {
			return nil, nil, nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewDrawingService(drawingRepo, nil, nil, ps, nil)

	newName := "New Name"
	dwu, err := svc.Update(context.Background(), drawingID, ownerID, false, UpdateDrawingRequest{Name: &newName})
	if err != nil {
		t.Fatalf("Update() error = %v", err)
	}
	// After Update, GetByID is called again to re-fetch, so name will be "Old" from mock
	// In real code it would return the updated version. We just verify no error.
	_ = dwu
}

func TestDrawingService_Update_NonOwner_Forbidden(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: uuid.New()}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	newName := "New"
	_, err := svc.Update(context.Background(), uuid.New(), uuid.New(), false, UpdateDrawingRequest{Name: &newName})
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}

func TestDrawingService_Update_EmptyName(t *testing.T) {
	ownerID := uuid.New()
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: ownerID}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	empty := ""
	_, err := svc.Update(context.Background(), uuid.New(), ownerID, false, UpdateDrawingRequest{Name: &empty})
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}

func TestDrawingService_Share_NoTarget(t *testing.T) {
	svc := NewDrawingService(nil, nil, nil, nil, nil)
	_, err := svc.Share(context.Background(), uuid.New(), uuid.New(), false, ShareDrawingRequest{})
	if err == nil {
		t.Fatal("expected error when neither group_id nor recipient_id is set")
	}
}

func TestDrawingService_Share_BothTargets(t *testing.T) {
	gid := uuid.New()
	rid := uuid.New()
	svc := NewDrawingService(nil, nil, nil, nil, nil)
	_, err := svc.Share(context.Background(), uuid.New(), uuid.New(), false, ShareDrawingRequest{
		GroupID:     &gid,
		RecipientID: &rid,
	})
	if err == nil {
		t.Fatal("expected error when both targets are set")
	}
}

func TestDrawingService_Share_NonOwner(t *testing.T) {
	ownerID := uuid.New()
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: ownerID}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	gid := uuid.New()
	_, err := svc.Share(context.Background(), uuid.New(), uuid.New(), false, ShareDrawingRequest{GroupID: &gid})
	if err == nil {
		t.Fatal("expected forbidden error when non-owner tries to share")
	}
}

func TestDrawingService_Share_ToGroup(t *testing.T) {
	ownerID := uuid.New()
	groupID := uuid.New()
	drawingID := uuid.New()
	messageID := uuid.New()

	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{ID: drawingID, OwnerID: ownerID, Name: "D1"}}, nil
		},
	}
	groupRepo := &mockrepo.GroupRepo{
		GetMemberFn: func(ctx context.Context, gid, uid uuid.UUID) (*model.GroupMember, error) {
			return &model.GroupMember{CanWrite: true}, nil
		},
	}
	messageRepo := &mockrepo.MessageRepo{
		CreateFn: func(ctx context.Context, msg *model.Message) error {
			msg.ID = messageID
			return nil
		},
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			return &model.MessageWithUser{
				Message:  model.Message{ID: messageID, GroupID: &groupID, SenderID: ownerID, MessageType: "drawing", CreatedAt: time.Now()},
				Username: "bob",
			}, nil
		},
	}
	ps := pubsub.NewMockPubSub()
	svc := NewDrawingService(drawingRepo, messageRepo, groupRepo, ps, newTestPermSvc())

	msg, err := svc.Share(context.Background(), drawingID, ownerID, false, ShareDrawingRequest{GroupID: &groupID})
	if err != nil {
		t.Fatalf("Share() error = %v", err)
	}
	if msg.ID != messageID {
		t.Errorf("msg.ID = %v, want %v", msg.ID, messageID)
	}
}

func TestDrawingService_ListShares_Owner(t *testing.T) {
	ownerID := uuid.New()
	drawingID := uuid.New()
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: ownerID}}, nil
		},
		ListSharesFn: func(ctx context.Context, did uuid.UUID) ([]model.DrawingShareInfo, error) {
			return []model.DrawingShareInfo{{MessageID: uuid.New()}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	shares, err := svc.ListShares(context.Background(), drawingID, ownerID)
	if err != nil {
		t.Fatalf("ListShares() error = %v", err)
	}
	if len(shares) != 1 {
		t.Errorf("len(shares) = %d, want 1", len(shares))
	}
}

func TestDrawingService_ListShares_NonOwner(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: uuid.New()}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	_, err := svc.ListShares(context.Background(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected forbidden error for non-owner")
	}
}

func TestDrawingService_Unshare_Owner(t *testing.T) {
	ownerID := uuid.New()
	drawingID := uuid.New()
	messageID := uuid.New()
	revoked := false
	notifCreated := false

	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{ID: drawingID, OwnerID: ownerID, Name: "D1"}}, nil
		},
		RevokeShareFn: func(ctx context.Context, mid uuid.UUID) error {
			revoked = true
			return nil
		},
	}
	messageRepo := &mockrepo.MessageRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.MessageWithUser, error) {
			gid := uuid.New()
			return &model.MessageWithUser{Message: model.Message{GroupID: &gid}}, nil
		},
		CreateFn: func(ctx context.Context, msg *model.Message) error {
			notifCreated = true
			return nil
		},
	}
	svc := NewDrawingService(drawingRepo, messageRepo, nil, nil, nil)

	err := svc.Unshare(context.Background(), drawingID, ownerID, messageID)
	if err != nil {
		t.Fatalf("Unshare() error = %v", err)
	}
	if !revoked {
		t.Error("expected RevokeShare to be called")
	}
	if !notifCreated {
		t.Error("expected notification message to be created")
	}
}

func TestDrawingService_Unshare_NonOwner(t *testing.T) {
	drawingRepo := &mockrepo.DrawingRepo{
		GetByIDFn: func(ctx context.Context, id uuid.UUID) (*model.DrawingWithUser, error) {
			return &model.DrawingWithUser{Drawing: model.Drawing{OwnerID: uuid.New()}}, nil
		},
	}
	svc := NewDrawingService(drawingRepo, nil, nil, nil, nil)

	err := svc.Unshare(context.Background(), uuid.New(), uuid.New(), uuid.New())
	if err == nil {
		t.Fatal("expected forbidden error")
	}
}
