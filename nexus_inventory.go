package main

import (
	"io"
	"io/fs"
	"path/filepath"
	"sort"
)

type nexusChunkInventory struct {
	Chunks          []nexusChunk `json:"chunks"`
	UserChunkCount  int          `json:"nexus_user_chunk_count"`
	AgentChunkCount int          `json:"nexus_agent_chunk_count"`
}

func buildNexusChunkInventory(vaultPath string) (nexusChunkInventory, error) {
	var inv nexusChunkInventory
	root := filepath.Join(vaultPath, "Nexus")
	err := filepath.WalkDir(root, func(path string, e fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if e.Type()&fs.ModeSymlink != 0 {
			return nil
		}
		if e.IsDir() || filepath.Ext(path) != ".md" {
			return nil
		}
		f, err := openVaultMarkdown(path)
		if err != nil {
			return err
		}
		data, err := io.ReadAll(f)
		f.Close()
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(vaultPath, path)
		if err != nil {
			return err
		}
		chunks := chunkNexusConversation(filepath.ToSlash(rel), string(data))
		inv.Chunks = append(inv.Chunks, chunks...)
		return nil
	})
	if err != nil {
		return inv, err
	}
	sort.Slice(inv.Chunks, func(i, j int) bool { return inv.Chunks[i].Path < inv.Chunks[j].Path })
	for _, c := range inv.Chunks {
		if c.Authorship == "nexus_user" {
			inv.UserChunkCount++
		} else if c.Authorship == "nexus_agent" {
			inv.AgentChunkCount++
		}
	}
	return inv, nil
}
