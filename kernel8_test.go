package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKernel8LegacyConfigLoads(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pantheon.json")
	writeTestFile(t, path, `{"model":"apollo","ollama_url":"http://172.17.0.1:11434/api/chat","frontier_model":"frontier","vault_path":"/tmp/vault","roadmap_path":"/tmp/roadmap"}`)

	resolution, err := readConfigResolution(path)
	if err != nil {
		t.Fatal(err)
	}
	if !resolution.Legacy || resolution.Config.LocalModel != "apollo" || resolution.Config.FrontierModel != "frontier" {
		t.Fatalf("unexpected legacy resolution: %#v", resolution)
	}
}

func TestKernel8CurrentConfigLoadsAndAppliesDefaults(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pantheon.json")
	writeTestFile(t, path, `{"schema_version":1,"models":{"apollo":{"model":"apollo","endpoint":"http://127.0.0.1:11434/api/chat"}}}`)

	resolution, err := readConfigResolution(path)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Legacy || resolution.SchemaVersion != 1 {
		t.Fatalf("unexpected current resolution: %#v", resolution)
	}
	if resolution.Config.ListenAddress != defaultListenAddress {
		t.Fatalf("listen default = %q", resolution.Config.ListenAddress)
	}
	if resolution.Sources["runtime.listen"] != "default" {
		t.Fatalf("listen source = %q", resolution.Sources["runtime.listen"])
	}
}

func TestKernel8ConfigRejectsUnsupportedMalformedAndIncompleteDocuments(t *testing.T) {
	tests := map[string]struct {
		contents string
		want     string
	}{
		"unsupported schema": {`{"schema_version":2}`, "unsupported schema_version"},
		"malformed JSON":     {`{"schema_version":`, "malformed JSON"},
		"missing model":      {`{"schema_version":1,"models":{"apollo":{"endpoint":"http://127.0.0.1:11434/api/chat"}}}`, "models.apollo.model is required"},
		"remote endpoint":    {`{"schema_version":1,"models":{"apollo":{"model":"apollo","endpoint":"http://8.8.8.8/api/chat"}}}`, "approved local HTTP transport"},
	}
	for name, test := range tests {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(t.TempDir(), "pantheon.json")
			writeTestFile(t, path, test.contents)
			_, err := readConfigResolution(path)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestKernel8ResolvedConfigOmitsSecrets(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pantheon.json")
	writeTestFile(t, path, `{"schema_version":1,"models":{"apollo":{"model":"apollo","endpoint":"http://127.0.0.1:11434/api/chat"}}}`)
	resolution, err := validateConfigFile(path)
	if err != nil {
		t.Fatal(err)
	}
	output := renderResolvedConfig(path, resolution)
	if strings.Contains(output, "control-token") || strings.Contains(output, "secret") && !strings.Contains(output, "secrets: omitted") {
		t.Fatalf("resolved configuration exposed a secret-like value: %s", output)
	}
}

func TestKernel8MigrationDryRunApplyAndIdempotency(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "pantheon.json")
	legacy := `{"local_model":"apollo","ollama_url":"http://127.0.0.1:11434/api/chat","vault_path":"/tmp/vault"}`
	writeTestFile(t, path, legacy)

	dryRun, err := migrateConfig(path, false)
	if err != nil {
		t.Fatal(err)
	}
	if dryRun.Noop || dryRun.Applied || len(dryRun.Proposed) == 0 {
		t.Fatalf("unexpected dry-run result: %#v", dryRun)
	}
	contents, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(contents) != legacy {
		t.Fatal("migration dry run changed the source")
	}

	applied, err := migrateConfig(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if !applied.Applied || applied.BackupPath == "" {
		t.Fatalf("unexpected apply result: %#v", applied)
	}
	if _, err := os.Stat(applied.BackupPath); err != nil {
		t.Fatalf("backup missing: %v", err)
	}
	resolution, err := validateConfigFile(path)
	if err != nil || resolution.Legacy {
		t.Fatalf("migrated config = %#v, err=%v", resolution, err)
	}
	idempotent, err := migrateConfig(path, true)
	if err != nil {
		t.Fatal(err)
	}
	if !idempotent.Noop || idempotent.Applied {
		t.Fatalf("migration was not idempotent: %#v", idempotent)
	}
}

func TestKernel8MigrationRefusesUnknownLegacyFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "pantheon.json")
	writeTestFile(t, path, `{"local_model":"apollo","ollama_url":"http://127.0.0.1:11434/api/chat","unknown":"preserve me"}`)
	if _, err := migrateConfig(path, false); err == nil || !strings.Contains(err.Error(), "unrecognized legacy fields") {
		t.Fatalf("unexpected migration result: %v", err)
	}
}

func TestKernel8ProtocolPauseWorksWithoutDaemon(t *testing.T) {
	marker := filepath.Join(t.TempDir(), prometheusPauseMarkerName)
	service := &fakeKernel8Service{state: "inactive"}
	output, err := protocolPause(service, "http://127.0.0.1:1", marker)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "persisted locally") {
		t.Fatalf("pause output = %s", output)
	}
	if !pauseMarkerExists(marker) {
		t.Fatal("pause marker was not persisted")
	}
}

func TestKernel8ProtocolResumeStartsStoppedServiceAndClearsState(t *testing.T) {
	dir := t.TempDir()
	dbPath := filepath.Join(dir, "pantheon.db")
	cfgPath := filepath.Join(dir, "pantheon.json")
	capsPath := filepath.Join(dir, "capabilities.json")
	writeTestFile(t, cfgPath, `{"local_model":"apollo","ollama_url":"http://127.0.0.1:1"}`)
	writeTestFile(t, capsPath, `{"capabilities":[]}`)
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	if err := d.setPaused(true); err != nil {
		d.Close()
		t.Fatal(err)
	}
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}

	service := &fakeKernel8Service{state: "inactive"}
	output, err := runPrometheusProtocolWith("resume", service, "http://127.0.0.1:1", pauseMarkerPathForDatabase(dbPath), dbPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(output, "user service") || service.startCalls != 1 {
		t.Fatalf("output=%s startCalls=%d", output, service.startCalls)
	}
	if pauseMarkerExists(pauseMarkerPathForDatabase(dbPath)) {
		t.Fatal("pause marker remains after resume")
	}
	reopened, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: dbPath})
	if err != nil {
		t.Fatal(err)
	}
	paused, err := reopened.paused()
	_ = reopened.Close()
	if err != nil || paused {
		t.Fatalf("persisted pause state = %t, err=%v", paused, err)
	}
}

func TestKernel8ProtocolHardStopUsesServiceBoundary(t *testing.T) {
	service := &fakeKernel8Service{state: "active"}
	output, err := protocolHardStop(service, "http://127.0.0.1:1")
	if err != nil {
		t.Fatal(err)
	}
	if service.stopCalls != 1 || !strings.Contains(output, "hard-stopped") {
		t.Fatalf("output=%s stopCalls=%d", output, service.stopCalls)
	}
}

func TestKernel8PauseBlocksReasoningBeforeModelCall(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "pantheon.json")
	capsPath := filepath.Join(dir, "capabilities.json")
	writeTestFile(t, cfgPath, `{"local_model":"apollo","ollama_url":"http://127.0.0.1:1"}`)
	writeTestFile(t, capsPath, `{"capabilities":[{"name":"reason","type":"model","risk":"low"}]}`)
	d, err := newDaemon(daemonOptions{ConfigPath: cfgPath, CapabilitiesPath: capsPath, DatabasePath: filepath.Join(dir, "pantheon.db")})
	if err != nil {
		t.Fatal(err)
	}
	defer d.Close()
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, newJSONRequest(t, http.MethodPost, "/v1/input", inputRequest{Input: "reason"}))
	if response.Code != http.StatusServiceUnavailable {
		t.Fatalf("status=%d body=%s", response.Code, response.Body.String())
	}
}

type fakeKernel8Service struct {
	state      string
	startCalls int
	stopCalls  int
	statusErr  error
}

func (f *fakeKernel8Service) status(context.Context) (serviceStatus, error) {
	if f.statusErr != nil {
		return serviceStatus{}, f.statusErr
	}
	return serviceStatus{State: f.state, Available: true}, nil
}

func (f *fakeKernel8Service) start(context.Context) error {
	f.startCalls++
	f.state = "active"
	return nil
}

func (f *fakeKernel8Service) stop(context.Context) error {
	f.stopCalls++
	f.state = "inactive"
	return nil
}

func writeTestFile(t *testing.T, path, contents string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
}
