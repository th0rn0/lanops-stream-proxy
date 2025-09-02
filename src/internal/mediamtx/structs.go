package mediamtx

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"time"
)

type Client struct {
	cfg   config.Config
	db    *dbstreams.Client
	msgCh chan<- channels.MsgCh
}

type ClientError struct {
	Err     error
	Message string
}

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
