package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
)

type kernel18Fixture struct {
	Daemon     *daemon
	Root       string
	Watch      string
	Vault      string
	Model      string
	Whisper    string
	Invocation string
	Alert      string
}

func kernel18FixtureDaemon(t *testing.T) kernel18Fixture {
	t.Helper()
	d := kernel15Daemon(t, janusTestPackage("TEST:RUMOR", "expire"))
	root := filepath.Dir(d.databasePath)
	watch := filepath.Join(root, "audio-watch")
	vault := filepath.Join(root, "Vault")
	model := filepath.Join(root, "ggml-test.bin")
	whisper := filepath.Join(root, "whisper-helper")
	invocation := filepath.Join(root, "invocations.log")
	alert := filepath.Join(root, "alerts.log")
	for _, path := range []string{watch, filepath.Join(vault, "80 Pantheon Raw", "Audio Transcripts")} {
		if err := os.MkdirAll(path, 0o700); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(model, []byte("local test model"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(whisper, []byte("#!/bin/sh\nprintf '%s\\n' \"$*\" >> \""+invocation+"\"\nprintf '%s\\n' 'auto-detected language: es'\nprintf '%s\\n' '  Transcript output is preserved exactly.  '\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	// Keep the test's local alert adapter deterministic and side-effect free.
	alertCommand := filepath.Join(root, "alert-helper")
	if err := os.WriteFile(alertCommand, []byte("#!/bin/sh\nprintf '%s|%s|%s\\n' \"$1\" \"$2\" \"$3\" >> \""+alert+"\"\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	d.cfg.RumorEnabled = true
	d.cfg.RumorWatchPaths = []string{watch}
	d.cfg.RumorWhisperModel = model
	d.cfg.RumorLanguage = "auto"
	d.cfg.RumorRetryLimit = 3
	d.cfg.RumorTranscriptDestination = filepath.Join(vault, "80 Pantheon Raw", "Audio Transcripts")
	d.cfg.RumorOnComplete = "intake"
	d.cfg.RumorStabilitySeconds = 0
	d.cfg.RumorPollSeconds = 1
	d.cfg.VaultPath = vault
	d.cfg.RawPath = filepath.Join(vault, "80 Pantheon Raw")
	d.cfg.WhisperBinary = whisper
	d.cfg.WhisperModel = model
	oldNotify := notifyCommand
	notifyCommand = alertCommand
	t.Cleanup(func() { notifyCommand = oldNotify })
	return kernel18Fixture{Daemon: d, Root: root, Watch: watch, Vault: vault, Model: model, Whisper: whisper, Invocation: invocation, Alert: alert}
}

func kernel18Audio(t *testing.T, dir, name, contents string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(contents), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func kernel18InvocationCount(t *testing.T, path string) int {
	t.Helper()
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return 0
	}
	if err != nil {
		t.Fatal(err)
	}
	if strings.TrimSpace(string(data)) == "" {
		return 0
	}
	return len(strings.Split(strings.TrimSpace(string(data)), "\n"))
}

func TestKernel18ManualRumorIsLocalDurableAndIdempotent(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	audio := kernel18Audio(t, fixture.Watch, "meeting one.wav", "stable audio bytes")
	first, err := d.rumorTranscribePath(context.Background(), audio)
	if err != nil || first.State != rumorStateCompleted {
		t.Fatalf("first Rumor transcription=%#v err=%v", first, err)
	}
	if first.TranscriptPath == "" || first.MetadataPath == "" || first.IntakeState != "received" || first.IntakeOccurrence == "" {
		t.Fatalf("Rumor did not persist transcript, metadata, and Intake receipt: %#v", first)
	}
	second, err := d.rumorTranscribePath(context.Background(), audio)
	if err != nil || second.SourceID != first.SourceID || second.Attempts != 1 {
		t.Fatalf("repeat transcription was not idempotent: first=%#v second=%#v err=%v", first, second, err)
	}
	if count := kernel18InvocationCount(t, fixture.Invocation); count != 1 {
		t.Fatalf("Whisper invocation count=%d, want 1", count)
	}
	invocation, err := os.ReadFile(fixture.Invocation)
	if err != nil || !strings.Contains(string(invocation), "-m "+fixture.Model) || !strings.Contains(string(invocation), "-f "+audio) || !strings.Contains(string(invocation), "--output-txt") || !strings.Contains(string(invocation), "-l auto") {
		t.Fatalf("Whisper invocation did not use the local model/file/language contract: %q err=%v", invocation, err)
	}
	transcript, err := os.ReadFile(first.TranscriptPath)
	if err != nil || !strings.Contains(string(transcript), "Transcript output is preserved exactly.") {
		t.Fatalf("transcript artifact=%q err=%v", transcript, err)
	}
	metadata, err := os.ReadFile(first.MetadataPath)
	if err != nil {
		t.Fatal(err)
	}
	var document map[string]any
	if err := json.Unmarshal(metadata, &document); err != nil || document["source_hash"] != first.SourceHash {
		t.Fatalf("metadata=%s err=%v", metadata, err)
	}
	var occurrences int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE occurrence_id = ?`, first.IntakeOccurrence).Scan(&occurrences); err != nil || occurrences != 1 {
		t.Fatalf("Intake occurrence count=%d err=%v", occurrences, err)
	}
}

func TestKernel18SupportedFormatsMissingModelIgnoreAndConcurrentManualRequests(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	for _, extension := range []string{".wav", ".flac", ".mp3", ".ogg"} {
		if !rumorSupportedAudio("sample" + extension) {
			t.Fatalf("installed whisper format %s was not accepted", extension)
		}
	}
	if rumorSupportedAudio("sample.m4a") {
		t.Fatal("unsupported m4a was advertised without a converter contract")
	}
	ignoredPath := kernel18Audio(t, fixture.Watch, "ignore.wav", "ignore")
	ignored, err := d.rumorRegisterPath(ignoredPath, true)
	if err != nil {
		t.Fatal(err)
	}
	ignored, err = d.ignoreRumorSource(ignored.SourceID)
	if err != nil || ignored.State != rumorStateIgnored {
		t.Fatalf("ignored source=%#v err=%v", ignored, err)
	}
	if _, err := d.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if kernel18InvocationCount(t, fixture.Invocation) != 0 {
		t.Fatal("ignored source was transcribed")
	}

	concurrentPath := kernel18Audio(t, fixture.Watch, "concurrent.wav", "concurrent")
	var wait sync.WaitGroup
	results := make([]rumorSource, 2)
	errorsSeen := make([]error, 2)
	for index := range results {
		wait.Add(1)
		go func(index int) {
			defer wait.Done()
			results[index], errorsSeen[index] = d.rumorTranscribePath(context.Background(), concurrentPath)
		}(index)
	}
	wait.Wait()
	for index, err := range errorsSeen {
		if err != nil {
			t.Fatalf("concurrent request %d failed: %v", index, err)
		}
	}
	if results[0].SourceID != results[1].SourceID || results[0].Attempts != 1 || results[1].Attempts != 1 {
		t.Fatalf("concurrent requests created duplicate work: %#v", results)
	}
	if kernel18InvocationCount(t, fixture.Invocation) != 1 {
		t.Fatalf("concurrent request invocation count=%d, want 1", kernel18InvocationCount(t, fixture.Invocation))
	}

	missingModelPath := kernel18Audio(t, fixture.Watch, "missing-model.wav", "missing model")
	d.cfg.RumorWhisperModel = ""
	d.cfg.WhisperModel = ""
	d.capabilities = capabilityConfig{}
	missing, err := d.rumorTranscribePath(context.Background(), missingModelPath)
	if err == nil || missing.State != rumorStateRetryWait || missing.FailureKind != "transcription" || !strings.Contains(missing.LastError, "model") {
		t.Fatalf("missing local model was not a controlled failure: source=%#v err=%v", missing, err)
	}
}

func TestKernel18WhisperDiagnosticsAreNotPersistedAsTranscript(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	if err := os.WriteFile(fixture.Whisper, []byte("#!/bin/sh\nprintf '%s\\n' 'failed to read audio data' >&2\nexit 0\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	audio := kernel18Audio(t, fixture.Watch, "diagnostic-only.wav", "not audio")
	source, err := d.rumorTranscribePath(context.Background(), audio)
	if err == nil || source.State != rumorStateRetryWait || !strings.Contains(source.LastError, "no transcript") {
		t.Fatalf("diagnostic output became a transcript: source=%#v err=%v", source, err)
	}
	if source.TranscriptPath != "" || source.MetadataPath != "" {
		t.Fatalf("diagnostic-only run persisted artifacts: %#v", source)
	}
}

func TestKernel18TranscriptDestinationCannotEscapeVault(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	d.cfg.RumorTranscriptDestination = filepath.Join(fixture.Vault, "..", "outside-rumor")
	audio := kernel18Audio(t, fixture.Watch, "escape.wav", "escape")
	source, err := d.rumorTranscribePath(context.Background(), audio)
	if err == nil || source.State != rumorStateRetryWait || !strings.Contains(source.LastError, "beneath the configured Vault") {
		t.Fatalf("Vault escape was not refused: source=%#v err=%v", source, err)
	}
}

func TestKernel18CopiedAndRenamedBytesDoNotRetranscribeChangedBytesDo(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	original := kernel18Audio(t, fixture.Watch, "original.wav", "same bytes")
	first, err := d.rumorTranscribePath(context.Background(), original)
	if err != nil {
		t.Fatal(err)
	}
	copyPath := kernel18Audio(t, fixture.Watch, "renamed.wav", "same bytes")
	duplicate, err := d.rumorTranscribePath(context.Background(), copyPath)
	if err != nil || duplicate.State != rumorStateDuplicate || duplicate.DuplicateOf != first.SourceID {
		t.Fatalf("renamed identical source=%#v err=%v", duplicate, err)
	}
	if count := kernel18InvocationCount(t, fixture.Invocation); count != 1 {
		t.Fatalf("identical content retranscribed: %d invocations", count)
	}
	if err := os.WriteFile(original, []byte("changed bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	changed, err := d.rumorTranscribePath(context.Background(), original)
	if err != nil || changed.SourceID == first.SourceID || changed.SourceHash == first.SourceHash || changed.State != rumorStateCompleted {
		t.Fatalf("changed source was not a new completed identity: %#v err=%v", changed, err)
	}
	if count := kernel18InvocationCount(t, fixture.Invocation); count != 2 {
		t.Fatalf("changed content invocation count=%d, want 2", count)
	}
}

func TestKernel18WatcherRequiresStableFilesAndSkipsHiddenAndSymlinkAudio(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	d.cfg.RumorStabilitySeconds = 60
	stable := kernel18Audio(t, fixture.Watch, "stable.wav", "stable")
	if err := os.WriteFile(filepath.Join(fixture.Watch, ".hidden.wav"), []byte("hidden"), 0o600); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(fixture.Root, "outside.wav")
	if err := os.WriteFile(target, []byte("outside"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, filepath.Join(fixture.Watch, "linked.wav")); err != nil {
		t.Fatal(err)
	}
	first, err := d.rumorProcessCycle(context.Background())
	if err != nil || first.Transcribed != 0 {
		t.Fatalf("unstable watcher cycle=%#v err=%v", first, err)
	}
	source, err := d.loadLatestRumorPath(stable)
	if err != nil || source.State != rumorStateWaiting {
		t.Fatalf("stable file state=%#v err=%v", source, err)
	}
	var sources int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM rumor_sources`).Scan(&sources); err != nil {
		t.Fatal(err)
	}
	if sources != 1 {
		t.Fatalf("hidden/symlink files became Rumor sources: %d", sources)
	}
	d.cfg.RumorStabilitySeconds = 0
	if _, err := d.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if source, err := d.loadLatestRumorPath(stable); err != nil || source.State != rumorStateCompleted {
		t.Fatalf("stable file did not complete after stability gate: %#v err=%v", source, err)
	}
}

func TestKernel18PauseDoesNotReplayAndResumeProcessesOnce(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	audio := kernel18Audio(t, fixture.Watch, "paused.wav", "paused")
	paused, err := d.rumorProcessCycle(context.Background())
	if err != nil || !paused.Paused || kernel18InvocationCount(t, fixture.Invocation) != 0 {
		t.Fatalf("paused Rumor cycle=%#v invocations=%d err=%v", paused, kernel18InvocationCount(t, fixture.Invocation), err)
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	resumed, err := d.rumorProcessCycle(context.Background())
	if err != nil || resumed.Transcribed != 1 {
		t.Fatalf("resumed Rumor cycle=%#v err=%v", resumed, err)
	}
	if _, err := d.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	if kernel18InvocationCount(t, fixture.Invocation) != 1 {
		t.Fatalf("resume replayed audio: %d invocations", kernel18InvocationCount(t, fixture.Invocation))
	}
	if _, err := os.Stat(audio); err != nil {
		t.Fatalf("raw audio was removed: %v", err)
	}
}

func TestKernel18RetriesAreBoundedAlertedAndManuallyRecoverable(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	d.cfg.RumorRetryLimit = 2
	if err := os.WriteFile(fixture.Whisper, []byte("#!/bin/sh\nprintf '%s\\n' \"failed helper\" >&2\nexit 7\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	audio := kernel18Audio(t, fixture.Watch, "broken.wav", "broken")
	first, err := d.rumorTranscribePath(context.Background(), audio)
	if err == nil || first.State != rumorStateRetryWait || first.FailureKind != "transcription" {
		t.Fatalf("first retry state=%#v err=%v", first, err)
	}
	if _, err := d.db.Exec(`UPDATE rumor_sources SET next_attempt_at = '' WHERE source_id = ?`, first.SourceID); err != nil {
		t.Fatal(err)
	}
	second, err := d.rumorProcessCycle(context.Background())
	if second.Failed != 1 {
		t.Fatalf("bounded failure cycle=%#v err=%v", second, err)
	}
	failed, err := d.loadRumorSource(first.SourceID)
	if err != nil || failed.State != rumorStateFailed || !failed.AlertSent || failed.Attempts != 2 {
		t.Fatalf("paper-jam source=%#v err=%v", failed, err)
	}
	if alertCount := kernel18InvocationCount(t, fixture.Alert); alertCount != 1 {
		t.Fatalf("alert count=%d, want 1", alertCount)
	}
	if err := os.WriteFile(fixture.Whisper, []byte("#!/bin/sh\nprintf '%s\\n' 'auto-detected language: vi'\nprintf '%s\\n' 'recovered transcript'\n"), 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := d.retryRumorSource(first.SourceID); err != nil {
		t.Fatal(err)
	}
	if _, err := d.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	recovered, err := d.loadRumorSource(first.SourceID)
	if err != nil || recovered.State != rumorStateCompleted || recovered.DetectedLanguage != "vi" {
		t.Fatalf("recovered source=%#v err=%v", recovered, err)
	}
	if kernel18InvocationCount(t, fixture.Alert) != 1 {
		t.Fatal("manual retry duplicated the paper-jam alert")
	}
}

func TestKernel18IntakeHandoffFailureDoesNotRetranscribe(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	defer d.Close()
	blockedIntake := filepath.Join(fixture.Root, "intake-block")
	if err := os.WriteFile(blockedIntake, []byte("not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.cfg.IntakePath = blockedIntake
	d.cfg.IntakeConfigured = true
	audio := kernel18Audio(t, fixture.Watch, "handoff.wav", "handoff")
	source, err := d.rumorTranscribePath(context.Background(), audio)
	if err != nil || source.State != rumorStateCompleted || source.IntakeState != "pending" {
		t.Fatalf("handoff failure source=%#v err=%v", source, err)
	}
	if kernel18InvocationCount(t, fixture.Invocation) != 1 {
		t.Fatal("handoff failure retranscribed audio")
	}
	if err := os.Remove(blockedIntake); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(blockedIntake, 0o700); err != nil {
		t.Fatal(err)
	}
	if _, err := d.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	received, err := d.loadRumorSource(source.SourceID)
	if err != nil || received.IntakeState != "received" || received.IntakeOccurrence == "" {
		t.Fatalf("recovered handoff source=%#v err=%v", received, err)
	}
	if kernel18InvocationCount(t, fixture.Invocation) != 1 {
		t.Fatal("handoff recovery retranscribed audio")
	}
}

func TestKernel18RestartKeepsQueuedSourceAndHelpSurfacesRumor(t *testing.T) {
	fixture := kernel18FixtureDaemon(t)
	d := fixture.Daemon
	audio := kernel18Audio(t, fixture.Watch, "restart.wav", "restart")
	if _, err := d.rumorRegisterPath(audio, true); err != nil {
		t.Fatal(err)
	}
	databasePath := d.databasePath
	configPath := filepath.Join(filepath.Dir(databasePath), "pantheon.json")
	capsPath := filepath.Join(filepath.Dir(databasePath), "capabilities.json")
	if err := d.Close(); err != nil {
		t.Fatal(err)
	}
	reopened, err := newDaemon(daemonOptions{ConfigPath: configPath, CapabilitiesPath: capsPath, DatabasePath: databasePath})
	if err != nil {
		t.Fatal(err)
	}
	defer reopened.Close()
	reopened.cfg.RumorEnabled = true
	reopened.cfg.RumorWatchPaths = []string{fixture.Watch}
	reopened.cfg.RumorWhisperModel = fixture.Model
	reopened.cfg.RumorLanguage = "auto"
	reopened.cfg.RumorRetryLimit = 3
	reopened.cfg.RumorTranscriptDestination = filepath.Join(fixture.Vault, "80 Pantheon Raw", "Audio Transcripts")
	reopened.cfg.RumorOnComplete = "none"
	reopened.cfg.RumorStabilitySeconds = 0
	reopened.cfg.WhisperBinary = fixture.Whisper
	reopened.cfg.WhisperModel = fixture.Model
	if _, err := reopened.rumorProcessCycle(context.Background()); err != nil {
		t.Fatal(err)
	}
	source, err := reopened.loadLatestRumorPath(audio)
	if err != nil || source.State != rumorStateCompleted {
		t.Fatalf("restarted source=%#v err=%v", source, err)
	}
	if !strings.Contains(commandDiscoveryText(), "pantheon rumor transcribe <audio-path>") || !strings.Contains(commandDiscoveryText(), "pantheon show rumor queue") {
		t.Fatal("help discovery omitted Rumor controls")
	}
	request, err := http.NewRequest(http.MethodGet, "/v1/rumor/sources/"+source.SourceID, nil)
	if err != nil {
		t.Fatal(err)
	}
	recorder := httptest.NewRecorder()
	reopened.Handler().ServeHTTP(recorder, request)
	if recorder.Code != http.StatusOK {
		t.Fatalf("Rumor source inspection status=%d body=%s", recorder.Code, recorder.Body.String())
	}
}

func TestKernel18ConfiguredWatcherFailsSafeWhenEmpty(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:RUMOREMPTY", "expire"))
	defer d.Close()
	d.cfg.RumorEnabled = false
	d.cfg.RumorWatchPaths = nil
	result, err := d.rumorProcessCycle(context.Background())
	if err != nil || result.Scanned != 0 || result.Transcribed != 0 {
		t.Fatalf("disabled/empty watcher was not inert: %#v err=%v", result, err)
	}
	if _, err := d.rumorRegisterPath(filepath.Join(t.TempDir(), "missing.wav"), false); err == nil {
		t.Fatal("missing manual/watch path was accepted")
	}
}
