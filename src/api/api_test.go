package api

// NOTE: The api package uses an init() that calls config.Load() on startup.
// A .env file in this directory (src/api/.env) is loaded automatically by
// godotenv and provides the required environment variables so that init()
// succeeds without a real OBS / MediaMTX / DB deployment.
//
// TestMain then re-opens the same SQLite file, runs AutoMigrate, and replaces
// the package-level dbStreamsClient so that all handler tests operate against
// a clean, migrated schema.

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/gin-gonic/gin"

	"lanops/obs-proxy-bridge/internal/dbstreams"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var testDB *gorm.DB

func TestMain(m *testing.M) {
	gin.SetMode(gin.TestMode)

	// init() has already opened cfg.DbPath (loaded from .env).
	// Open the same file here, run AutoMigrate, and replace dbStreamsClient.
	var err error
	testDB, err = gorm.Open(sqlite.Open(cfg.DbPath), &gorm.Config{})
	if err != nil {
		panic("TestMain: failed to open test DB: " + err.Error())
	}
	if err := testDB.AutoMigrate(&dbstreams.Stream{}); err != nil {
		panic("TestMain: failed to migrate: " + err.Error())
	}
	dbStreamsClient, err = dbstreams.New(cfg, testDB)
	if err != nil {
		panic("TestMain: failed to create db client: " + err.Error())
	}

	code := m.Run()
	os.Remove(cfg.DbPath)
	os.Exit(code)
}

// clearStreams hard-deletes every stream between tests.
func clearStreams(t *testing.T) {
	t.Helper()
	testDB.Exec("DELETE FROM streams")
}

// testRouter builds a Gin router matching the production layout but without
// starting an actual TCP listener.
func testRouter() *gin.Engine {
	r := gin.New()
	authorized := r.Group("", gin.BasicAuth(gin.Accounts{
		cfg.ApiAdminUsername: cfg.ApiAdminPassword,
	}))
	authorized.GET("/streams", handleGetStreams)
	authorized.GET("/streams/:name", handleGetStreamByName)
	authorized.POST("/streams/:name/enable", handleEnableStreamByName)
	return r
}

func authHeader() (string, string) {
	return cfg.ApiAdminUsername, cfg.ApiAdminPassword
}

// ── handleGetStreams ──────────────────────────────────────────────────────────

func TestHandleGetStreams_Empty(t *testing.T) {
	clearStreams(t)
	r := testRouter()

	req := httptest.NewRequest(http.MethodGet, "/streams", nil)
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var streams []dbstreams.Stream
	json.NewDecoder(w.Body).Decode(&streams) //nolint:errcheck
	if len(streams) != 0 {
		t.Errorf("expected 0 streams, got %d", len(streams))
	}
}

func TestHandleGetStreams_WithData(t *testing.T) {
	clearStreams(t)
	dbStreamsClient.CreateStream("alpha") //nolint:errcheck
	dbStreamsClient.CreateStream("beta")  //nolint:errcheck

	r := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/streams", nil)
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var streams []dbstreams.Stream
	json.NewDecoder(w.Body).Decode(&streams) //nolint:errcheck
	if len(streams) != 2 {
		t.Errorf("expected 2 streams, got %d", len(streams))
	}
}

func TestHandleGetStreams_Unauthorized(t *testing.T) {
	clearStreams(t)
	r := testRouter()

	req := httptest.NewRequest(http.MethodGet, "/streams", nil)
	// No auth header
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// ── handleGetStreamByName ─────────────────────────────────────────────────────

func TestHandleGetStreamByName_Found(t *testing.T) {
	clearStreams(t)
	dbStreamsClient.CreateStream("find-this") //nolint:errcheck

	r := testRouter()
	req := httptest.NewRequest(http.MethodGet, "/streams/find-this", nil)
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Handler returns StatusFound (302)
	if w.Code != http.StatusFound {
		t.Errorf("expected 302, got %d", w.Code)
	}

	var stream dbstreams.Stream
	json.NewDecoder(w.Body).Decode(&stream) //nolint:errcheck
	if stream.Name != "find-this" {
		t.Errorf("expected 'find-this', got %q", stream.Name)
	}
}

func TestHandleGetStreamByName_NotFound(t *testing.T) {
	clearStreams(t)
	r := testRouter()

	req := httptest.NewRequest(http.MethodGet, "/streams/ghost", nil)
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// ── handleEnableStreamByName ─────────────────────────────────────────────────

func TestHandleEnableStreamByName_Enable(t *testing.T) {
	clearStreams(t)
	s, _ := dbStreamsClient.CreateStream("toggle-stream")
	// Disable first so we can test enabling.
	dbStreamsClient.UpdateStream(s, map[string]interface{}{"enabled": false}) //nolint:errcheck

	r := testRouter()
	body, _ := json.Marshal(HandleEnableStreamByNameParams{Enabled: true})
	req := httptest.NewRequest(http.MethodPost, "/streams/toggle-stream/enable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d: %s", w.Code, w.Body.String())
	}

	var stream dbstreams.Stream
	json.NewDecoder(w.Body).Decode(&stream) //nolint:errcheck
	if !stream.Enabled {
		t.Error("expected stream to be enabled")
	}
}

func TestHandleEnableStreamByName_Disable(t *testing.T) {
	clearStreams(t)
	dbStreamsClient.CreateStream("disable-me") //nolint:errcheck

	r := testRouter()
	body, _ := json.Marshal(HandleEnableStreamByNameParams{Enabled: false})
	req := httptest.NewRequest(http.MethodPost, "/streams/disable-me/enable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}

	var stream dbstreams.Stream
	json.NewDecoder(w.Body).Decode(&stream) //nolint:errcheck
	if stream.Enabled {
		t.Error("expected stream to be disabled")
	}
}

func TestHandleEnableStreamByName_NotFound(t *testing.T) {
	clearStreams(t)
	r := testRouter()

	body, _ := json.Marshal(HandleEnableStreamByNameParams{Enabled: true})
	req := httptest.NewRequest(http.MethodPost, "/streams/nonexistent/enable", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

func TestHandleEnableStreamByName_BadJSON(t *testing.T) {
	clearStreams(t)
	dbStreamsClient.CreateStream("json-test") //nolint:errcheck

	r := testRouter()
	req := httptest.NewRequest(http.MethodPost, "/streams/json-test/enable", bytes.NewReader([]byte("not json")))
	req.Header.Set("Content-Type", "application/json")
	user, pass := authHeader()
	req.SetBasicAuth(user, pass)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

// ── HandleEnableStreamByNameParams ───────────────────────────────────────────

func TestHandleEnableStreamByNameParams_JSONRoundtrip(t *testing.T) {
	p := HandleEnableStreamByNameParams{Enabled: true}
	b, err := json.Marshal(p)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}
	var p2 HandleEnableStreamByNameParams
	if err := json.Unmarshal(b, &p2); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}
	if p2.Enabled != true {
		t.Error("expected Enabled=true after round-trip")
	}
}
