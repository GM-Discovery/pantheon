package pantheon

import "sort"

func applyPrimaryIndexDiff(old, rebuilt weaveIndex, diff weaveChangeReport) weaveIndex {
	affected := map[string]bool{}
	for _, paths := range [][]string{diff.New, diff.Changed, diff.Removed} {
		for _, p := range paths {
			affected[p] = true
		}
	}
	out := weaveIndex{PrimaryTermCounts: map[string]int{}}
	for _, p := range old.Passages {
		if !affected[p.Path] {
			out.Passages = append(out.Passages, p)
		}
	}
	for _, p := range rebuilt.Passages {
		if affected[p.Path] {
			out.Passages = append(out.Passages, p)
		}
	}
	sort.Slice(out.Passages, func(i, j int) bool { return out.Passages[i].Path < out.Passages[j].Path })
	for _, p := range out.Passages {
		seen := map[string]bool{}
		for _, term := range p.Terms {
			if !seen[term] {
				out.PrimaryTermCounts[term]++
				seen[term] = true
			}
		}
	}
	return out
}
