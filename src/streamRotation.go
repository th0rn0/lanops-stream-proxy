package main

import (
	"time"
)

func startOBSStreamRotation() {
	streams, err := getDBStreams()
	if err != nil {
		logger.Error().Err(err).Msg("Cannot pull DB Streams")
	}
	if len(streams) > 1 {
		for _, stream := range streams {
			if err := setStreamOpacity(stream, 0); err != nil {
				logger.Error().Err(err).Msg("Error setting opacity for Stream Scene Items in OBS")
			}
			if err := setStreamVisibility(stream, false); err != nil {
				logger.Error().Err(err).Msg("Error setting visibility for Stream Scene Items in OBS")
			}
		}
	}
	for {
		streams, err := getDBStreams()
		if err != nil {
			logger.Error().Err(err).Msg("Cannot pull DB Streams")
		}
		for _, stream := range streams {
			// Because of how we are pulling the records from the DB and not wanting to just pull random streams (give all streams a fair viewing)
			// We will check if the stream still exists, if it doesn't then we will move on to the next
			if streamExists, _ := checkDBStreamExists(stream.Name); !streamExists {
				continue
			}
			if len(streams) > 1 {
				go fadeInStream(stream)
				time.Sleep(20 * time.Second)
				go fadeOutStream(stream)
			} else {
				if err := setStreamOpacity(stream, 100); err != nil {
					logger.Error().Err(err).Msg("Error setting opacity for Stream Scene Items in OBS")
				}
				if err := setStreamVisibility(stream, true); err != nil {
					logger.Error().Err(err).Msg("Error setting visibility for Stream Scene Items in OBS")
				}
				time.Sleep(20 * time.Second)
			}

		}
		if !obsStreamRotation {
			break
		}
	}
}
