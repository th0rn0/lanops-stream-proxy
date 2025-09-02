package dbstreams

import (
	"lanops/obs-proxy-bridge/internal/config"

	"gorm.io/gorm"
)

type Client struct {
	cfg config.Config
	db  *gorm.DB
}

type Stream struct {
	ID      uint   `gorm:"primaryKey" json:"id"`
	Name    string `gorm:"unique" json:"name"`
	Enabled bool   `gorm:"default:false" json:"enabled"`
	// We need both UUID and ID due to imitations with the OBS WebSockets
	ObsStreamUuid string `json:"obsStreamUuid;null;default:null"`
	ObsStreamId   int    `json:"obsStreamId;null;default:null"`
	ObsTextUuid   string `json:"obsTextUuid;null;default:null"`
	ObsTextId     int    `json:"obsTextId;null;default:null"`
}
