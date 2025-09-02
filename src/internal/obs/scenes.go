package obs

import (
	"github.com/andreykaipov/goobs/api/typedefs"
)

func (client *Client) getScenes() ([]*typedefs.Scene, error) {
	var scenes []*typedefs.Scene
	items, err := client.obs.Scenes.GetSceneList()
	if err != nil {
		return scenes, err
	}
	scenes = items.Scenes
	return scenes, nil
}
