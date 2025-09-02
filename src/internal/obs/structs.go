package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/andreykaipov/goobs"
)

type Client struct {
	cfg        config.Config
	obs        *goobs.Client
	db         *dbstreams.Client
	msgCh      chan<- channels.MsgCh
	obsStreams struct {
		current  *dbstreams.Stream
		previous *dbstreams.Stream
	}
}

type ClientError struct {
	Err     error
	Message string
}

type CreateStreamMediaSourceInputOutput struct {
	InputUuid   string
	SceneItemId int64
}
