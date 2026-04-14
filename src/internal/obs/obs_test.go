package obs

import (
	"testing"

	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// mustCreate creates a stream and re-fetches it so the returned value has its
// DB-assigned ID populated (GORM map-create doesn't reflect the ID back).
func mustCreate(t *testing.T, client *dbstreams.Client, name string) dbstreams.Stream {
	t.Helper()
	if _, err := client.CreateStream(name); err != nil {
		t.Fatalf("CreateStream(%q) failed: %v", name, err)
	}
	stream, err := client.GetStreamByName(name)
	if err != nil {
		t.Fatalf("GetStreamByName(%q) after create failed: %v", name, err)
	}
	return stream
}

// setupTestClient creates an obs.Client with a real in-memory DB but a nil
// goobs connection.  Any test that would exercise OBS WebSocket calls must
// skip itself if it reaches a goobs method.
func setupTestClient(t *testing.T) (*Client, *dbstreams.Client, chan channels.MsgCh) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	if err := db.AutoMigrate(&dbstreams.Stream{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	cfg := config.Config{
		ObsProxySceneName: "TestScene",
		ObsProxySceneUuid: "test-uuid-0000",
	}
	dbClient, _ := dbstreams.New(cfg, db)
	msgCh := make(chan channels.MsgCh, 64)
	client, err := New(cfg, dbClient, nil /* goobs – not needed for DB-only paths */, msgCh)
	if err != nil {
		t.Fatalf("failed to create obs client: %v", err)
	}
	return client, dbClient, msgCh
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	client, _, _ := setupTestClient(t)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

func TestNew_NilObsClientAccepted(t *testing.T) {
	// obs.New must not panic or error when goobs client is nil.
	cfg := config.Config{}
	msgCh := make(chan channels.MsgCh, 1)
	c, err := New(cfg, nil, nil, msgCh)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if c == nil {
		t.Fatal("expected non-nil Client")
	}
}

// ── ClientError ───────────────────────────────────────────────────────────────

func TestClientError_Fields(t *testing.T) {
	e := &ClientError{
		Err:     nil,
		Message: "test error message",
	}
	if e.Message != "test error message" {
		t.Errorf("unexpected Message: %s", e.Message)
	}
	if e.Err != nil {
		t.Errorf("expected nil Err, got %v", e.Err)
	}
}

// ── CreateStreamMediaSourceInputOutput ────────────────────────────────────────

func TestCreateStreamMediaSourceInputOutput_Fields(t *testing.T) {
	out := CreateStreamMediaSourceInputOutput{
		InputUuid:   "some-uuid",
		SceneItemId: 42,
	}
	if out.InputUuid != "some-uuid" {
		t.Errorf("expected 'some-uuid', got %s", out.InputUuid)
	}
	if out.SceneItemId != 42 {
		t.Errorf("expected 42, got %d", out.SceneItemId)
	}
}

// ── RotateActiveStream — paths that do NOT require a live OBS connection ──────

func TestRotateActiveStream_NoStreams(t *testing.T) {
	client, _, msgCh := setupTestClient(t)
	// Empty DB → streamCount = 0 → else branch (no OBS calls).
	result := client.RotateActiveStream()
	if result != nil {
		t.Errorf("expected nil ClientError, got %+v", result)
	}
	msg := <-msgCh
	if msg.Message != "No Enabled Streams found" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
	if client.obsStreams.current != nil {
		t.Error("expected current stream to be nil")
	}
	if client.obsStreams.previous != nil {
		t.Error("expected previous stream to be nil")
	}
}

func TestRotateActiveStream_AllDisabled(t *testing.T) {
	client, dbClient, msgCh := setupTestClient(t)
	s := mustCreate(t, dbClient, "disabled-only")
	dbClient.UpdateStream(s, map[string]interface{}{"enabled": false}) //nolint:errcheck

	result := client.RotateActiveStream()
	if result != nil {
		t.Errorf("expected nil ClientError, got %+v", result)
	}
	msg := <-msgCh
	if msg.Message != "No Enabled Streams found" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
}

func TestRotateActiveStream_SameStream_NoOBSCall(t *testing.T) {
	// When the only enabled stream is already current, the rotation skips
	// both if-blocks that call OBS — no live connection needed.
	client, dbClient, _ := setupTestClient(t)
	// Re-fetch ensures the stream has its DB-assigned ID so nextStream.ID == current.ID.
	s := mustCreate(t, dbClient, "only-stream")
	client.obsStreams.current = &s

	result := client.RotateActiveStream()
	if result != nil {
		t.Errorf("expected nil ClientError, got %+v", result)
	}
	if client.obsStreams.current == nil {
		t.Fatal("expected current stream to remain set")
	}
	if client.obsStreams.current.Name != "only-stream" {
		t.Errorf("expected 'only-stream', got %s", client.obsStreams.current.Name)
	}
}

func TestRotateActiveStream_CurrentRemovedFromDB_NoRemainingEnabled(t *testing.T) {
	// Current points to a stream that no longer exists in DB AND there are no
	// other enabled streams → falls through to the else branch (no OBS calls).
	client, _, msgCh := setupTestClient(t)
	ghost := &dbstreams.Stream{ID: 9999, Name: "ghost"}
	client.obsStreams.current = ghost

	result := client.RotateActiveStream()
	if result != nil {
		t.Errorf("expected nil ClientError, got %+v", result)
	}
	msg := <-msgCh
	if msg.Message != "No Enabled Streams found" {
		t.Errorf("unexpected message: %q", msg.Message)
	}
	if client.obsStreams.current != nil {
		t.Error("expected current stream to be cleared")
	}
}
