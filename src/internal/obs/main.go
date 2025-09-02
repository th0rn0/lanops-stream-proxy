package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
)

func New(cfg config.Config, dbStreamsClient *dbstreams.Client, msgCh chan<- channels.MsgCh) (*Client, error) {
	obsClient, err := goobs.New(cfg.ObsWebSocketAddress, goobs.WithPassword(cfg.ObsWebSocketPassword))
	if err != nil {
		return nil, err
	}

	client := &Client{
		cfg:   cfg,
		obs:   obsClient,
		db:    dbStreamsClient,
		msgCh: msgCh,
	}
	return client, nil
}
