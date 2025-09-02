package obs

import (
	"lanops/obs-proxy-bridge/internal/channels"
)

func (client *Client) RotateActiveStream() *ClientError {
	streamCount, err := client.db.GetAvailableStreamsCount()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot pull DB Streams",
		}
	}

	if streamCount > 0 {
		// Check if Active Stream has been removed from DB.
		// Fixes bug when currently active stream is removed from DB.
		if client.obsStreams.current != nil {
			streamExistsInDB, err := client.db.CheckStreamExistsByName(client.obsStreams.current.Name)
			if err != nil {
				return &ClientError{
					Err:     err,
					Message: "Cannot pull DB Streams",
				}
			}
			if !streamExistsInDB {
				client.obsStreams.current = nil
			}
		}

		nextStream, err := client.db.GetNextAvailableStream(client.obsStreams.current)
		if err != nil {
			return &ClientError{
				Err:     err,
				Message: "Cannot pull next DB Stream",
			}
		}
		if client.obsStreams.current == nil {
			// Assume just app just booted OR its the first/only stream
			// TODO - RESET ALL streams
			if err = client.setStreamSceneItemsVisibility(*nextStream, true); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Cannot Set Stream Visibility",
				}
			}
			if err = client.setStreamSceneItemsFade(*nextStream, 0, 100); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Cannot Set Stream Visibility",
				}
			}
		}
		if client.obsStreams.current != nil && nextStream.ID != client.obsStreams.current.ID {
			go func() *ClientError {
				if err = client.setStreamSceneItemsVisibility(*nextStream, true); err != nil {
					return &ClientError{
						Err:     err,
						Message: "Cannot Set Stream Visibility",
					}
				}
				if err = client.setStreamSceneItemsFade(*nextStream, 0, 100); err != nil {
					return &ClientError{
						Err:     err,
						Message: "Cannot Set Stream Visibility",
					}
				}
				return nil
			}()
			if err = client.setStreamSceneItemsFade(*client.obsStreams.current, 100, 0); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Cannot Set Stream Visibility",
				}
			}
			if err = client.setStreamSceneItemsVisibility(*client.obsStreams.current, false); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Cannot Set Stream Visibility",
				}
			}
			client.obsStreams.previous = client.obsStreams.current
		}
		client.obsStreams.current = nextStream
	} else {
		client.msgCh <- channels.MsgCh{Err: nil, Message: "No Enabled Streams found", Level: "INFO"}
		client.obsStreams.previous = nil
		client.obsStreams.current = nil
	}
	return nil
}
