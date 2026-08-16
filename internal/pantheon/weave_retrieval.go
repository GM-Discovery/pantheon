package pantheon

import (
	"sort"
	"strings"
)

type weaveRetrievalResult struct {
	Passage    weavePassage
	Confidence float64
}

func rankWeavePassages(query string, passages []weavePassage, minimum float64) []weaveRetrievalResult {
	terms := strings.Fields(strings.ToLower(query))
	if len(terms) == 0 {
		return nil
	}
	var results []weaveRetrievalResult
	for _, passage := range passages {
		text := strings.ToLower(passage.Heading + "\n" + passage.Text)
		matched := 0
		for _, term := range terms {
			if strings.Contains(text, term) {
				matched++
			}
		}
		confidence := float64(matched) / float64(len(terms))
		if confidence >= minimum {
			results = append(results, weaveRetrievalResult{Passage: passage, Confidence: confidence})
		}
	}
	sort.SliceStable(results, func(i, j int) bool {
		if results[i].Confidence != results[j].Confidence {
			return results[i].Confidence > results[j].Confidence
		}
		return weaveSourcePriority(results[i].Passage.Path) < weaveSourcePriority(results[j].Passage.Path)
	})
	return results
}

func weaveSourcePriority(path string) int {
	path = strings.ReplaceAll(path, "\\", "/")
	switch {
	case strings.HasPrefix(path, "20 Projects/Socio/Writing/"):
		return 0
	case strings.HasPrefix(path, "20 Projects/Socio/Drafts/"):
		return 1
	case strings.HasPrefix(path, "20 Projects/Socio/Reference/"):
		return 2
	default:
		return 3
	}
}
