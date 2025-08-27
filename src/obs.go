package main

import (
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"

	goobRequestFilters "github.com/andreykaipov/goobs/api/requests/filters"
	goobRequestSceneItems "github.com/andreykaipov/goobs/api/requests/sceneitems"
	goobRequestScenes "github.com/andreykaipov/goobs/api/requests/scenes"

	"github.com/andreykaipov/goobs/api/typedefs"
)

func getOBSScenes() ([]*typedefs.Scene, error) {
	var scenes []*typedefs.Scene
	items, err := obsClient.Scenes.GetSceneList()
	if err != nil {
		return scenes, err
	}
	scenes = items.Scenes
	return scenes, nil
}

func deleteOBSScene(sceneUuid string) error {
	_, err := obsClient.SceneItems.RemoveSceneItem(&goobRequestSceneItems.RemoveSceneItemParams{
		SceneUuid: &sceneUuid,
	})
	if err != nil {
		return err
	}
	return nil
}

func createOBSScene(sceneName string) (string, error) {
	params := goobRequestScenes.CreateSceneParams{
		SceneName: &sceneName,
	}
	resp, err := obsClient.Scenes.CreateScene(&params)
	if err != nil {
		return "", err
	}
	return resp.SceneUuid, nil
}

func createOBSInput(sceneUuid, name, kind string, inputSettings map[string]interface{}) (string, int, error) {
	params := goobRequestInputs.NewCreateInputParams().
		WithSceneUuid(sceneUuid).
		WithInputName(name).
		WithInputKind(kind).
		WithInputSettings(inputSettings)
	resp, err := obsClient.Inputs.CreateInput(params)
	if err != nil {
		return "", 0, err
	}
	return resp.InputUuid, resp.SceneItemId, nil
}

func deleteOBSInput(inputUuid string) error {
	params := goobRequestInputs.NewRemoveInputParams().
		WithInputUuid(inputUuid)
	_, err := obsClient.Inputs.RemoveInput(params)
	if err != nil {
		return err
	}
	return nil
}

func getOBSSceneItems(sceneUuid string) ([]*typedefs.SceneItem, error) {
	var obsSceneItems []*typedefs.SceneItem
	obsClient.SceneItems.GetSceneItemList()
	items, err := obsClient.SceneItems.GetSceneItemList(&goobRequestSceneItems.GetSceneItemListParams{
		SceneUuid: &sceneUuid,
	})
	if err != nil {
		return obsSceneItems, err
	}
	obsSceneItems = items.SceneItems
	return obsSceneItems, err
}

func getOBSSceneItemTransform(sceneUuid string, sceneItemId int) (*goobRequestSceneItems.GetSceneItemTransformResponse, error) {
	transformResp, err := obsClient.SceneItems.GetSceneItemTransform(&goobRequestSceneItems.GetSceneItemTransformParams{
		SceneItemId: &sceneItemId,
		SceneUuid:   &sceneUuid,
	})
	if err != nil {
		return transformResp, err
	}
	return transformResp, nil
}

func setOBSSceneItemTransform(sceneUuid string, sceneItemId int, sceneItemTransform *typedefs.SceneItemTransform) (*goobRequestSceneItems.SetSceneItemTransformResponse, error) {
	transformResp, err := obsClient.SceneItems.SetSceneItemTransform(&goobRequestSceneItems.SetSceneItemTransformParams{
		SceneItemId:        &sceneItemId,
		SceneUuid:          &sceneUuid,
		SceneItemTransform: sceneItemTransform,
	})
	if err != nil {
		return transformResp, err
	}
	return transformResp, nil
}

func setOBSSceneItemEnabled(sceneUuid string, sceneItemId int, enabled bool) error {
	_, err := obsClient.SceneItems.SetSceneItemEnabled(&goobRequestSceneItems.SetSceneItemEnabledParams{
		SceneUuid:        &sceneUuid,
		SceneItemId:      &sceneItemId,
		SceneItemEnabled: &enabled, // set to false to hide
	})
	if err != nil {
		return err
	}
	return nil
}

func createOBSSourceFilter(sourceUuid string, filterName string, filterKind string, params map[string]interface{}) error {
	_, err := obsClient.Filters.CreateSourceFilter(&goobRequestFilters.CreateSourceFilterParams{
		SourceUuid:     &sourceUuid,
		FilterName:     &filterName,
		FilterKind:     &filterKind,
		FilterSettings: params,
	})
	if err != nil {
		return err
	}
	return nil
}

func setOBSSourceFilterSettings(sourceUuid string, filterName string, filterKind string, params map[string]interface{}) error {
	_, err := obsClient.Filters.SetSourceFilterSettings(&goobRequestFilters.SetSourceFilterSettingsParams{
		SourceUuid:     &sourceUuid,
		FilterName:     &filterName,
		FilterSettings: params,
	})
	if err != nil {
		return err
	}
	return nil
}
