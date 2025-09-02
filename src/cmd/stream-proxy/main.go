package main

import (
	"lanops/obs-proxy-bridge/api"
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"lanops/obs-proxy-bridge/internal/mediamtx"
	"lanops/obs-proxy-bridge/internal/obs"

	"os"
	"time"

	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	logger zerolog.Logger
	cfg    config.Config
	msgCh  = make(chan channels.MsgCh, 20)
)

func init() {
	logger = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	logger.Info().Msg("Starting LanOps Stream Proxy")

	logger.Info().Msg("Loading Config")
	cfg = config.Load()

	db, err := gorm.Open(sqlite.Open(cfg.DbPath), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed Connecting to DB")
	}
	db.AutoMigrate(dbstreams.Stream{})
}

func main() {

	logger.Info().Msg("Starting Proxy Stream Sync")

	// Message Channel
	go func() {
		for msg := range msgCh {
			logger.Info().Msg(msg.Message)
		}
	}()

	dbStreamsClient, err := dbstreams.New(cfg)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create DB Streams Client")
	}

	mediamtxClient, err := mediamtx.New(cfg, dbStreamsClient, msgCh)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create MediaMTX Client")
	}

	obsClient, err := obs.New(cfg, dbStreamsClient, msgCh)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to OBS Client")
	}

	logger.Info().Msg("Checking if Proxy Scene exists in OBS")
	if err := obsClient.InitProxyScene(); err != nil {
		logger.Fatal().Err(err.Err).Msg("Cannot create OBS Scene")
	}

	logger.Info().Msg("Starting API")
	go api.Run()

	logger.Info().Msg("Starting Stream Syncs")
	go func() {
		c := time.Tick(7 * time.Second)
		for _ = range c {
			err := mediamtxClient.SyncStreams()
			if err != nil {
				logger.Error().Err(err.Err).Msg(err.Message)
			}
			errw := obsClient.SyncStreams()
			if errw != nil {
				logger.Error().Err(errw.Err).Msg(err.Message)
			}
		}
	}()

	logger.Info().Msg("Starting Active Stream Rotation")
	c := time.Tick(30 * time.Second)
	for _ = range c {
		err := obsClient.RotateActiveStream()
		if err != nil {
			logger.Error().Err(err.Err).Msg(err.Message)
		}
	}
}
