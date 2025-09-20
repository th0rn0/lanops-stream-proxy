package obs

import (
	"fmt"
	"lanops/obs-proxy-bridge/internal/dbstreams"
	"time"

	goobRequestFilters "github.com/andreykaipov/goobs/api/requests/filters"
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"
	goobRequestSceneItems "github.com/andreykaipov/goobs/api/requests/sceneitems"
)

func (client *Client) setStreamSceneItemsVisibility(stream dbstreams.Stream, enabled bool) (err error) {
	_, err = client.obs.SceneItems.SetSceneItemEnabled(&goobRequestSceneItems.SetSceneItemEnabledParams{
		SceneUuid:        &client.cfg.ObsProxySceneUuid,
		SceneItemId:      &stream.ObsTextId,
		SceneItemEnabled: &enabled,
	})
	if err != nil {
		return err
	}

	_, err = client.obs.SceneItems.SetSceneItemEnabled(&goobRequestSceneItems.SetSceneItemEnabledParams{
		SceneUuid:        &client.cfg.ObsProxySceneUuid,
		SceneItemId:      &stream.ObsStreamId,
		SceneItemEnabled: &enabled,
	})
	if err != nil {
		return err
	}
	return nil
}

func (client *Client) createStreamSceneItemFadeSourceFilter(sourceUuid string) error {
	filterSettings := map[string]interface{}{
		"opacity": 100.0,
	}
	filterName := "FadeFilter"
	filterKind := "color_filter"
	_, err := client.obs.Filters.CreateSourceFilter(&goobRequestFilters.CreateSourceFilterParams{
		SourceUuid:     &sourceUuid,
		FilterName:     &filterName,
		FilterKind:     &filterKind,
		FilterSettings: filterSettings,
	})
	if err != nil {
		return err
	}
	return nil
}

func (client *Client) setStreamSceneItemsFade(stream dbstreams.Stream, start, end float64) error {
	steps := 40                                        // how many fade steps
	interval := 5 * time.Second / time.Duration(steps) // delay between updates
	for i := 0; i <= steps; i++ {
		opacity := start + (end-start)*(float64(i)/float64(steps))
		filterSettings := map[string]interface{}{
			"opacity": opacity,
		}
		filterName := "FadeFilter"

		// Media Source
		mediaSourceName := fmt.Sprintf("media_source-%s", stream.Name)
		_, err := client.obs.Filters.SetSourceFilterSettings(&goobRequestFilters.SetSourceFilterSettingsParams{
			SourceName: &mediaSourceName,
			// SourceUuid:     &stream.ObsStreamUuid,
			FilterName:     &filterName,
			FilterSettings: filterSettings,
		})
		if err != nil {
			return err
		}
		// Text
		textName := fmt.Sprintf("text-%s", stream.Name)
		_, err = client.obs.Filters.SetSourceFilterSettings(&goobRequestFilters.SetSourceFilterSettingsParams{
			SourceName: &textName,
			// SourceUuid:     &stream.ObsTextUuid,
			FilterName:     &filterName,
			FilterSettings: filterSettings,
		})
		if err != nil {
			return err
		}

		time.Sleep(interval)
	}
	return nil
}

func (client *Client) createStreamMediaSourceInput(stream dbstreams.Stream) (dbstreams.Stream, error) {
	inputSettings := map[string]interface{}{
		"input":               fmt.Sprintf("rtmp://%s/%s", client.cfg.MediaMtxRtmpAddress, stream.Name),
		"is_local_file":       false,
		"looping":             false,
		"restart_on_activate": false,
		"reconnect":           true,
		"buffering_mb":        4,
		"enabled":             false,
	}
	params := goobRequestInputs.NewCreateInputParams().
		WithSceneUuid(client.cfg.ObsProxySceneUuid).
		WithInputName(fmt.Sprintf("media_source-%s", stream.Name)).
		WithInputKind("ffmpeg_source").
		WithInputSettings(inputSettings)
	resp, err := client.obs.Inputs.CreateInput(params)
	if err != nil {
		return stream, err
	}
	updateStreamParams := map[string]interface{}{
		"obs_stream_uuid": resp.InputUuid,
		"obs_stream_id":   resp.SceneItemId,
	}
	if stream, err = client.db.UpdateStream(stream, updateStreamParams); err != nil {
		return stream, err
	}

	// Add Fade Source Filter
	err = client.createStreamSceneItemFadeSourceFilter(stream.ObsStreamUuid)
	if err != nil {
		return stream, err
	}

	// // Fit to screen
	// _, err = client.obs.SceneItems.SetSceneItemTransform(&goobRequestSceneItems.SetSceneItemTransformParams{
	// 	SceneItemId: &stream.ObsStreamId,
	// 	SceneItemTransform: &typedefs.SceneItemTransform{
	// 		BoundsType:      "OBS_BOUNDS_SCALE_INNER", // fit within screen
	// 		BoundsAlignment: 0,                        // center
	// 		BoundsWidth:     1920,                     // match your canvas width
	// 		BoundsHeight:    1080,                     // match your canvas height
	// 	},
	// })
	// if err != nil {
	// 	return stream, err
	// }

	return stream, nil
}

func (client *Client) createStreamTextInput(stream dbstreams.Stream) (dbstreams.Stream, error) {
	inputSettings := map[string]interface{}{
		"text": stream.Name, // the actual text content
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
	params := goobRequestInputs.NewCreateInputParams().
		WithSceneUuid(client.cfg.ObsProxySceneUuid).
		WithInputName(fmt.Sprintf("text-%s", stream.Name)).
		WithInputKind("text_ft2_source_v2").
		WithInputSettings(inputSettings)
	resp, err := client.obs.Inputs.CreateInput(params)
	if err != nil {
		return stream, err
	}

	updateStreamParams := map[string]interface{}{
		"obs_text_uuid": resp.InputUuid,
		"obs_text_id":   resp.SceneItemId,
	}
	if stream, err = client.db.UpdateStream(stream, updateStreamParams); err != nil {
		return stream, err
	}

	// Transform the text
	// First get all the current Item Transforms as we need to pass ALL in the object
	transformResp, err := client.obs.SceneItems.GetSceneItemTransform(&goobRequestSceneItems.GetSceneItemTransformParams{
		SceneItemId: &resp.SceneItemId,
		SceneUuid:   &client.cfg.ObsProxySceneUuid,
	})
	if err != nil {
		return stream, err
	}
	// Set to Top Left with a small border
	transformResp.SceneItemTransform.PositionX = 102.0
	transformResp.SceneItemTransform.PositionY = 47.0
	transformResp.SceneItemTransform.BoundsWidth = 500.0
	transformResp.SceneItemTransform.BoundsHeight = 500.0
	_, err = client.obs.SceneItems.SetSceneItemTransform(&goobRequestSceneItems.SetSceneItemTransformParams{
		SceneItemId:        &resp.SceneItemId,
		SceneUuid:          &client.cfg.ObsProxySceneUuid,
		SceneItemTransform: transformResp.SceneItemTransform,
	})
	if err != nil {
		return stream, err
	}

	// Add Fade Source Filter
	err = client.createStreamSceneItemFadeSourceFilter(stream.ObsTextUuid)
	if err != nil {
		return stream, err
	}
	return stream, nil
}
