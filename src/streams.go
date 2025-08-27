package main

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

func getProxyStreams() ([]MediamtxListStreamsOutput, error) {
	var response MediamtxListStreamsResponse
	var streams []MediamtxListStreamsOutput
	// Example URL (replace with your target)
	url := fmt.Sprintf("http://%s/v3/paths/list", proxyStreamAddress)

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

func getDBStreams() ([]Stream, error) {
	var dbStreams []Stream
	result := db.Find(&dbStreams)
	if result.Error != nil {
		return dbStreams, result.Error
	}
	return dbStreams, nil
}

func getDBStreamByName(streamName string) (Stream, error) {
	var stream Stream

	if err := db.First(&stream, "name = ?", streamName).Error; err != nil {
		return stream, err
	}
	return stream, nil
}

func createDBStream(streamName string) error {
	var stream Stream

	result := db.FirstOrCreate(&stream, Stream{
		Name:    streamName,
		Enabled: true,
	})
	if result.Error != nil {
		return result.Error
	}
	return nil
}

func deleteDBStream(streamName string) error {
	var stream Stream

	if err := db.First(&stream, "name = ?", streamName).Error; err != nil {
		return err
	}
	if err := db.Unscoped().Delete(&stream).Error; err != nil {
		return err
	}
	return nil
}

func checkDBStreamExists(streamName string) (bool, error) {
	var stream Stream
	var count int64
	err := db.Model(stream).Where("name = ?", streamName).Count(&count).Error
	if err != nil {
		return false, err
	}
	return count > 0, err
}
