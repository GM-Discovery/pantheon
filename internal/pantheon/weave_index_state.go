package pantheon

import (
	"encoding/json"
	"os"
)

func writeWeaveIndex(path string, index weaveIndex) error {
	data, err := json.MarshalIndent(index, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
func readWeaveIndex(path string) (weaveIndex, error) {
	var index weaveIndex
	data, err := os.ReadFile(path)
	if err != nil {
		return index, err
	}
	err = json.Unmarshal(data, &index)
	return index, err
}
