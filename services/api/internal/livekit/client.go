package livekit

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/livekit/protocol/auth"
	"github.com/livekit/protocol/livekit"
	lksdk "github.com/livekit/server-sdk-go/v2"
)

// Config holds LiveKit connection settings.
type Config struct {
	URL       string
	APIKey    string
	APISecret string
}

// Client wraps LiveKit SDK clients for room, egress, and ingress management.
type Client struct {
	cfg     Config
	room    *lksdk.RoomServiceClient
	egress  *lksdk.EgressClient
	ingress *lksdk.IngressClient
}

// NewClient creates a new LiveKit client wrapper.
func NewClient(cfg Config) *Client {
	return &Client{
		cfg:     cfg,
		room:    lksdk.NewRoomServiceClient(cfg.URL, cfg.APIKey, cfg.APISecret),
		egress:  lksdk.NewEgressClient(cfg.URL, cfg.APIKey, cfg.APISecret),
		ingress: lksdk.NewIngressClient(cfg.URL, cfg.APIKey, cfg.APISecret),
	}
}

// ---------------------------------------------------------------------------
// Token generation
// ---------------------------------------------------------------------------

// GenerateToken creates a LiveKit JWT for a participant to join a room.
func (c *Client) GenerateToken(identity string, roomName string, opts TokenOptions) (string, error) {
	at := auth.NewAccessToken(c.cfg.APIKey, c.cfg.APISecret)

	grant := &auth.VideoGrant{
		RoomJoin: true,
		Room:     roomName,
	}
	if opts.CanPublish != nil {
		grant.SetCanPublish(*opts.CanPublish)
	}
	if opts.CanSubscribe != nil {
		grant.SetCanSubscribe(*opts.CanSubscribe)
	}
	if opts.CanPublishData != nil {
		grant.SetCanPublishData(*opts.CanPublishData)
	}

	ttl := 1 * time.Hour
	if opts.TTL > 0 {
		ttl = opts.TTL
	}

	at.SetVideoGrant(grant).
		SetIdentity(identity).
		SetName(opts.Name).
		SetValidFor(ttl)

	return at.ToJWT()
}

// TokenOptions configures permissions for a generated token.
type TokenOptions struct {
	Name           string
	CanPublish     *bool
	CanSubscribe   *bool
	CanPublishData *bool
	TTL            time.Duration
}

// BoolPtr is a helper to create a *bool.
func BoolPtr(v bool) *bool {
	return &v
}

// ---------------------------------------------------------------------------
// Room management
// ---------------------------------------------------------------------------

// CreateRoom creates a new LiveKit room.
func (c *Client) CreateRoom(ctx context.Context, name string, maxParticipants int, emptyTimeout time.Duration) (*livekit.Room, error) {
	slog.Info("creating livekit room", "name", name, "max_participants", maxParticipants)
	return c.room.CreateRoom(ctx, &livekit.CreateRoomRequest{
		Name:            name,
		EmptyTimeout:    uint32(emptyTimeout.Seconds()),
		MaxParticipants: uint32(maxParticipants),
	})
}

// DeleteRoom deletes a LiveKit room.
func (c *Client) DeleteRoom(ctx context.Context, name string) error {
	slog.Info("deleting livekit room", "name", name)
	_, err := c.room.DeleteRoom(ctx, &livekit.DeleteRoomRequest{Room: name})
	return err
}

// ListParticipants lists participants in a room.
func (c *Client) ListParticipants(ctx context.Context, roomName string) ([]*livekit.ParticipantInfo, error) {
	res, err := c.room.ListParticipants(ctx, &livekit.ListParticipantsRequest{Room: roomName})
	if err != nil {
		return nil, err
	}
	return res.Participants, nil
}

// RemoveParticipant removes a participant from a room.
func (c *Client) RemoveParticipant(ctx context.Context, roomName, identity string) error {
	_, err := c.room.RemoveParticipant(ctx, &livekit.RoomParticipantIdentity{
		Room:     roomName,
		Identity: identity,
	})
	return err
}

// MuteTrack server-side mutes or unmutes a participant's track.
func (c *Client) MuteTrack(ctx context.Context, roomName, identity, trackSID string, muted bool) error {
	_, err := c.room.MutePublishedTrack(ctx, &livekit.MuteRoomTrackRequest{
		Room:     roomName,
		Identity: identity,
		TrackSid: trackSID,
		Muted:    muted,
	})
	return err
}

// ---------------------------------------------------------------------------
// Egress (recording)
// ---------------------------------------------------------------------------

// StartRoomRecording starts recording a room to S3.
func (c *Client) StartRoomRecording(ctx context.Context, roomName string, s3 S3Config) (*livekit.EgressInfo, error) {
	slog.Info("starting room recording", "room", roomName)
	return c.egress.StartRoomCompositeEgress(ctx, &livekit.RoomCompositeEgressRequest{
		RoomName: roomName,
		Layout:   "grid",
		Output: &livekit.RoomCompositeEgressRequest_File{
			File: &livekit.EncodedFileOutput{
				FileType: livekit.EncodedFileType_MP4,
				Filepath: fmt.Sprintf("recordings/%s-{time}.mp4", roomName),
				Output: &livekit.EncodedFileOutput_S3{
					S3: &livekit.S3Upload{
						AccessKey:      s3.AccessKey,
						Secret:         s3.SecretKey,
						Region:         s3.Region,
						Endpoint:       s3.Endpoint,
						Bucket:         s3.Bucket,
						ForcePathStyle: s3.ForcePathStyle,
					},
				},
			},
		},
	})
}

// StartTrackRecording starts recording a single track.
func (c *Client) StartTrackRecording(ctx context.Context, roomName, trackID string, s3 S3Config) (*livekit.EgressInfo, error) {
	slog.Info("starting track recording", "room", roomName, "track", trackID)
	return c.egress.StartTrackEgress(ctx, &livekit.TrackEgressRequest{
		RoomName: roomName,
		TrackId:  trackID,
		Output: &livekit.TrackEgressRequest_File{
			File: &livekit.DirectFileOutput{
				Filepath: fmt.Sprintf("recordings/%s-%s-{time}", roomName, trackID),
				Output: &livekit.DirectFileOutput_S3{
					S3: &livekit.S3Upload{
						AccessKey:      s3.AccessKey,
						Secret:         s3.SecretKey,
						Region:         s3.Region,
						Endpoint:       s3.Endpoint,
						Bucket:         s3.Bucket,
						ForcePathStyle: s3.ForcePathStyle,
					},
				},
			},
		},
	})
}

// StopRecording stops an active egress.
func (c *Client) StopRecording(ctx context.Context, egressID string) (*livekit.EgressInfo, error) {
	slog.Info("stopping recording", "egress_id", egressID)
	return c.egress.StopEgress(ctx, &livekit.StopEgressRequest{EgressId: egressID})
}

// ---------------------------------------------------------------------------
// Ingress (external feed ingest)
// ---------------------------------------------------------------------------

// CreateRTMPIngress creates an RTMP ingress for an external feed.
func (c *Client) CreateRTMPIngress(ctx context.Context, roomName, participantIdentity, participantName string) (*livekit.IngressInfo, error) {
	slog.Info("creating RTMP ingress", "room", roomName, "identity", participantIdentity)
	return c.ingress.CreateIngress(ctx, &livekit.CreateIngressRequest{
		InputType:           livekit.IngressInput_RTMP_INPUT,
		Name:                fmt.Sprintf("feed-%s", participantIdentity),
		RoomName:            roomName,
		ParticipantIdentity: participantIdentity,
		ParticipantName:     participantName,
		EnableTranscoding:   boolVal(true),
	})
}

// CreateWHIPIngress creates a WHIP ingress for an external feed.
func (c *Client) CreateWHIPIngress(ctx context.Context, roomName, participantIdentity, participantName string) (*livekit.IngressInfo, error) {
	slog.Info("creating WHIP ingress", "room", roomName, "identity", participantIdentity)
	return c.ingress.CreateIngress(ctx, &livekit.CreateIngressRequest{
		InputType:           livekit.IngressInput_WHIP_INPUT,
		Name:                fmt.Sprintf("feed-%s", participantIdentity),
		RoomName:            roomName,
		ParticipantIdentity: participantIdentity,
		ParticipantName:     participantName,
	})
}

// DeleteIngress removes an ingress.
func (c *Client) DeleteIngress(ctx context.Context, ingressID string) error {
	slog.Info("deleting ingress", "ingress_id", ingressID)
	_, err := c.ingress.DeleteIngress(ctx, &livekit.DeleteIngressRequest{IngressId: ingressID})
	return err
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

// S3Config holds S3/Minio settings for recording output.
type S3Config struct {
	Endpoint       string
	AccessKey      string
	SecretKey      string
	Bucket         string
	Region         string
	ForcePathStyle bool
}

// RoomName generates a unique LiveKit room name for a SitAware media room.
func RoomName(roomType string, id uuid.UUID) string {
	return fmt.Sprintf("sa-%s-%s", roomType, id.String()[:8])
}

func boolVal(v bool) *bool {
	return &v
}
