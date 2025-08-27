package main

import (
	"fmt"
	"time"
)

func transformStreamText(dbStream Stream) error {
	transformResp, err := getOBSSceneItemTransform(obsSceneUuid, dbStream.ObsTextId)
	if err != nil {
		return err
	}
	transformResp.SceneItemTransform.PositionX = 102.0
	transformResp.SceneItemTransform.PositionY = 47.0
	transformResp.SceneItemTransform.BoundsWidth = 500.0
	transformResp.SceneItemTransform.BoundsHeight = 500.0
	_, err = setOBSSceneItemTransform(obsSceneUuid, dbStream.ObsTextId, transformResp.SceneItemTransform)
	if err != nil {
		return err
	}
	return nil
}

func addFadeSourceFilterToStream(dbStream Stream) error {
	filterSettings := map[string]interface{}{
		"opacity": 1.0,
	}
	if err := createOBSSourceFilter(dbStream.ObsStreamUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
		return err
	}
	if err := createOBSSourceFilter(dbStream.ObsTextUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
		return err
	}
	return nil
}

func fadeStream(dbStream Stream, start, end float64) error {
	steps := 40                                        // how many fade steps
	interval := 2 * time.Second / time.Duration(steps) // delay between updates

	for i := 0; i <= steps; i++ {
		// opacity := 1.0 - (float64(i) / float64(steps))
		opacity := start + (end-start)*(float64(i)/float64(steps))
		filterSettings := map[string]interface{}{
			"opacity": opacity,
		}
		if err := setOBSSourceFilterSettings(dbStream.ObsTextUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
			return err
		}
		if err := setOBSSourceFilterSettings(dbStream.ObsStreamUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
			return err
		}
		time.Sleep(interval)
	}
	return nil
}

func setStreamOpacity(dbStream Stream, opacity float64) error {
	filterSettings := map[string]interface{}{
		"opacity": opacity,
	}
	if err := setOBSSourceFilterSettings(dbStream.ObsStreamUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
		return err
	}
	if err := setOBSSourceFilterSettings(dbStream.ObsTextUuid, "FadeFilter", "color_filter", filterSettings); err != nil {
		return err
	}
	return nil
}

func setStreamVisibility(dbStream Stream, enabled bool) error {
	if err := setOBSSceneItemEnabled(obsSceneUuid, dbStream.ObsStreamId, enabled); err != nil {
		return err
	}
	if err := setOBSSceneItemEnabled(obsSceneUuid, dbStream.ObsTextId, enabled); err != nil {
		return err
	}
	return nil
}

func addStreamNameTextToStream(dbStream Stream) (Stream, error) {
	inputSettingsText := map[string]interface{}{
		"text": dbStream.Name, // the actual text content
		"font": map[string]interface{}{
			"face":  "Arial",
			"size":  50,
			"flags": 0, // optional font flags (bold/italic etc.)
		},
		"color1":     0xFFFFFFFF, // ARGB color
		"outline":    1,          // outline thickness
		"boundsType": "OBS_BOUNDS_NONE",
		"enabled":    false,
	}
	inputTextUuid, inputTextId, err := createOBSInput(obsSceneUuid, fmt.Sprintf("text-%s", dbStream.Name), "text_gdiplus_v3", inputSettingsText)
	if err != nil {
		return dbStream, err
	}
	updateText := map[string]interface{}{
		"obs_text_uuid": inputTextUuid,
		"obs_text_id":   inputTextId,
	}
	if err := db.Model(&dbStream).Updates(updateText).Error; err != nil {
		return dbStream, err
	}
	return dbStream, nil
}

func addStreamMediaSourceToStream(dbStream Stream) (Stream, error) {
	inputSettingsMediaSource := map[string]interface{}{
		"input":               fmt.Sprintf("rtmp://%s/%s", proxyStreamRTMPAddress, dbStream.Name),
		"is_local_file":       false,
		"looping":             false,
		"restart_on_activate": false,
		"reconnect":           true,
		"buffering_mb":        4,
		"enabled":             false,
	}
	inputMediaSourceUuid, inputMediaSourceId, err := createOBSInput(obsSceneUuid, fmt.Sprintf("media_source-%s", dbStream.Name), "ffmpeg_source", inputSettingsMediaSource)
	if err != nil {
		return dbStream, err
	}
	updateStream := map[string]interface{}{
		"obs_stream_uuid": inputMediaSourceUuid,
		"obs_stream_id":   inputMediaSourceId,
	}
	if err := db.Model(&dbStream).Updates(updateStream).Error; err != nil {
		return dbStream, err
	}
	return dbStream, nil
}

func fadeInStream(stream Stream) {
	if err := setStreamVisibility(stream, true); err != nil {
		logger.Error().Err(err).Msg("Error setting visibility for Stream Scene Items in OBS")
	}
	if err := fadeStream(stream, 0, 100); err != nil {
		logger.Error().Err(err).Msg("Error setting opacity for Stream Scene Items in OBS")
	}
}

func fadeOutStream(stream Stream) {
	if err := fadeStream(stream, 100, 0); err != nil {
		logger.Error().Err(err).Msg("Error setting opacity for Stream Scene Items in OBS")
	}
	time.Sleep(5 * time.Second)
	if err := setStreamVisibility(stream, false); err != nil {
		logger.Error().Err(err).Msg("Error setting visibility for Stream Scene Items in OBS")
	}
}
