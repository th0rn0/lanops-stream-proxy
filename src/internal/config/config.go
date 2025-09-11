package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

func Load() Config {
	godotenv.Load()

	// DB
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		log.Fatal("❌ DB_PATH not set in environment")
	}

	// OBS
	obsWebSocketAddress := os.Getenv("OBS_WEBSOCKET_ADDRESS")
	if obsWebSocketAddress == "" {
		log.Fatal("❌ OBS_WEBSOCKET_ADDRESS not set in environment")
	}
	obsWebSocketPassword := os.Getenv("OBS_WEBSOCKET_PASSWORD")
	if obsWebSocketPassword == "" {
		log.Fatal("❌ OBS_WEBSOCKET_PASSWORD not set in environment")
	}
	obsProxySceneName := os.Getenv("OBS_PROXY_SCENE_NAME")
	if obsProxySceneName == "" {
		log.Fatal("❌ OBS_PROXY_SCENE_NAME not set in environment")
	}

	// Media MTX
	mediaMtxApiAddress := os.Getenv("MEDIAMTX_API_ADDRESS")
	if mediaMtxApiAddress == "" {
		log.Fatal("❌ MEDIAMTX_API_ADDRESS not set in environment")
	}
	mediaMtxRtmpAddress := os.Getenv("MEDIAMTX_RTMP_ADDRESS")
	if mediaMtxRtmpAddress == "" {
		log.Fatal("❌ MEDIAMTX_RTMP_ADDRESS not set in environment")
	}

	// API
	apiAdminUsername := os.Getenv("API_ADMIN_USERNAME")
	if apiAdminUsername == "" {
		log.Fatal("❌ API_ADMIN_USERNAME not set in environment")
	}
	apiAdminPassword := os.Getenv("API_ADMIN_PASSWORD")
	if apiAdminPassword == "" {
		log.Fatal("❌ API_ADMIN_PASSWORD not set in environment")
	}
	apiPort := os.Getenv("API_PORT")
	if apiPort == "" {
		log.Fatal("❌ API_PORT not set in environment")
	}

	return Config{
		DbPath:               dbPath,
		ObsWebSocketAddress:  obsWebSocketAddress,
		ObsWebSocketPassword: obsWebSocketPassword,
		ObsProxySceneName:    obsProxySceneName,
		MediaMtxApiAddress:   mediaMtxApiAddress,
		MediaMtxRtmpAddress:  mediaMtxRtmpAddress,
		ApiAdminUsername:     apiAdminUsername,
		ApiAdminPassword:     apiAdminPassword,
		ApiPort:              apiPort,
	}
}
