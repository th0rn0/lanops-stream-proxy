package main

import (
	"fmt"
	"os"
	"time"

	"github.com/andreykaipov/goobs"
	"github.com/gin-gonic/gin"

	"github.com/joho/godotenv"
	"github.com/rs/zerolog"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"

	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"
)

var (
	logger                 zerolog.Logger
	db                     *gorm.DB
	dbPath                 string
	obsClient              *goobs.Client
	obsWebsocketAddress    string
	obsSceneName           string
	obsSceneUuid           string
	proxyStreamApiAddress  string
	proxyStreamRTMPAddress string
	obsStreamRotation      bool
)

func init() {
	var err error
	// TODO - Add to more
	var envExists bool

	logger = zerolog.New(
		zerolog.ConsoleWriter{Out: os.Stderr, TimeFormat: time.RFC3339},
	).Level(zerolog.TraceLevel).With().Timestamp().Caller().Logger()
	logger.Info().Msg("Initializing Mediamtx > OBS Bridge")

	// Env Variables
	logger.Info().Msg("Loading Environment Variables")
	godotenv.Load()

	// Check required Variables
	if proxyStreamApiAddress, envExists = os.LookupEnv("PROXY_STREAM_API_ADDRESS"); !envExists || proxyStreamApiAddress == "" {
		logger.Fatal().Err(err).Msg("PROXY_STREAM_API_ADDRESS IS NOT SET!")
	}
	if proxyStreamRTMPAddress, envExists = os.LookupEnv("PROXY_STREAM_RTMP_ADDRESS"); !envExists || proxyStreamRTMPAddress == "" {
		logger.Fatal().Err(err).Msg("PROXY_STREAM_RTMP_ADDRESS IS NOT SET!")
	}
	if obsSceneName, envExists = os.LookupEnv("OBS_SCENE_NAME"); !envExists || obsSceneName == "" {
		logger.Fatal().Err(err).Msg("OBS_SCENE_NAME IS NOT SET!")
	}
	if dbPath, envExists = os.LookupEnv("DB_PATH"); !envExists || dbPath == "" {
		logger.Fatal().Err(err).Msg("DB_PATH IS NOT SET!")
	}
	if obsWebsocketAddress, envExists = os.LookupEnv("OBS_WEBSOCKET_ADDRESS"); !envExists || obsWebsocketAddress == "" {
		logger.Fatal().Err(err).Msg("OBS_WEBSOCKET_ADDRESS IS NOT SET!")
	}

	// Load Database & Migrate the schema
	db, err = gorm.Open(sqlite.Open(dbPath), &gorm.Config{})
	logger.Info().Msg("Connecting to Database")
	if err != nil {
		logger.Fatal().Err(err).Msg("Error Connecting to Database")
	}
	db.AutoMigrate(&Stream{})

	// Connect to OBS
	logger.Info().Msg("Connecting to OBS")
	obsClient, err = goobs.New(obsWebsocketAddress, goobs.WithPassword(os.Getenv("OBS_PASSWORD")))
	if err != nil {
		logger.Fatal().Err(err).Msg("Error Connecting to OBS")
	}

}

func main() {
	// Defer the Websocket Disconnect
	defer obsClient.Disconnect()

	// Check if Proxy Scene Exists
	logger.Info().Msg("Checking if Proxy Scene exists in OBS")
	obsScenes, err := getOBSScenes()
	if err != nil {
		logger.Fatal().Err(err).Msg("Cannot List OBS Scenes")
	}
	obsSceneExists := false
	for _, obsScene := range obsScenes {
		if obsScene.SceneName == obsSceneName {
			obsSceneExists = true
			obsSceneUuid = obsScene.SceneUuid
		}
	}
	if !obsSceneExists {
		logger.Info().Msg("OBS Scene doesn't exist. Creating...")
		sceneUuid, err := createOBSScene(obsSceneName)
		if err != nil {
			logger.Fatal().Err(err).Msg("Cannot create OBS Scene")
		}
		obsSceneUuid = sceneUuid
		logger.Info().Msg("OBS Scene created!")
	} else {
		logger.Info().Msg("OBS Scene already exists!")
	}
	logger.Info().Msg(fmt.Sprintf("OBS Scene Name: %s", obsSceneName))
	logger.Info().Msg(fmt.Sprintf("OBS Scene UUID: %s", obsSceneUuid))

	// Start Stream Sync
	go syncProxyStreamsToDatabase()
	go syncDatabaseStreamsToOBS()

	resp, _ := obsClient.Inputs.GetInputKindList(&goobRequestInputs.GetInputKindListParams{})

	fmt.Println("Supported OBS Input Kinds:")
	for _, kind := range resp.InputKinds {
		fmt.Println("-", kind)
	}

	// Start Stream Rotation
	// DEBUG - mpve me to a DB entry?
	obsStreamRotation = true
	// obsStreamRotation = false
	go startOBSStreamRotation()

	r := gin.Default()
	// r.GET("/ping", func(c *gin.Context) {
	// 	getProxyStreams()
	// 	addStreamToDB("th0rn0")
	// 	// deleteStream("th0rn0")

	// 	c.JSON(http.StatusOK, gin.H{
	// 		"message": "pong",
	// 	})
	// })
	r.Run() // listen and serve on 0.0.0.0:8080 (for windows "localhost:8080")

}
