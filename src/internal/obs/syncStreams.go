package obs

import (
	"fmt"
	"lanops/obs-proxy-bridge/internal/channels"
)

func (client *Client) SyncStreams() *ClientError {
	obsSceneItems, err := client.getProxySceneItems()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pulling OBS Streams",
		}
	}

	// Check if OBS Scene Items are in the DB as a valid stream and enabled. If not, remove them from OBS
	for _, obsSceneItem := range obsSceneItems {
		// Check if stream Exists in DB. If not, remove from OBS
		dbStreamExists, err := client.db.CheckStreamExistsByObsSceneItemUuid(obsSceneItem.SourceUuid)
		if err != nil {
			return &ClientError{
				Err:     err,
				Message: "Error pulling DB Streams",
			}
		}
		if !dbStreamExists {
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene Item (Stream) %s not found in DB. Deleting...", obsSceneItem.SourceName), Level: "INFO"}
			// OBS Scene Item is not found in DB. Assume stream no longer exists and remove it from OBS
			if err := client.deleteInput(obsSceneItem.SourceUuid); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error deleting Scene Item in OBS",
				}
			}
			continue
		}

		// Stream exists in DB, check if stream is Enabled. If not, remove from OBS
		dbStream, err := client.db.GetStreamByObsSceneItemUuid(obsSceneItem.SourceUuid)
		if err != nil {
			return &ClientError{
				Err:     err,
				Message: "Error pulling DB Stream",
			}
		}
		if !dbStream.Enabled {
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene Item (Stream) %s Disabled. Deleting...", obsSceneItem.SourceName), Level: "INFO"}
			if err := client.deleteInput(obsSceneItem.SourceUuid); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error deleting Scene Item in OBS",
				}
			}
			params := map[string]interface{}{
				"obs_stream_id":   nil,
				"obs_stream_uuid": nil,
				"obs_text_uuid":   nil,
				"obs_text_id":     nil,
			}
			if _, err := client.db.UpdateStream(dbStream, params); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error Updating DB Stream",
				}
			}
		}
	}

	// Now only valid streams should be in OBS, Check if DB Streams are in OBS as a Scene item. If not, add them to OBS
	dbStreams, err := client.db.GetStreams()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pulling DB Streams",
		}
	}
	for _, dbStream := range dbStreams {
		var dbStreamExistsInObs = false
		for _, obsSceneItem := range obsSceneItems {
			if obsSceneItem.SourceUuid == dbStream.ObsStreamUuid {
				dbStreamExistsInObs = true
			}
		}

		// Check if stream exists in OBS. If not, add it.
		if !dbStreamExistsInObs && dbStream.Enabled {
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Adding stream %s to OBS", dbStream.Name), Level: "INFO"}

			// Add Media Source
			if dbStream, err = client.createStreamMediaSourceInput(dbStream); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error adding Media Source Scene Item (Stream) in OBS",
				}
			}

			// Add Text
			if dbStream, err = client.createStreamTextInput(dbStream); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error adding Text Scene Item (Stream) in OBS",
				}
			}

			// Set Visibility of Stream Scene Items to FALSE
			if err = client.setStreamSceneItemsVisibility(dbStream, false); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error setting visibility for Stream Scene Items in OBS",
				}
			}
		}
	}

	return nil
}
