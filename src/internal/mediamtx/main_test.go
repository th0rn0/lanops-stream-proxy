package mediamtx

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// newTestClient creates a Client pointed at the given httptest server URL,
// backed by a fresh in-memory SQLite DB.
func newTestClient(t *testing.T, serverURL string) (*Client, chan channels.MsgCh, *dbstreams.Client) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to open in-memory DB: %v", err)
	}
	if err := db.AutoMigrate(&dbstreams.Stream{}); err != nil {
		t.Fatalf("failed to migrate: %v", err)
	}

	host := strings.TrimPrefix(serverURL, "http://")
	cfg := config.Config{MediaMtxApiAddress: host}

	dbClient, err := dbstreams.New(cfg, db)
	if err != nil {
		t.Fatalf("failed to create dbstreams client: %v", err)
	}

	msgCh := make(chan channels.MsgCh, 64)
	client, err := New(cfg, dbClient, msgCh)
	if err != nil {
		t.Fatalf("failed to create mediamtx client: %v", err)
	}

	return client, msgCh, dbClient
}

func serveJSON(t *testing.T, payload interface{}) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(payload)
	}))
}

// ── New ───────────────────────────────────────────────────────────────────────

func TestNew(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	defer server.Close()

	client, _, _ := newTestClient(t, server.URL)
	if client == nil {
		t.Fatal("expected non-nil client")
	}
}

// ── GetStreams ─────────────────────────────────────────────────────────────────

func TestGetStreams_Success(t *testing.T) {
	response := MediamtxListStreamsResponse{
		ItemCount: 2,
		PageCount: 1,
		Items: []MediamtxListStreamsOutput{
			{Name: "stream-a", ConfName: "stream-a", Ready: true},
			{Name: "stream-b", ConfName: "stream-b", Ready: false},
		},
	}
	server := serveJSON(t, response)
	defer server.Close()

	client, _, _ := newTestClient(t, server.URL)
	streams, err := client.GetStreams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 2 {
		t.Fatalf("expected 2 streams, got %d", len(streams))
	}
	if streams[0].Name != "stream-a" {
		t.Errorf("expected 'stream-a', got %s", streams[0].Name)
	}
	if streams[1].Name != "stream-b" {
		t.Errorf("expected 'stream-b', got %s", streams[1].Name)
	}
}

func TestGetStreams_Empty(t *testing.T) {
	server := serveJSON(t, MediamtxListStreamsResponse{Items: []MediamtxListStreamsOutput{}})
	defer server.Close()

	client, _, _ := newTestClient(t, server.URL)
	streams, err := client.GetStreams()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(streams))
	}
}

func TestGetStreams_InvalidJSON(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not valid json {{{{"))
	}))
	defer server.Close()

	client, _, _ := newTestClient(t, server.URL)
	_, err := client.GetStreams()
	if err == nil {
		t.Error("expected error for invalid JSON body")
	}
}

func TestGetStreams_ConnectionRefused(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	addr := server.URL
	server.Close() // close before making the request

	client, _, _ := newTestClient(t, addr)
	_, err := client.GetStreams()
	if err == nil {
		t.Error("expected connection error for closed server")
	}
}

func TestGetStreams_CorrectPath(t *testing.T) {
	var capturedPath string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		capturedPath = r.URL.Path
		json.NewEncoder(w).Encode(MediamtxListStreamsResponse{})
	}))
	defer server.Close()

	client, _, _ := newTestClient(t, server.URL)
	client.GetStreams() //nolint:errcheck

	if capturedPath != "/v3/paths/list" {
		t.Errorf("expected path '/v3/paths/list', got %s", capturedPath)
	}
}

// ── SyncStreams ───────────────────────────────────────────────────────────────

func TestSyncStreams_AddsNewStreams(t *testing.T) {
	response := MediamtxListStreamsResponse{
		Items: []MediamtxListStreamsOutput{
			{Name: "new-stream-1"},
			{Name: "new-stream-2"},
		},
	}
	server := serveJSON(t, response)
	defer server.Close()

	client, msgCh, dbClient := newTestClient(t, server.URL)
	clientErr := client.SyncStreams()
	if clientErr != nil {
		t.Fatalf("unexpected error: %v", clientErr.Err)
	}

	// Both streams should now be in the DB
	exists1, _ := dbClient.CheckStreamExistsByName("new-stream-1")
	exists2, _ := dbClient.CheckStreamExistsByName("new-stream-2")
	if !exists1 {
		t.Error("expected 'new-stream-1' to exist in DB")
	}
	if !exists2 {
		t.Error("expected 'new-stream-2' to exist in DB")
	}

	// Should have received "Stream Found" messages
	var msgs []string
	for len(msgCh) > 0 {
		msgs = append(msgs, (<-msgCh).Message)
	}
	if len(msgs) == 0 {
		t.Error("expected at least one info message")
	}
}

func TestSyncStreams_RemovesStaleDBStreams(t *testing.T) {
	// DB has a stream that mediamtx no longer knows about.
	response := MediamtxListStreamsResponse{
		Items: []MediamtxListStreamsOutput{
			{Name: "live-stream"},
		},
	}
	server := serveJSON(t, response)
	defer server.Close()

	client, msgCh, dbClient := newTestClient(t, server.URL)
	// Pre-populate DB with a stale stream (CreateStream result ignored — only existence matters here)
	if _, err := dbClient.CreateStream("stale-stream"); err != nil {
		t.Fatalf("failed to seed stale-stream: %v", err)
	}
	if _, err := dbClient.CreateStream("live-stream"); err != nil {
		t.Fatalf("failed to seed live-stream: %v", err)
	}

	clientErr := client.SyncStreams()
	if clientErr != nil {
		t.Fatalf("unexpected error: %v", clientErr.Err)
	}

	existsStale, _ := dbClient.CheckStreamExistsByName("stale-stream")
	existsLive, _ := dbClient.CheckStreamExistsByName("live-stream")

	if existsStale {
		t.Error("expected 'stale-stream' to be removed from DB")
	}
	if !existsLive {
		t.Error("expected 'live-stream' to remain in DB")
	}

	// Drain and check messages
	var msgs []channels.MsgCh
	for len(msgCh) > 0 {
		msgs = append(msgs, <-msgCh)
	}
	found := false
	for _, m := range msgs {
		if strings.Contains(m.Message, "Removed Stream stale-stream") {
			found = true
		}
	}
	if !found {
		t.Error("expected 'Removed Stream stale-stream' message")
	}
}

func TestSyncStreams_NoChanges(t *testing.T) {
	response := MediamtxListStreamsResponse{
		Items: []MediamtxListStreamsOutput{
			{Name: "existing"},
		},
	}
	server := serveJSON(t, response)
	defer server.Close()

	client, msgCh, dbClient := newTestClient(t, server.URL)
	dbClient.CreateStream("existing") //nolint:errcheck

	clientErr := client.SyncStreams()
	if clientErr != nil {
		t.Fatalf("unexpected error: %v", clientErr.Err)
	}

	count, _ := dbClient.GetStreamsCount()
	if count != 1 {
		t.Errorf("expected 1 stream, got %d", count)
	}

	// No messages expected (no changes)
	if len(msgCh) != 0 {
		t.Errorf("expected no messages, got %d", len(msgCh))
	}
}

func TestSyncStreams_MediamtxDown(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {}))
	addr := server.URL
	server.Close()

	client, _, _ := newTestClient(t, addr)
	clientErr := client.SyncStreams()
	if clientErr == nil {
		t.Fatal("expected ClientError when mediamtx is unreachable")
	}
	if clientErr.Message != "Error pulling MediaMTX Proxy Streams" {
		t.Errorf("unexpected error message: %s", clientErr.Message)
	}
}

func TestSyncStreams_EmptyProxy_ClearsDB(t *testing.T) {
	// Mediamtx reports no streams → DB should be wiped.
	server := serveJSON(t, MediamtxListStreamsResponse{Items: []MediamtxListStreamsOutput{}})
	defer server.Close()

	client, _, dbClient := newTestClient(t, server.URL)
	dbClient.CreateStream("orphan") //nolint:errcheck

	clientErr := client.SyncStreams()
	if clientErr != nil {
		t.Fatalf("unexpected error: %v", clientErr.Err)
	}

	count, _ := dbClient.GetStreamsCount()
	if count != 0 {
		t.Errorf("expected 0 streams after sync with empty proxy, got %d", count)
	}
}

// ── ClientError ───────────────────────────────────────────────────────────────

func TestClientError_Fields(t *testing.T) {
	err := &ClientError{
		Err:     nil,
		Message: "something went wrong",
	}
	if err.Message != "something went wrong" {
		t.Errorf("unexpected message: %s", err.Message)
	}
}

// ── Struct marshalling ────────────────────────────────────────────────────────

func TestMediamtxListStreamsOutput_Fields(t *testing.T) {
	raw := `{
		"name": "mystream",
		"confName": "mystream",
		"source": {"type": "rtmp", "id": "abc"},
		"ready": true,
		"bytesReceived": 100,
		"bytesSent": 200,
		"readers": []
	}`
	var out MediamtxListStreamsOutput
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if out.Name != "mystream" {
		t.Errorf("expected 'mystream', got %s", out.Name)
	}
	if !out.Ready {
		t.Error("expected Ready=true")
	}
	if out.Source.Type != "rtmp" {
		t.Errorf("expected source type 'rtmp', got %s", out.Source.Type)
	}
	if out.BytesReceived != 100 {
		t.Errorf("expected BytesReceived=100, got %d", out.BytesReceived)
	}
}

func TestMediamtxListStreamsResponse_Fields(t *testing.T) {
	raw := `{"itemCount": 1, "pageCount": 1, "items": [{"name": "s1"}]}`
	var resp MediamtxListStreamsResponse
	if err := json.Unmarshal([]byte(raw), &resp); err != nil {
		t.Fatalf("unexpected unmarshal error: %v", err)
	}
	if resp.ItemCount != 1 {
		t.Errorf("expected ItemCount=1, got %d", resp.ItemCount)
	}
	if len(resp.Items) != 1 {
		t.Errorf("expected 1 item, got %d", len(resp.Items))
	}
}
