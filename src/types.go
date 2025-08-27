package main

import "time"

// Models
type Stream struct {
	Name    string `gorm:"primaryKey" json:"name"`
	Enabled bool   `gorm:"default:true" json:"enabled"`
	// We need both UUID and ID due to imitations with the OBS WebSockets
	ObsStreamUuid string `json:"obsStreamUuid"`
	ObsStreamId   int    `json:"obsStreamId"`
	ObsTextUuid   string `json:"obsTextUuid"`
	ObsTextId     int    `json:"obsTextId"`
}

// Responses
type MediamtxListStreamsOutput struct {
	Name     string `json:"name"`
	ConfName string `json:"confName"`
	Source   struct {
		Type string `json:"type"`
		ID   string `json:"id"`
	} `json:"source"`
	Ready         bool          `json:"ready"`
	ReadyTime     time.Time     `json:"readyTime"`
	Tracks        []string      `json:"tracks"`
	BytesReceived int           `json:"bytesReceived"`
	BytesSent     int           `json:"bytesSent"`
	Readers       []interface{} `json:"readers"`
}

type MediamtxListStreamsResponse struct {
	ItemCount int                         `json:"itemCount"`
	PageCount int                         `json:"pageCount"`
	Items     []MediamtxListStreamsOutput `json:"items"`
}
