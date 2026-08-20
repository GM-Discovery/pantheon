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
