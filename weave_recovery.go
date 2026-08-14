package main

import (
	"sort"
)

func fullRebuildDiff(current weaveFingerprintState) weaveChangeReport {
	d := weaveChangeReport{Current: current, New: make([]string, 0, len(current.Files))}
	for path := range current.Files {
		d.New = append(d.New, path)
	}
	sort.Strings(d.New)
	return d
}
