package main

import "sort"

func applyNexusChunkDiff(old, rebuilt nexusChunkInventory, diff fingerprintDiff) nexusChunkInventory {
	replace := map[string]bool{}
	for _, paths := range [][]string{diff.New, diff.Changed, diff.Removed} {
		for _, p := range paths {
			replace[p] = true
		}
	}
	var out nexusChunkInventory
	for _, c := range old.Chunks {
		if !replace[c.Path] {
			out.Chunks = append(out.Chunks, c)
		}
	}
	for _, c := range rebuilt.Chunks {
		if replace[c.Path] {
			out.Chunks = append(out.Chunks, c)
		}
	}
	sort.Slice(out.Chunks, func(i, j int) bool { return out.Chunks[i].Path < out.Chunks[j].Path })
	for _, c := range out.Chunks {
		if c.Authorship == "nexus_user" {
			out.UserChunkCount++
		} else if c.Authorship == "nexus_agent" {
			out.AgentChunkCount++
		}
	}
	return out
}
