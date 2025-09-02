package mediamtx

import (
	"encoding/json"
	"fmt"
	"io"
	"lanops/obs-proxy-bridge/internal/channels"
	"lanops/obs-proxy-bridge/internal/config"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"net/http"
)

func New(cfg config.Config, dbStreamsClient *dbstreams.Client, msgCh chan<- channels.MsgCh) (*Client, error) {
	client := &Client{
		cfg:   cfg,
		db:    dbStreamsClient,
		msgCh: msgCh,
	}
	return client, nil
}

func (client *Client) SyncStreams() *ClientError {
	// Read Streams from MediaMTX
	proxyStreamsList, err := client.GetStreams()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pulling MediaMTX Proxy Streams",
		}
	}

	dbStreamsList, err := client.db.GetStreams()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Error pulling DB Streams",
		}
	}

	// Sync Streams
	// Remove DB Streams that do NOT exist in Proxy Streams
	var deleteStream = true
	for _, dbStream := range dbStreamsList {
		// check if dbStream exists in proxyStreams. If not, delete
		deleteStream = true
		for _, proxyStream := range proxyStreamsList {
			if dbStream.Name == proxyStream.Name {
				deleteStream = false
			}
		}
		if deleteStream {
			if err := client.db.DeleteStream(dbStream.Name); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error removing DB Stream",
				}
			}
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Removed Stream %s", dbStream.Name), Level: "INFO"}
		}
	}

	// Now we have a 'clean' DB of only existing streams, add any extra streams from the Proxy
	for _, proxyStream := range proxyStreamsList {
		dbStreamExists, err := client.db.CheckStreamExistsByName(proxyStream.Name)
		if err != nil {
			return &ClientError{
				Err:     err,
				Message: "Error checking DB Stream",
			}
		}
		if !dbStreamExists {
			if _, err := client.db.CreateStream(proxyStream.Name); err != nil {
				return &ClientError{
					Err:     err,
					Message: "Error adding DB Stream",
				}
			}
			client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("Stream Found! Adding %s to the DB", proxyStream.Name), Level: "INFO"}
		}
	}
	return nil
}

func (client *Client) GetStreams() ([]MediamtxListStreamsOutput, error) {
	var response MediamtxListStreamsResponse
	var streams []MediamtxListStreamsOutput
	url := fmt.Sprintf("http://%s/v3/paths/list", client.cfg.MediaMtxApiAddress)

	// Create HTTP GET request
	resp, err := http.Get(url)
	if err != nil {
		return streams, err
	}
	defer resp.Body.Close()

	// Read response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return streams, err
	}

	err = json.Unmarshal(body, &response)
	if err != nil {
		return streams, err
	}
	streams = response.Items
	return streams, nil
}
