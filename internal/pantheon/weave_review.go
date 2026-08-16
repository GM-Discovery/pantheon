package pantheon

import (
	"fmt"
	"sort"
	"strings"
)

func renderWeavePrimaryReview(index weaveIndex) string {
	var b strings.Builder
	b.WriteString("# Weave Primary Review\n\n> Generated navigation surface, not a document to read end-to-end. It makes no claim beyond cited primary-source passages. Use a specific query with `pantheon` retrieval to zoom into short excerpts and context windows.\n\n")
	sources := map[string]int{}
	for _, p := range index.Passages {
		sources[p.Path]++
	}
	fmt.Fprintf(&b, "## Index scope\n\n- Primary source notes: %d\n- Passage units: %d\n- This page intentionally does not repeat every excerpt. The full local index is under Pantheon state.\n\n## Included sources\n\n", len(sources), len(index.Passages))
	paths := make([]string, 0, len(sources))
	for path := range sources {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	for _, path := range paths {
		fmt.Fprintf(&b, "- [[%s|%s]] — %d passages\n", path, path, sources[path])
	}
	return b.String()
}
