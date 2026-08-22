package pantheon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEchoSeedsLocalMeProfileAndKeepsImportedSpeakersAnonymous(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHO", "expire"))
	defer d.Close()

	source := filepath.Join(t.TempDir(), "recording.wav")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	diarization := filepath.Join(t.TempDir(), "run.tsv")
	if err := os.WriteFile(diarization, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t1.000\tSPEAKER_00\t1.000\n1.000\t2.000\tSPEAKER_01\t1.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	snapshot, err := d.echoSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.Profiles) != 1 || snapshot.Profiles[0].VoiceID != "me" || snapshot.Profiles[0].State != echoVoiceAwaitingSample {
		t.Fatalf("initial Echo profiles = %#v", snapshot.Profiles)
	}
	run, err := d.importEchoRun(echoImportRequest{SourcePath: source, DiarizationPath: diarization})
	if err != nil {
		t.Fatal(err)
	}
	if run.SpeakerCount != 2 || run.SegmentCount != 2 {
		t.Fatalf("imported Echo run = %#v", run)
	}
	snapshot, err = d.echoSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.AnonymousSpeakers) != 2 || len(snapshot.Segments) != 2 {
		t.Fatalf("Echo snapshot after import = %#v", snapshot)
	}
	for _, segment := range snapshot.Segments {
		if segment.MappingState != echoMappingAnonymous || segment.VoiceID != "" {
			t.Fatalf("speaker was assigned without confirmation: %#v", segment)
		}
	}
}

func TestEchoVoiceEnrollmentStoresReferenceAndHashOnly(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHOENROLL", "expire"))
	defer d.Close()

	source := filepath.Join(t.TempDir(), "sample.m4a")
	if err := os.WriteFile(source, []byte("voice sample fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	sample, err := d.enrollEchoVoice("me", echoEnrollRequest{SourcePath: source, StartSeconds: 12, EndSeconds: 20})
	if err != nil {
		t.Fatal(err)
	}
	if sample.SourcePath != source || sample.StartSeconds != 12 || sample.EndSeconds != 20 || sample.SHA256 == "" {
		t.Fatalf("sample = %#v", sample)
	}
	var content string
	if err := d.db.QueryRow(`SELECT source_path FROM echo_voice_samples WHERE sample_id = ?`, sample.SampleID).Scan(&content); err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(content, "sample.m4a") {
		t.Fatalf("stored sample reference = %q", content)
	}
	profiles, err := readEchoProfiles(d.db)
	if err != nil {
		t.Fatal(err)
	}
	if profiles[0].State != echoVoiceEnrolled || profiles[0].SampleCount != 1 {
		t.Fatalf("profile after enrollment = %#v", profiles[0])
	}
}

func TestEchoHTTPSurfaceAndSlashCommand(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHOHTTP", "expire"))
	defer d.Close()

	for _, path := range []string{"/v1/echo", "/v1/gui/echo"} {
		recorder := httptest.NewRecorder()
		d.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, path, nil))
		if recorder.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, recorder.Code, recorder.Body.String())
		}
	}
	recorder := httptest.NewRecorder()
	d.Handler().ServeHTTP(recorder, httptest.NewRequest(http.MethodPost, "/v1/echo/command", strings.NewReader(`{"input":"/echo voices"}`)))
	if recorder.Code != http.StatusOK {
		t.Fatalf("slash command status=%d body=%s", recorder.Code, recorder.Body.String())
	}
	var body echoSnapshot
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil || len(body.Profiles) != 1 {
		t.Fatalf("slash command body=%s err=%v", recorder.Body.String(), err)
	}
}

func TestEchoReadsSpeakerTranscriptSnippetsByTimestamp(t *testing.T) {
	path := filepath.Join(t.TempDir(), "speaker-transcript.md")
	content := "# Transcript\n\n[00:00:01.440–00:00:03.120] **SPEAKER_00:** Did you send the documents?\n[00:00:03.980–00:00:09.030] **SPEAKER_01:** Then if there is no documentation.\n"
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	snippets, err := readEchoTranscript(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(snippets) != 2 {
		t.Fatalf("parsed snippets=%d", len(snippets))
	}
	text := echoTranscriptSnippetForSegment(snippets, echoSpeakerSegment{AnonymousSpeaker: "SPEAKER_00", StartSeconds: 1.482, EndSeconds: 2.100})
	if text != "Did you send the documents?" {
		t.Fatalf("matched snippet=%q", text)
	}
}

func TestEchoIdentityReviewChangesOneSegmentAndSupportsRenameAndClear(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHOREVIEW", "expire"))
	defer d.Close()

	source := filepath.Join(t.TempDir(), "recording.m4a")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	diarization := filepath.Join(t.TempDir(), "run.tsv")
	if err := os.WriteFile(diarization, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t1.000\tSPEAKER_00\t1.000\n1.000\t2.000\tSPEAKER_00\t1.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run, err := d.importEchoRun(echoImportRequest{RunID: "echo-review", SourcePath: source, DiarizationPath: diarization})
	if err != nil {
		t.Fatal(err)
	}
	profile, err := d.createEchoVoice(echoVoiceRequest{Name: "Review Voice"})
	if err != nil {
		t.Fatal(err)
	}

	confirm := httptest.NewRecorder()
	d.Handler().ServeHTTP(confirm, httptest.NewRequest(http.MethodPost, "/v1/echo/segments/echo-review-000000/confirm", strings.NewReader(`{"voice_id":"review-voice"}`)))
	if confirm.Code != http.StatusOK {
		t.Fatalf("confirm status=%d body=%s", confirm.Code, confirm.Body.String())
	}
	var first, second string
	if err := d.db.QueryRow(`SELECT mapping_state FROM echo_speaker_segments WHERE segment_id = ?`, "echo-review-000000").Scan(&first); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT mapping_state FROM echo_speaker_segments WHERE segment_id = ?`, "echo-review-000001").Scan(&second); err != nil {
		t.Fatal(err)
	}
	if first != echoMappingConfirmed || second != echoMappingAnonymous {
		t.Fatalf("segment states after one confirmation = %q, %q", first, second)
	}

	rename := httptest.NewRecorder()
	d.Handler().ServeHTTP(rename, httptest.NewRequest(http.MethodPost, "/v1/echo/voices/"+profile.VoiceID+"/rename", strings.NewReader(`{"name":"Renamed Voice"}`)))
	if rename.Code != http.StatusOK {
		t.Fatalf("rename status=%d body=%s", rename.Code, rename.Body.String())
	}
	var name string
	if err := d.db.QueryRow(`SELECT name FROM echo_voice_profiles WHERE voice_id = ?`, profile.VoiceID).Scan(&name); err != nil {
		t.Fatal(err)
	}
	if name != "Renamed Voice" {
		t.Fatalf("renamed profile=%q", name)
	}

	clear := httptest.NewRecorder()
	d.Handler().ServeHTTP(clear, httptest.NewRequest(http.MethodPost, "/v1/echo/segments/echo-review-000000/clear", strings.NewReader(`{}`)))
	if clear.Code != http.StatusOK {
		t.Fatalf("clear status=%d body=%s", clear.Code, clear.Body.String())
	}
	if err := d.db.QueryRow(`SELECT mapping_state, COALESCE(voice_id,'') FROM echo_speaker_segments WHERE segment_id = ?`, "echo-review-000000").Scan(&first, &second); err != nil {
		t.Fatal(err)
	}
	if first != echoMappingAnonymous || second != "" {
		t.Fatalf("segment state after clear = %q, voice=%q", first, second)
	}
	_ = run
}

func TestEchoRunSelectionAndVoiceArchive(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHORUNS", "expire"))
	defer d.Close()

	source := filepath.Join(t.TempDir(), "recording.m4a")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	diarizationOne := filepath.Join(t.TempDir(), "run-one.tsv")
	if err := os.WriteFile(diarizationOne, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t1.000\tSPEAKER_00\t1.000\n1.000\t2.000\tSPEAKER_01\t1.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	diarizationTwo := filepath.Join(t.TempDir(), "run-two.tsv")
	if err := os.WriteFile(diarizationTwo, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t1.000\tSPEAKER_00\t1.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.importEchoRun(echoImportRequest{RunID: "echo-run-one", SourcePath: source, DiarizationPath: diarizationOne}); err != nil {
		t.Fatal(err)
	}
	if _, err := d.importEchoRun(echoImportRequest{RunID: "echo-run-two", SourcePath: source, DiarizationPath: diarizationTwo}); err != nil {
		t.Fatal(err)
	}

	snapshot, err := d.echoSnapshotForRun("echo-run-one")
	if err != nil {
		t.Fatal(err)
	}
	if snapshot.SelectedRunID != "echo-run-one" || snapshot.LatestRun == nil || snapshot.LatestRun.RunID != "echo-run-one" || len(snapshot.AnonymousSpeakers) != 2 {
		t.Fatalf("selected Echo run = %#v", snapshot)
	}
	audio := httptest.NewRecorder()
	d.Handler().ServeHTTP(audio, httptest.NewRequest(http.MethodGet, "/v1/echo/runs/echo-run-one/audio", nil))
	if audio.Code != http.StatusOK || audio.Body.Len() == 0 {
		t.Fatalf("audio preview status=%d bytes=%d", audio.Code, audio.Body.Len())
	}

	profile, err := d.createEchoVoice(echoVoiceRequest{Name: "Accidental profile"})
	if err != nil {
		t.Fatal(err)
	}
	archive := httptest.NewRecorder()
	d.Handler().ServeHTTP(archive, httptest.NewRequest(http.MethodPost, "/v1/echo/voices/"+profile.VoiceID+"/archive", strings.NewReader(`{}`)))
	if archive.Code != http.StatusOK {
		t.Fatalf("archive status=%d body=%s", archive.Code, archive.Body.String())
	}
	profiles, err := readEchoProfiles(d.db)
	if err != nil {
		t.Fatal(err)
	}
	for _, item := range profiles {
		if item.VoiceID == profile.VoiceID {
			t.Fatalf("archived profile remained active: %#v", profiles)
		}
	}
	var state string
	if err := d.db.QueryRow(`SELECT state FROM echo_voice_profiles WHERE voice_id = ?`, profile.VoiceID).Scan(&state); err != nil {
		t.Fatal(err)
	}
	if state != echoVoiceDisabled {
		t.Fatalf("archived profile state=%q", state)
	}
}

func TestEchoBulkConfirmationIsScopedToOneRunAndSpeaker(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:ECHOBULK", "expire"))
	defer d.Close()
	source := filepath.Join(t.TempDir(), "bulk-audio.m4a")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	makeRun := func(id string) echoRun {
		diarization := filepath.Join(t.TempDir(), id+".tsv")
		if err := os.WriteFile(diarization, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t2.000\tSPEAKER_00\t2.000\n2.000\t4.000\tSPEAKER_01\t2.000\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		run, err := d.importEchoRun(echoImportRequest{RunID: id, SourcePath: source, DiarizationPath: diarization})
		if err != nil {
			t.Fatal(err)
		}
		return run
	}
	first := makeRun("echo-bulk-one")
	second := makeRun("echo-bulk-two")
	profile, err := d.createEchoVoice(echoVoiceRequest{Name: "Bulk Person"})
	if err != nil {
		t.Fatal(err)
	}
	record := httptest.NewRecorder()
	d.Handler().ServeHTTP(record, httptest.NewRequest(http.MethodPost, "/v1/echo/voices/"+profile.VoiceID+"/confirm", strings.NewReader(`{"run_id":"echo-bulk-one","speaker":"SPEAKER_00"}`)))
	if record.Code != http.StatusOK {
		t.Fatalf("bulk confirmation status=%d body=%s", record.Code, record.Body.String())
	}
	var response map[string]any
	if err := json.Unmarshal(record.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response["confirmed"] != float64(1) {
		t.Fatalf("bulk confirmation response=%s", record.Body.String())
	}
	var target, otherSpeaker, otherRun int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM echo_speaker_segments WHERE run_id = ? AND anonymous_speaker = ? AND voice_id = ?`, first.RunID, "SPEAKER_00", profile.VoiceID).Scan(&target); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM echo_speaker_segments WHERE run_id = ? AND anonymous_speaker = ? AND voice_id IS NULL`, first.RunID, "SPEAKER_01").Scan(&otherSpeaker); err != nil {
		t.Fatal(err)
	}
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM echo_speaker_segments WHERE run_id = ? AND voice_id = ?`, second.RunID, profile.VoiceID).Scan(&otherRun); err != nil {
		t.Fatal(err)
	}
	if target != 1 || otherSpeaker != 1 || otherRun != 0 {
		t.Fatalf("bulk scope target=%d other speaker remaining=%d other run assigned=%d", target, otherSpeaker, otherRun)
	}
}
