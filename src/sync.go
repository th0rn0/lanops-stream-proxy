package main

import (
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
)

// Sync Proxy Streams to the DB
func syncProxyStreamsToDatabase() {
	c := time.Tick(5 * time.Second)
	logger.Info().Msg("Starting Proxy Stream Sync")
	for _ = range c {
		// Read Streams from MediaMTX
		proxyStreams, err := getProxyStreams()
		if err != nil {
			logger.Fatal().Err(err).Msg("Error pulling MediaMTX Proxy Streams")
		}

		dbStreams, err := getDBStreams()
		if err != nil {
			logger.Fatal().Err(err).Msg("Error Connecting to DB")
		}

		// Sync Streams
		// Remove DB Streams that do NOT exist in Proxy Streams
		var deleteStreamFlag = true
		for _, dbStream := range dbStreams {
			// check if dbStream exists in proxyStreams.
			// If not - delete
			deleteStreamFlag = true
			for _, proxyStream := range proxyStreams {
				if dbStream.Name == proxyStream.Name {
					deleteStreamFlag = false
				}
			}
			if deleteStreamFlag {
				if err := deleteDBStream(dbStream.Name); err != nil {
					logger.Error().Err(err).Msg("Error removing Stream")
				}
				logger.Info().Msg(fmt.Sprintf("Removed Stream %s", dbStream.Name))
			}
		}

		// Now we have a 'clean' DB of only existing streams, add any extra streams from the Proxy
		for _, proxyStream := range proxyStreams {
			dbStreamExistsFlag, err := checkDBStreamExists(proxyStream.Name)
			if err != nil {
				logger.Error().Err(err).Msg("Error Checking Stream Exists")
			}
			if !dbStreamExistsFlag {
				if err := createDBStream(proxyStream.Name); err != nil {
					logger.Error().Err(err).Msg("Error Adding Stream")
				}
				logger.Info().Msg(fmt.Sprintf("Added Stream %s", proxyStream.Name))
			}
		}
	}
}

// Sync DB Streams to OBS
func syncDatabaseStreamsToOBS() {
	c := time.Tick(5 * time.Second)
	logger.Info().Msg("Starting OBS Stream Sync")
	for _ = range c {
		// Check if OBS Scene Items are in the DB as a valid stream. Remove them if not
		obsSceneItems, err := getOBSSceneItems(obsSceneUuid)
		if err != nil {
			logger.Fatal().Err(err).Msg("Error pulling OBS Streams")
		}

		for _, obsSceneItem := range obsSceneItems {
			var dbStream Stream
			result := db.Where("obs_stream_uuid = ? OR obs_text_uuid = ?", obsSceneItem.SourceUuid, obsSceneItem.SourceUuid).First(&dbStream)
			if errors.Is(result.Error, gorm.ErrRecordNotFound) {
				logger.Info().Msg(fmt.Sprintf("OBS Scene Item (Stream) %s not found in DB. Deleting...", obsSceneItem.SourceName))
				// OBS Scene Item is not found in DB. Assume stream no longer exists and remove it from OBS
				if err := deleteOBSInput(obsSceneItem.SourceUuid); err != nil {
					logger.Error().Err(err).Msg("Error deleting Scene Item (Stream) in OBS")
				}
			}
		}

		// Now only valid streams should be in OBS, Check if DB Streams are in OBS as a Scene item. Add them if not
		dbStreams, err := getDBStreams()
		if err != nil {
			logger.Fatal().Err(err).Msg("Error Connecting to DB")
		}

		for _, dbStream := range dbStreams {
			var existsInObsFlag = false
			for _, obsSceneItem := range obsSceneItems {
				if obsSceneItem.SourceUuid == dbStream.ObsStreamUuid {
					existsInObsFlag = true
				}
			}
			if !existsInObsFlag && dbStream.Enabled {
				// Add Media Source
				if dbStream, err = addStreamMediaSourceToStream(dbStream); err != nil {
					logger.Error().Err(err).Msg("Error adding Media Source Scene Item (Stream) in OBS")
				}

				// Add Text
				if dbStream, err = addStreamNameTextToStream(dbStream); err != nil {
					logger.Error().Err(err).Msg("Error adding Text Scene Item (Stream) in OBS")
				}

				// Transform the text - put me into a function
				if err := transformStreamText(dbStream); err != nil {
					logger.Error().Err(err).Msg("Error Transforming Text Scene Item in OBS")
				}

				// Set Visibility to FALSE
				if err = setStreamVisibility(dbStream, false); err != nil {
					logger.Error().Err(err).Msg("Error setting visibility for Stream Scene Items in OBS")
				}

				// Set Transitions to fade
				// if err = setStreamTransitionsToFade(dbStream); err != nil {
				// 	logger.Error().Err(err).Msg("Error setting transition fade for Stream Scene Items in OBS")
				// }

				// Create Stream Fade Source Filters
				if err = addFadeSourceFilterToStream(dbStream); err != nil {
					logger.Error().Err(err).Msg("Error creating Fade Source filter for Stream Scene Items in OBS")
				}

			}
		}
	}
}
