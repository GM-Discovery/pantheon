package main

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDaemonWeaveRetrieveReturnsCitedPastPassages(t *testing.T) {
	dir := t.TempDir()
	vault := filepath.Join(dir, "vault")
	for name, text := range map[string]string{
		"20 Projects/Socio/Writing/rules.md":      "# Rules\nBread Standard governs the common good.\n",
		"10 Operations/noise.md":                  "# Noise\nBread makes good toast.\n",
		"10 Operations/Email Triage/triage.md":    "# Triage\nBread Standard confidential detail.\n",
		"10 Operations/Situation Briefs/brief.md": "# Brief\nBread Standard confidential brief.\n",
		"20 Projects/Writings Inbox/inbox.md":     "# Inbox\nBread Standard confidential inbox.\n",
	} {
		path := filepath.Join(vault, name)
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	cfg := filepath.Join(dir, "pantheon.json")
	caps := filepath.Join(dir, "capabilities.json")
	if err := os.WriteFile(cfg, []byte(`{"local_model":"test","ollama_url":"http://127.0.0.1:11434/api/chat","vault_path":"`+vault+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(caps, []byte(`{"capabilities":[]}`), 0o600); err != nil {
		t.Fatal(err)
	}
	d, err := newDaemon(daemonOptions{ConfigPath: cfg, CapabilitiesPath: caps, DatabasePath: filepath.Join(dir, "state.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v1/weave/retrieve", strings.NewReader(`{"query":"bread standard","mode":"past"}`))
	d.Handler().ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if !strings.Contains(rr.Body.String(), "20 Projects/Socio/Writing/rules.md") || strings.Contains(rr.Body.String(), "toast") || strings.Contains(rr.Body.String(), "confidential") {
		t.Fatalf("unexpected retrieval: %s", rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), "terms") || strings.Contains(rr.Body.String(), "Nexus") {
		t.Fatalf("Past response leaked internals or Nexus material: %s", rr.Body.String())
	}
}
