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

func TestKernel24AFamePreservesIdentityEvidenceAndCorrectionHistory(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:FAME", "expire"))
	defer d.Close()

	source := filepath.Join(t.TempDir(), "recording-2.m4a")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	diarization := filepath.Join(t.TempDir(), "run.tsv")
	if err := os.WriteFile(diarization, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t4.000\tSPEAKER_00\t4.000\n4.000\t8.000\tSPEAKER_01\t4.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run, err := d.importEchoRun(echoImportRequest{RunID: "echo-recording-2", SourcePath: source, DiarizationPath: diarization})
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := d.createFameConversation(fameConversationRequest{RunID: run.RunID})
	if err != nil {
		t.Fatal(err)
	}
	if conversation.BoundaryState != fameBoundaryProvisional {
		t.Fatalf("conversation boundary=%q", conversation.BoundaryState)
	}

	trinh, err := d.createFamePerson(famePersonRequest{PreferredName: "Trinh"})
	if err != nil {
		t.Fatal(err)
	}
	if trinh.Status != fameStatusProvisional {
		t.Fatalf("new person status=%q", trinh.Status)
	}
	evidence, err := d.addFameEvidence(trinh.PersonID, fameEvidenceRequest{Kind: "operator_context", SourceRef: "recording-2:user-note", Detail: "User identifies Trinh as the speaker associated with the Lisa request; diarization label remains inspectable.", Confidence: 1}, "")
	if err != nil {
		t.Fatal(err)
	}
	if evidence.PersonID != trinh.PersonID || evidence.Confidence != 1 {
		t.Fatalf("identity evidence=%+v", evidence)
	}

	correction, err := d.createFameCorrection(fameCorrectionRequest{
		TargetType: "interaction",
		Kind:       "speaker_identity_context",
		Statement:  "Trinh said the line about Lisa requesting one of us; do not infer that every anonymous segment in this recording is Trinh.",
	})
	if err != nil {
		t.Fatal(err)
	}
	if correction.CorrectionID == "" {
		t.Fatal("correction has no durable ID")
	}

	observation, err := d.createFameObservation(fameObservationRequest{
		ConversationID: conversation.ConversationID,
		Kind:           "request",
		Content:        "Lisa requested that one of us accompany her to the apartment Friday at eleven.",
		ActorPersonID:  trinh.PersonID,
		SourceRef:      "echo-recording-2:SPEAKER_00:00:00-00:04:00",
		StartSeconds:   0,
		EndSeconds:     4,
		Confidence:     0.82,
	})
	if err != nil {
		t.Fatal(err)
	}
	if observation.ActorPersonID != trinh.PersonID || observation.Status != "accountable_observation" {
		t.Fatalf("observation=%+v", observation)
	}

	snapshot, err := d.fameSnapshot()
	if err != nil {
		t.Fatal(err)
	}
	if len(snapshot.People) != 2 || len(snapshot.Conversations) != 1 || len(snapshot.Attributions) != 2 || len(snapshot.Observations) != 1 || len(snapshot.Corrections) != 1 {
		t.Fatalf("Fame snapshot counts: people=%d conversations=%d attributions=%d observations=%d corrections=%d", len(snapshot.People), len(snapshot.Conversations), len(snapshot.Attributions), len(snapshot.Observations), len(snapshot.Corrections))
	}
	for _, attribution := range snapshot.Attributions {
		if attribution.State != fameAttributionAnonymous || attribution.PersonID != "" {
			t.Fatalf("prose correction silently changed attribution: %+v", attribution)
		}
	}
}

func TestFameHTTPSurfaceAndSlashCommand(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:FAMEHTTP", "expire"))
	defer d.Close()

	for _, path := range []string{"/v1/fame", "/v1/gui/fame", "/v1/fame/people"} {
		response := httptest.NewRecorder()
		d.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodGet, path, nil))
		if response.Code != http.StatusOK {
			t.Fatalf("GET %s status=%d body=%s", path, response.Code, response.Body.String())
		}
	}
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/fame/command", strings.NewReader(`{"input":"/fame people"}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("slash command status=%d body=%s", response.Code, response.Body.String())
	}
	var body fameSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.People) != 1 {
		t.Fatalf("slash command body=%s err=%v", response.Body.String(), err)
	}
}
