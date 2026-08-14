package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
)

type weaveSubjectReviewConfig struct {
	Subjects []weaveSubjectReviewSpec `json:"subjects"`
}
type weaveSubjectReviewSpec struct {
	Slug  string `json:"slug"`
	Query string `json:"query"`
}

func writeConfiguredSubjectReviews(configPath, stateDir string, index weaveIndex) error {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return err
	}
	var cfg weaveSubjectReviewConfig
	if err = json.Unmarshal(data, &cfg); err != nil {
		return err
	}
	dir := filepath.Join(stateDir, "reviews")
	if err = os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	for _, s := range cfg.Subjects {
		if s.Slug == "" || filepath.Base(s.Slug) != s.Slug || strings.ContainsAny(s.Slug, `/\\`) {
			return errors.New("subject review slug must be a single filename")
		}
		r := rankWeavePassages(s.Query, index.Passages, .30)
		if len(r) > 12 {
			r = r[:12]
		}
		path := filepath.Join(dir, s.Slug+".md")
		if err = os.WriteFile(path+".tmp", []byte(renderWeaveQueryReview(s.Query, r)), 0o600); err != nil {
			return err
		}
		if err = os.Rename(path+".tmp", path); err != nil {
			return err
		}
	}
	return nil
}
