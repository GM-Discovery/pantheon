package main

func buildPrimaryWeaveIndex(vaultPath string) (weaveIndex, error) {
	all, err := buildWeaveIndex(vaultPath)
	if err != nil {
		return weaveIndex{}, err
	}
	out := weaveIndex{PrimaryTermCounts: map[string]int{}}
	for _, p := range all.Passages {
		if weaveCorpusDisposition(p.Path) != "primary" {
			continue
		}
		out.Passages = append(out.Passages, p)
		seen := map[string]bool{}
		for _, term := range p.Terms {
			if !seen[term] {
				out.PrimaryTermCounts[term]++
				seen[term] = true
			}
		}
	}
	return out, nil
}
