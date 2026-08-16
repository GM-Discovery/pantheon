package pantheon

import (
	"encoding/json"
	"os"
)

func readNexusChunkInventory(path string) (nexusChunkInventory, error) {
	var inv nexusChunkInventory
	data, err := os.ReadFile(path)
	if err != nil {
		return inv, err
	}
	err = json.Unmarshal(data, &inv)
	return inv, err
}
func readFingerprintState(path string) (weaveFingerprintState, error) {
	var state weaveFingerprintState
	data, err := os.ReadFile(path)
	if err != nil {
		return state, err
	}
	err = json.Unmarshal(data, &state)
	if state.Files == nil {
		state.Files = map[string]string{}
	}
	return state, err
}
