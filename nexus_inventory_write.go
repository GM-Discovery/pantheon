package main

import (
	"encoding/json"
	"os"
)

func writeNexusChunkInventory(path string, inventory nexusChunkInventory) error {
	data, err := json.MarshalIndent(inventory, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}
