package api

import (
	"errors"
	"fmt"
	"net/http"
	"os"
	"time"

	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

var (
	logger          zerolog.Logger
	dbStreamsClient *dbstreams.Client
	cfg             config.Config
)

func init() {
	var err error
	logger = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	logger.Info().Msg("Loading API")

	cfg = config.Load()
	db, err := gorm.Open(sqlite.Open(cfg.DbPath), &gorm.Config{})
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed Connecting to DB")
	}
	dbStreamsClient, err = dbstreams.New(cfg, db)
	if err != nil {
		logger.Fatal().Err(err).Msg("Failed to create DB Streams Client")
	}
}

// TODO - new me up and pass in the logger and db client
// func Run(client) {
func Run() {
	gin.DefaultWriter = zerolog.ConsoleWriter{Out: os.Stdout, TimeFormat: time.RFC3339}
	r := gin.Default()
	r.Use(func(c *gin.Context) {
		start := time.Now()
		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		logger.Info().
			Str("method", c.Request.Method).
			Str("path", c.Request.URL.Path).
			Int("status", status).
			Dur("latency", latency).
			Msg("request handled")
	})
	r.Use(cors.Default())
	authorized := r.Group("", gin.BasicAuth(gin.Accounts{
		cfg.ApiAdminUsername: cfg.ApiAdminPassword,
	}))
	authorized.GET("/streams", handleGetStreams)
	authorized.GET("/streams/:name", handleGetStreamByName)
	authorized.POST("/streams/:name/enable", handleEnableStreamByName)
	r.Run(fmt.Sprintf(":%s", cfg.ApiPort))
}

func handleGetStreams(c *gin.Context) {
	streams, err := dbStreamsClient.GetStreams()
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Error with Database")
	}
	c.JSON(http.StatusOK, streams)
}

func handleGetStreamByName(c *gin.Context) {
	name := c.Param("name")
	dbStream, err := dbStreamsClient.GetStreamByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Cannot find Stream")
			return
		}
		c.JSON(http.StatusInternalServerError, "Something went wrong")
		return
	}
	c.JSON(http.StatusFound, dbStream)
}

func handleEnableStreamByName(c *gin.Context) {
	var handleEnableStreamByNameParams HandleEnableStreamByNameParams
	name := c.Param("name")
	dbStream, err := dbStreamsClient.GetStreamByName(name)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			c.JSON(http.StatusNotFound, "Cannot find Stream")
			return
		}
		c.JSON(http.StatusInternalServerError, "Something went wrong")
		return
	}
	if err := c.ShouldBindJSON(&handleEnableStreamByNameParams); err != nil {
		c.JSON(http.StatusBadRequest, err)
		return
	}
	dbStreamExists, err := dbStreamsClient.CheckStreamExistsByName(dbStream.Name)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Something went wrong")
		return
	}
	if !dbStreamExists {
		c.JSON(http.StatusNotFound, "Cannot find Stream")
		return
	}
	params := map[string]interface{}{
		"enabled": true,
	}
	if !handleEnableStreamByNameParams.Enabled {
		params = map[string]interface{}{
			"enabled": false,
		}
	}
	dbStream, err = dbStreamsClient.UpdateStream(dbStream, params)
	if err != nil {
		c.JSON(http.StatusInternalServerError, "Something went wrong")
		return
	}

	c.JSON(http.StatusOK, dbStream)
}

type HandleEnableStreamByNameParams struct {
	Enabled bool `json:"enabled"`
}
