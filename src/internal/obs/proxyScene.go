package obs

import (
	"fmt"
	"lanops/obs-proxy-bridge/internal/channels"

	goobRequestSceneItems "github.com/andreykaipov/goobs/api/requests/sceneitems"
	goobRequestScenes "github.com/andreykaipov/goobs/api/requests/scenes"

	"github.com/andreykaipov/goobs/api/typedefs"
)

func (client *Client) InitProxyScene() *ClientError {
	obsScenes, err := client.getScenes()
	if err != nil {
		return &ClientError{
			Err:     err,
			Message: "Cannot List OBS Scenes",
		}

	}
	obsSceneExists := false
	for _, obsScene := range obsScenes {
		fmt.Println(obsScene)
		if obsScene.SceneName == client.cfg.ObsProxySceneName {
			obsSceneExists = true
			client.cfg.ObsProxySceneUuid = obsScene.SceneUuid
		}
	}
	if !obsSceneExists {
		client.msgCh <- channels.MsgCh{Err: nil, Message: "OBS Scene doesn't exist. Creating...", Level: "INFO"}

		sceneUuid, err := client.createProxyScene()
		if err != nil {
			return &ClientError{
				Err:     err,
				Message: "Cannot Create OBS Scene",
			}
		}
		client.cfg.ObsProxySceneUuid = *sceneUuid
		client.msgCh <- channels.MsgCh{Err: nil, Message: "OBS Scene created!", Level: "INFO"}
	} else {
		client.msgCh <- channels.MsgCh{Err: nil, Message: "OBS Scene already exists!", Level: "INFO"}
	}
	client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene Name: %s", client.cfg.ObsProxySceneName), Level: "INFO"}
	client.msgCh <- channels.MsgCh{Err: nil, Message: fmt.Sprintf("OBS Scene UUID: %s", client.cfg.ObsProxySceneUuid), Level: "INFO"}

	return nil
}

func (client *Client) createProxyScene() (*string, error) {
	params := goobRequestScenes.CreateSceneParams{
		SceneName: &client.cfg.ObsProxySceneName,
	}
	resp, err := client.obs.Scenes.CreateScene(&params)
	if err != nil {
		return nil, err
	}
	return &resp.SceneUuid, nil
}

func (client *Client) getProxySceneItems() ([]*typedefs.SceneItem, error) {
	items, err := client.obs.SceneItems.GetSceneItemList(&goobRequestSceneItems.GetSceneItemListParams{
		SceneUuid: &client.cfg.ObsProxySceneUuid,
	})
	if err != nil {
		return nil, err
	}
	return items.SceneItems, nil
}
