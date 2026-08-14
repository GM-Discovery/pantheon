package main

import "sort"

type fingerprintDiff struct{ New, Changed, Removed []string }

func diffFingerprintStates(old, now weaveFingerprintState) fingerprintDiff {
	var d fingerprintDiff
	for p, v := range now.Files {
		if ov, ok := old.Files[p]; !ok {
			d.New = append(d.New, p)
		} else if ov != v {
			d.Changed = append(d.Changed, p)
		}
	}
	for p := range old.Files {
		if _, ok := now.Files[p]; !ok {
			d.Removed = append(d.Removed, p)
		}
	}
	sort.Strings(d.New)
	sort.Strings(d.Changed)
	sort.Strings(d.Removed)
	return d
}
