package pantheon

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFameExtractionUsesConfiguredModelAndPersistsOnlyCitedObservations(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:FAMEEXTRACT", "expire"))
	defer d.Close()
	root := t.TempDir()
	vault := filepath.Join(root, "Vault")
	source := filepath.Join(root, "recording.m4a")
	diarization := filepath.Join(root, "run.tsv")
	if err := os.WriteFile(source, []byte("local audio fixture"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(diarization, []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t4.000\tSPEAKER_00\t4.000\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	transcript := filepath.Join(vault, "80 Pantheon Raw", "Audio", "2026", "08", "21", "recording.kernel24-safe-test.speaker-transcript.md")
	if err := os.MkdirAll(filepath.Dir(transcript), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(transcript, []byte("[00:00:00.000–00:00:04.000] **SPEAKER_00:** I will send the revised intake form by Friday.\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	run, err := d.importEchoRun(echoImportRequest{RunID: "echo-extract", SourcePath: source, DiarizationPath: diarization})
	if err != nil {
		t.Fatal(err)
	}
	conversation, err := d.createFameConversation(fameConversationRequest{RunID: run.RunID})
	if err != nil {
		t.Fatal(err)
	}
	d.cfg.VaultPath = vault
	d.cfg.ApolloExtractionModel = "qwen3.5:9b"
	d.cfg.ApolloMaxReasoningDurationSeconds = 5
	citation := fameExtractionCitation(vault, transcript, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var request fameOllamaRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Errorf("request JSON: %v", err)
		} else {
			if request.Model != "qwen3.5:9b" || request.Format != "json" || request.Think == nil || *request.Think {
				t.Errorf("extraction request controls = model %q format %q think=%v", request.Model, request.Format, request.Think)
			}
			if len(request.Messages) < 2 || !strings.Contains(request.Messages[1].Content, citation) || !strings.Contains(request.Messages[1].Content, "SPEAKER_00") {
				t.Errorf("extraction prompt omitted exact source context: %s", request.Messages[1].Content)
			}
		}
		responseContent := fmt.Sprintf("{\"action_items\":[{\"speaker\":\"SPEAKER_00\",\"action\":\"send the revised intake form\",\"time_constraint\":\"Friday\",\"evidence\":%q}],\"commitments\":[],\"decisions\":[],\"unresolved_questions\":[],\"uncertainties\":[]}", citation)
		_ = json.NewEncoder(w).Encode(ollamaResponse{Message: message{Role: "assistant", Content: responseContent}})
	}))
	defer server.Close()
	d.cfg.OllamaURL = server.URL
	result, err := d.extractFameConversation(context.Background(), conversation.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if result.Model != "qwen3.5:9b" || result.Status != "completed" || len(result.Observations) != 1 {
		t.Fatalf("extraction result=%+v", result)
	}
	observation := result.Observations[0]
	if observation.Kind != "action_item" || observation.SourceRef != citation || observation.ActorPersonID != "" || !strings.Contains(observation.Content, "Friday") {
		t.Fatalf("extracted observation=%+v", observation)
	}
	second, err := d.extractFameConversation(context.Background(), conversation.ConversationID)
	if err != nil {
		t.Fatal(err)
	}
	if len(second.Observations) != 0 {
		t.Fatalf("repeat extraction duplicated observations: %+v", second.Observations)
	}
}

func TestFameExtractionRejectsUnknownEvidence(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:FAMEEXTRACTREJECT", "expire"))
	defer d.Close()
	conversationID := "conversation-test"
	if _, err := d.db.Exec("INSERT INTO fame_conversations (conversation_id, source_path, boundary_state, status, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)", conversationID, "/tmp/transcript", fameBoundaryProvisional, "imported", "now", "now"); err != nil {
		t.Fatal(err)
	}
	_, err := d.persistFameExtraction(conversationID, fameExtractionOutput{ActionItems: []fameExtractionClaim{{Speaker: "SPEAKER_00", Action: "do not persist", Evidence: "[[not-supplied]]"}}}, "qwen3.5:9b", []fameExtractionSource{{Speaker: "SPEAKER_00", Citation: "[[supplied#00:00:00.000]]", Text: "source"}})
	if err == nil || !strings.Contains(err.Error(), "unknown source") {
		t.Fatalf("unknown evidence error=%v", err)
	}
}

func TestKernel25FameProcessConfirmsSpeakerAndExtracts(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL25PROCESS", "expire"))
	defer d.Close()
	root := t.TempDir()
	vault := filepath.Join(root, "Vault")
	source := filepath.Join(root, "recording.m4a")
	diarization := filepath.Join(root, "run.tsv")
	transcript := filepath.Join(vault, "80 Pantheon Raw", "Audio", "2026", "08", "22", "recording.kernel25-safe-test.speaker-transcript.md")
	for path, content := range map[string][]byte{
		source:      []byte("local audio fixture"),
		diarization: []byte("start_seconds\tend_seconds\tspeaker\tduration_seconds\n0.000\t4.000\tSPEAKER_00\t4.000\n4.000\t8.000\tSPEAKER_01\t4.000\n"),
		transcript:  []byte("[00:00:00.000–00:00:04.000] **SPEAKER_00:** I will send the revised intake form by Friday.\n[00:00:04.000–00:00:08.000] **SPEAKER_01:** That works.\n"),
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	d.cfg.VaultPath = vault
	d.cfg.ApolloExtractionModel = "qwen3.5:9b"
	d.cfg.ApolloMaxReasoningDurationSeconds = 5
	run, err := d.importEchoRun(echoImportRequest{RunID: "echo-kernel25-process", SourcePath: source, DiarizationPath: diarization})
	if err != nil {
		t.Fatal(err)
	}
	person, err := d.createFamePerson(famePersonRequest{PreferredName: "Trinh", Status: fameStatusConfirmed})
	if err != nil {
		t.Fatal(err)
	}
	citation := fameExtractionCitation(vault, transcript, 0)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		responseContent := fmt.Sprintf("{\"action_items\":[{\"speaker\":\"SPEAKER_00\",\"action\":\"send the revised intake form\",\"time_constraint\":\"Friday\",\"evidence\":%q}],\"commitments\":[],\"decisions\":[],\"unresolved_questions\":[],\"uncertainties\":[]}", citation)
		_ = json.NewEncoder(w).Encode(ollamaResponse{Message: message{Role: "assistant", Content: responseContent}})
	}))
	defer server.Close()
	d.cfg.OllamaURL = server.URL

	request := httptest.NewRequest(http.MethodPost, "/v1/fame/process", strings.NewReader(fmt.Sprintf(`{"run_id":"%s","confirmations":[{"speaker":"SPEAKER_00","person_id":"%s"}]}`, run.RunID, person.PersonID)))
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("process status=%d body=%s", response.Code, response.Body.String())
	}
	var result fameProcessResult
	if err := json.Unmarshal(response.Body.Bytes(), &result); err != nil {
		t.Fatal(err)
	}
	if result.Conversation.RunID != run.RunID || len(result.Corrections) != 1 || result.Extraction == nil || len(result.Extraction.Observations) != 1 {
		t.Fatalf("process result=%+v", result)
	}
	var assigned string
	if err := d.db.QueryRow(`SELECT person_id FROM fame_segment_attributions WHERE conversation_id = ? AND anonymous_speaker = ? LIMIT 1`, result.Conversation.ConversationID, "SPEAKER_00").Scan(&assigned); err != nil {
		t.Fatal(err)
	}
	if assigned != person.PersonID || result.Extraction.Observations[0].ActorPersonID != person.PersonID {
		t.Fatalf("speaker resolution assigned=%q observation=%+v", assigned, result.Extraction.Observations[0])
	}
}

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
	personResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(personResponse, httptest.NewRequest(http.MethodPost, "/v1/fame/command", strings.NewReader(`{"input":"/fame person add Trinh"}`)))
	if personResponse.Code != http.StatusCreated {
		t.Fatalf("person slash command status=%d body=%s", personResponse.Code, personResponse.Body.String())
	}
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, httptest.NewRequest(http.MethodPost, "/v1/fame/command", strings.NewReader(`{"input":"/fame people"}`)))
	if response.Code != http.StatusOK {
		t.Fatalf("slash command status=%d body=%s", response.Code, response.Body.String())
	}
	var body fameSnapshot
	if err := json.Unmarshal(response.Body.Bytes(), &body); err != nil || len(body.People) != 2 {
		t.Fatalf("slash command body=%s err=%v", response.Body.String(), err)
	}
}
