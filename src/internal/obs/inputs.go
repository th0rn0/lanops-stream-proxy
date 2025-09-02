package obs

import (
	goobRequestInputs "github.com/andreykaipov/goobs/api/requests/inputs"
)

func (client *Client) deleteInput(inputUuid string) error {
	params := goobRequestInputs.NewRemoveInputParams().
		WithInputUuid(inputUuid)
	_, err := client.obs.Inputs.RemoveInput(params)
	if err != nil {
		return err
	}
	return nil
}
