package config

import "testing"

// setAllEnvVars sets all required environment variables via t.Setenv so they
// are automatically restored after each test.
func setAllEnvVars(t *testing.T) {
	t.Helper()
	t.Setenv("DB_PATH", "/tmp/config-test.db")
	t.Setenv("OBS_WEBSOCKET_ADDRESS", "localhost:4455")
	t.Setenv("OBS_WEBSOCKET_PASSWORD", "testpassword")
	t.Setenv("OBS_PROXY_SCENE_NAME", "TestProxyScene")
	t.Setenv("MEDIAMTX_API_ADDRESS", "localhost:9997")
	t.Setenv("MEDIAMTX_RTMP_ADDRESS", "localhost:1935")
	t.Setenv("API_ADMIN_USERNAME", "testadmin")
	t.Setenv("API_ADMIN_PASSWORD", "testpassword")
	t.Setenv("API_PORT", "18080")
}

func TestLoad_AllFields(t *testing.T) {
	setAllEnvVars(t)

	cfg := Load()

	tests := []struct {
		name     string
		got      string
		expected string
	}{
		{"DbPath", cfg.DbPath, "/tmp/config-test.db"},
		{"ObsWebSocketAddress", cfg.ObsWebSocketAddress, "localhost:4455"},
		{"ObsWebSocketPassword", cfg.ObsWebSocketPassword, "testpassword"},
		{"ObsProxySceneName", cfg.ObsProxySceneName, "TestProxyScene"},
		{"MediaMtxApiAddress", cfg.MediaMtxApiAddress, "localhost:9997"},
		{"MediaMtxRtmpAddress", cfg.MediaMtxRtmpAddress, "localhost:1935"},
		{"ApiAdminUsername", cfg.ApiAdminUsername, "testadmin"},
		{"ApiAdminPassword", cfg.ApiAdminPassword, "testpassword"},
		{"ApiPort", cfg.ApiPort, "18080"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if tt.got != tt.expected {
				t.Errorf("expected %q, got %q", tt.expected, tt.got)
			}
		})
	}
}

func TestLoad_ObsProxySceneUuidNotLoaded(t *testing.T) {
	setAllEnvVars(t)
	// ObsProxySceneUuid is not loaded from env — it is set at runtime.
	cfg := Load()
	if cfg.ObsProxySceneUuid != "" {
		t.Errorf("expected empty ObsProxySceneUuid, got %q", cfg.ObsProxySceneUuid)
	}
}

func TestConfig_Struct(t *testing.T) {
	cfg := Config{
		DbPath:               "test.db",
		ObsWebSocketAddress:  "localhost:4455",
		ObsWebSocketPassword: "password",
		ObsProxySceneName:    "scene",
		ObsProxySceneUuid:    "uuid-abc-123",
		MediaMtxApiAddress:   "localhost:9997",
		MediaMtxRtmpAddress:  "localhost:1935",
		ApiAdminUsername:     "admin",
		ApiAdminPassword:     "pass",
		ApiPort:              "8080",
	}

	if cfg.DbPath != "test.db" {
		t.Errorf("unexpected DbPath: %s", cfg.DbPath)
	}
	if cfg.ObsProxySceneUuid != "uuid-abc-123" {
		t.Errorf("unexpected ObsProxySceneUuid: %s", cfg.ObsProxySceneUuid)
	}
	if cfg.ApiPort != "8080" {
		t.Errorf("unexpected ApiPort: %s", cfg.ApiPort)
	}
}

func TestLoad_ReturnsConfigType(t *testing.T) {
	setAllEnvVars(t)
	cfg := Load()
	// Verify the return type is Config (compile-time assertion via assignment)
	var _ Config = cfg
}
