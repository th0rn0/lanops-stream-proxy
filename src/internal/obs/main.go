package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
)

func New(cfg config.Config, dbStreamsClient *dbstreams.Client, obsClient *goobs.Client, msgCh chan<- channels.MsgCh) (*Client, error) {
	client := &Client{
		cfg:   cfg,
		obs:   obsClient,
		db:    dbStreamsClient,
		msgCh: msgCh,
	}
	return client, nil
}
