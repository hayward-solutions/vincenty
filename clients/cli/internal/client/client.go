// Package client provides REST and WebSocket communication with the
// SitAware API server for the CLI tool.
package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"net/url"
	"strings"
	"time"

	"nhooyr.io/websocket"
)

// Client communicates with the SitAware API.
type Client struct {
	baseURL string
	token   string
	http    *http.Client
}

// New creates a new API Client.
func New(baseURL, token string) *Client {
	return &Client{
		baseURL: strings.TrimRight(baseURL, "/"),
		token:   token,
		http:    &http.Client{Timeout: 30 * time.Second},
	}
}

// DeviceResponse mirrors the server's device JSON.
type DeviceResponse struct {
	ID         string `json:"id"`
	Name       string `json:"name"`
	DeviceType string `json:"device_type"`
}

// CreateDevice creates a new device for the authenticated user and returns it.
func (c *Client) CreateDevice(ctx context.Context, name, deviceType, appVersion string) (*DeviceResponse, error) {
	body := map[string]string{
		"name":        name,
		"device_type": deviceType,
		"app_version": appVersion,
	}

	resp, err := c.doJSON(ctx, http.MethodPost, "/api/v1/users/me/devices", body)
	if err != nil {
		return nil, fmt.Errorf("create device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated {
		return nil, readAPIError(resp)
	}

	var dev DeviceResponse
	if err := json.NewDecoder(resp.Body).Decode(&dev); err != nil {
		return nil, fmt.Errorf("decode device response: %w", err)
	}
	return &dev, nil
}

// DeleteDevice deletes a device by ID.
func (c *Client) DeleteDevice(ctx context.Context, deviceID string) error {
	resp, err := c.doJSON(ctx, http.MethodDelete, "/api/v1/devices/"+deviceID, nil)
	if err != nil {
		return fmt.Errorf("delete device: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		return readAPIError(resp)
	}
	return nil
}

// WSConn wraps a WebSocket connection for sending location updates.
type WSConn struct {
	conn *websocket.Conn
}

// ConnectWS establishes a WebSocket connection to the server.
func (c *Client) ConnectWS(ctx context.Context, deviceID, appVersion string) (*WSConn, error) {
	wsURL := c.wsURL(deviceID, appVersion)
	slog.Info("connecting websocket", "url", wsURL)

	conn, _, err := websocket.Dial(ctx, wsURL, nil)
	if err != nil {
		return nil, fmt.Errorf("websocket dial: %w", err)
	}

	// Wait for the "connected" ack from the server.
	if err := waitForConnected(ctx, conn); err != nil {
		conn.Close(websocket.StatusNormalClosure, "")
		return nil, fmt.Errorf("websocket handshake: %w", err)
	}

	return &WSConn{conn: conn}, nil
}

// SendLocationUpdate sends a location_update message over the WebSocket.
func (ws *WSConn) SendLocationUpdate(ctx context.Context, msg LocationUpdate) error {
	envelope := wsEnvelope{
		Type:    "location_update",
		Payload: msg,
	}

	data, err := json.Marshal(envelope)
	if err != nil {
		return fmt.Errorf("marshal location update: %w", err)
	}

	return ws.conn.Write(ctx, websocket.MessageText, data)
}

// Close gracefully closes the WebSocket connection.
func (ws *WSConn) Close() error {
	return ws.conn.Close(websocket.StatusNormalClosure, "client disconnecting")
}

// DrainMessages reads and discards incoming WebSocket messages to prevent
// blocking. Should be run in a goroutine for the lifetime of the connection.
func (ws *WSConn) DrainMessages(ctx context.Context) {
	for {
		_, _, err := ws.conn.Read(ctx)
		if err != nil {
			return
		}
	}
}

// LocationUpdate is the payload for a location_update WebSocket message.
type LocationUpdate struct {
	DeviceID string   `json:"device_id"`
	Lat      float64  `json:"lat"`
	Lng      float64  `json:"lng"`
	Altitude *float64 `json:"altitude,omitempty"`
	Heading  *float64 `json:"heading,omitempty"`
	Speed    *float64 `json:"speed,omitempty"`
}

type wsEnvelope struct {
	Type    string `json:"type"`
	Payload any    `json:"payload"`
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func (c *Client) doJSON(ctx context.Context, method, path string, body any) (*http.Response, error) {
	var bodyReader io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		bodyReader = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, bodyReader)
	if err != nil {
		return nil, err
	}

	req.Header.Set("Authorization", "Bearer "+c.token)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	return c.http.Do(req)
}

func (c *Client) wsURL(deviceID, appVersion string) string {
	u := c.baseURL
	u = strings.Replace(u, "https://", "wss://", 1)
	u = strings.Replace(u, "http://", "ws://", 1)

	return u + "/api/v1/ws?" + url.Values{
		"token":       {c.token},
		"device_id":   {deviceID},
		"app_version": {appVersion},
	}.Encode()
}

func waitForConnected(ctx context.Context, conn *websocket.Conn) error {
	// Set a deadline for the handshake
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var env struct {
			Type string `json:"type"`
		}
		if err := json.Unmarshal(data, &env); err != nil {
			continue
		}

		if env.Type == "connected" {
			return nil
		}
		// Keep draining messages (e.g. location_snapshot) until connected
	}
}

func readAPIError(resp *http.Response) error {
	data, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("API %d: %s", resp.StatusCode, string(data))
}
