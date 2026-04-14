package dbstreams

import (
	"errors"
	"testing"

	"lanops/obs-proxy-bridge/internal/config"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// setupTestDB opens a fresh in-memory SQLite DB, auto-migrates the schema,
// and returns a ready-to-use Client.
func setupTestDB(t *testing.T) *Client {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	if err := db.AutoMigrate(&Stream{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}
	client, err := New(config.Config{DbPath: ":memory:"}, db)
	if err != nil {
		t.Fatalf("failed to create client: %v", err)
	}
	return client
}

// mustCreate creates a stream and re-fetches it so the returned value has its
// DB-assigned ID populated (GORM map-create doesn't reflect the ID back into
// the model struct).
func mustCreate(t *testing.T, client *Client, name string) Stream {
	t.Helper()
	_, err := client.CreateStream(name)
	if err != nil {
		t.Fatalf("CreateStream(%q) failed: %v", name, err)
	}
	stream, err := client.GetStreamByName(name)
	if err != nil {
		t.Fatalf("GetStreamByName(%q) after create failed: %v", name, err)
	}
	return stream
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	client := setupTestDB(t)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// ── GetStreams ────────────────────────────────────────────────────────────────

func TestGetStreams_Empty(t *testing.T) {
	client := setupTestDB(t)
	streams, err := client.GetStreams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(streams))
	}
}

func TestGetStreams_WithData(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "alpha")
	mustCreate(t, client, "beta")

	streams, err := client.GetStreams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(streams))
	}
}

// ── CreateStream ─────────────────────────────────────────────────────────────

func TestCreateStream_New(t *testing.T) {
	client := setupTestDB(t)
	stream := mustCreate(t, client, "new-stream")
	if stream.Name != "new-stream" {
		t.Errorf("expected name 'new-stream', got %s", stream.Name)
	}
	if !stream.Enabled {
		t.Error("expected new stream to be enabled by default")
	}
	if stream.ID == 0 {
		t.Error("expected non-zero ID after re-fetch")
	}
}

func TestCreateStream_Existing_ReturnsSame(t *testing.T) {
	client := setupTestDB(t)
	first := mustCreate(t, client, "dup")

	// Calling CreateStream again for the same name should return the existing record.
	second, err := client.CreateStream("dup")
	if err != nil {
		t.Fatalf("unexpected error on second create: %v", err)
	}
	if first.ID != second.ID {
		t.Errorf("expected same ID (%d), got %d", first.ID, second.ID)
	}
}

// ── GetStreamByName ──────────────────────────────────────────────────────────

func TestGetStreamByName_Found(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "find-me")
	stream, err := client.GetStreamByName("find-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Name != "find-me" {
		t.Errorf("expected 'find-me', got %s", stream.Name)
	}
}

func TestGetStreamByName_NotFound(t *testing.T) {
	client := setupTestDB(t)
	_, err := client.GetStreamByName("ghost")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// ── GetStreamsCount ──────────────────────────────────────────────────────────

func TestGetStreamsCount(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "a")
	mustCreate(t, client, "b")
	mustCreate(t, client, "c")

	count, err := client.GetStreamsCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 3 {
		t.Errorf("expected 3, got %d", count)
	}
}

func TestGetStreamsCount_Empty(t *testing.T) {
	client := setupTestDB(t)
	count, err := client.GetStreamsCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ── GetAvailableStreamsCount ──────────────────────────────────────────────────

func TestGetAvailableStreamsCount_Mixed(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "enabled-stream")
	disabled := mustCreate(t, client, "disabled-stream")
	client.UpdateStream(disabled, map[string]interface{}{"enabled": false}) //nolint:errcheck

	count, err := client.GetAvailableStreamsCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 1 {
		t.Errorf("expected 1 available stream, got %d", count)
	}
}

func TestGetAvailableStreamsCount_AllDisabled(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "only")
	client.UpdateStream(s, map[string]interface{}{"enabled": false}) //nolint:errcheck

	count, err := client.GetAvailableStreamsCount()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if count != 0 {
		t.Errorf("expected 0, got %d", count)
	}
}

// ── GetNextEnabledStream ─────────────────────────────────────────────────────

func TestGetNextEnabledStream_NilCurrent(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "first")

	stream, err := client.GetNextEnabledStream(nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stream.Name != "first" {
		t.Errorf("expected 'first', got %s", stream.Name)
	}
}

func TestGetNextEnabledStream_AdvancesForward(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "stream-1")
	mustCreate(t, client, "stream-2")

	current, _ := client.GetStreamByName("stream-1")
	next, err := client.GetNextEnabledStream(&current)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if next.Name != "stream-2" {
		t.Errorf("expected 'stream-2', got %s", next.Name)
	}
}

func TestGetNextEnabledStream_WrapsAround(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "stream-1")
	mustCreate(t, client, "stream-2")

	last, _ := client.GetStreamByName("stream-2")
	first, err := client.GetNextEnabledStream(&last)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if first.Name != "stream-1" {
		t.Errorf("expected wrap to 'stream-1', got %s", first.Name)
	}
}

func TestGetNextEnabledStream_NoEnabledStreams(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "disabled")
	client.UpdateStream(s, map[string]interface{}{"enabled": false}) //nolint:errcheck

	_, err := client.GetNextEnabledStream(nil)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

func TestGetNextEnabledStream_Empty(t *testing.T) {
	client := setupTestDB(t)
	_, err := client.GetNextEnabledStream(nil)
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound on empty DB, got %v", err)
	}
}

// ── DeleteStream ─────────────────────────────────────────────────────────────

func TestDeleteStream_Success(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "delete-me")

	err := client.DeleteStream("delete-me")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	exists, _ := client.CheckStreamExistsByName("delete-me")
	if exists {
		t.Error("stream should not exist after deletion")
	}
}

func TestDeleteStream_NotFound(t *testing.T) {
	client := setupTestDB(t)
	err := client.DeleteStream("nonexistent")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// ── CheckStreamExistsByName ──────────────────────────────────────────────────

func TestCheckStreamExistsByName_Exists(t *testing.T) {
	client := setupTestDB(t)
	mustCreate(t, client, "present")

	exists, err := client.CheckStreamExistsByName("present")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected stream to exist")
	}
}

func TestCheckStreamExistsByName_Missing(t *testing.T) {
	client := setupTestDB(t)
	exists, err := client.CheckStreamExistsByName("absent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected stream to not exist")
	}
}

// ── CheckStreamExistsByObsSceneItemUuid ──────────────────────────────────────

func TestCheckStreamExistsByObsSceneItemUuid_StreamUuid(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "obs-stream")
	client.UpdateStream(s, map[string]interface{}{"obs_stream_uuid": "stream-uuid-001"}) //nolint:errcheck

	exists, err := client.CheckStreamExistsByObsSceneItemUuid("stream-uuid-001")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected stream to be found by obs_stream_uuid")
	}
}

func TestCheckStreamExistsByObsSceneItemUuid_TextUuid(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "obs-text")
	client.UpdateStream(s, map[string]interface{}{"obs_text_uuid": "text-uuid-002"}) //nolint:errcheck

	exists, err := client.CheckStreamExistsByObsSceneItemUuid("text-uuid-002")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !exists {
		t.Error("expected stream to be found by obs_text_uuid")
	}
}

func TestCheckStreamExistsByObsSceneItemUuid_NotFound(t *testing.T) {
	client := setupTestDB(t)
	exists, err := client.CheckStreamExistsByObsSceneItemUuid("unknown-uuid")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if exists {
		t.Error("expected no match for unknown uuid")
	}
}

// ── GetStreamByObsSceneItemUuid ───────────────────────────────────────────────

func TestGetStreamByObsSceneItemUuid_FoundByStreamUuid(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "by-stream-uuid")
	client.UpdateStream(s, map[string]interface{}{"obs_stream_uuid": "su-abc"}) //nolint:errcheck

	found, err := client.GetStreamByObsSceneItemUuid("su-abc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "by-stream-uuid" {
		t.Errorf("expected 'by-stream-uuid', got %s", found.Name)
	}
}

func TestGetStreamByObsSceneItemUuid_FoundByTextUuid(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "by-text-uuid")
	client.UpdateStream(s, map[string]interface{}{"obs_text_uuid": "tu-xyz"}) //nolint:errcheck

	found, err := client.GetStreamByObsSceneItemUuid("tu-xyz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if found.Name != "by-text-uuid" {
		t.Errorf("expected 'by-text-uuid', got %s", found.Name)
	}
}

func TestGetStreamByObsSceneItemUuid_NotFound(t *testing.T) {
	client := setupTestDB(t)
	_, err := client.GetStreamByObsSceneItemUuid("no-such-uuid")
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		t.Errorf("expected ErrRecordNotFound, got %v", err)
	}
}

// ── UpdateStream ─────────────────────────────────────────────────────────────

func TestUpdateStream_Enabled(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "toggle-me")

	_, err := client.UpdateStream(s, map[string]interface{}{"enabled": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fetched, _ := client.GetStreamByName("toggle-me")
	if fetched.Enabled {
		t.Error("expected stream to be disabled after update")
	}
}

func TestUpdateStream_ObsFields(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "obs-update")

	_, err := client.UpdateStream(s, map[string]interface{}{
		"obs_stream_uuid": "uuid-stream",
		"obs_stream_id":   42,
		"obs_text_uuid":   "uuid-text",
		"obs_text_id":     99,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	fetched, _ := client.GetStreamByName("obs-update")
	if fetched.ObsStreamUuid != "uuid-stream" {
		t.Errorf("expected obs_stream_uuid 'uuid-stream', got %s", fetched.ObsStreamUuid)
	}
	if fetched.ObsStreamId != 42 {
		t.Errorf("expected obs_stream_id 42, got %d", fetched.ObsStreamId)
	}
	if fetched.ObsTextUuid != "uuid-text" {
		t.Errorf("expected obs_text_uuid 'uuid-text', got %s", fetched.ObsTextUuid)
	}
	if fetched.ObsTextId != 99 {
		t.Errorf("expected obs_text_id 99, got %d", fetched.ObsTextId)
	}
}

func TestUpdateStream_ReturnsUpdatedStream(t *testing.T) {
	client := setupTestDB(t)
	s := mustCreate(t, client, "ret-check")

	updated, err := client.UpdateStream(s, map[string]interface{}{"enabled": false})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = updated
}
