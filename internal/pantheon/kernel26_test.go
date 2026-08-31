package pantheon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestKernel26ExternalOmiIntakePreservesAndConverts(t *testing.T) {
	t.Setenv(externalIntakeSecretEnv, "kernel26-secret")
	d := kernel15Daemon(t, janusTestPackage("KERNEL26:INTAKE", "expire"))
	defer d.Close()
	d.cfg.VaultPath = filepath.Join(t.TempDir(), "vault")
	d.cfg.RawPath = filepath.Join(d.cfg.VaultPath, "80 Pantheon Raw")
	if err := os.MkdirAll(d.cfg.RawPath, 0o700); err != nil {
		t.Fatal(err)
	}

	payload := json.RawMessage(`{"memory_id":"mem-001","numbers":[7,42],"labels":["personal","trinh"],"transcript":"full payload remains here"}`)
	envelope := externalIntakeEnvelope{
		SourceKind: "omi_memory", SourceID: "mem-001", SourceEventID: "evt-001",
		OccurredAt: "2026-08-23T10:15:00Z", Payload: payload,
		Transcript: "Trinh: The transcript is retained and converted.\nSelf: Confirmed.",
		Labels:     []string{"personal", "trinh"},
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/external", bytes.NewReader(body))
	req.Header.Set("X-Pantheon-Intake-Secret", "kernel26-secret")
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("external Intake status = %d, body = %s", response.Code, response.Body.String())
	}
	var receipt externalIntakeReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != "completed" || receipt.RawOccurrenceID == "" || receipt.TranscriptOccurrenceID == "" || receipt.NormalizedPath == "" {
		t.Fatalf("incomplete receipt: %+v", receipt)
	}
	raw, err := os.ReadFile(filepath.Join(d.cfg.IntakePath, filepath.FromSlash(receipt.RawPayloadPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, payload) {
		t.Fatalf("raw payload changed: got %s", raw)
	}
	envelopeArchive, err := os.ReadFile(filepath.Join(d.cfg.IntakePath, filepath.FromSlash(receipt.EnvelopePath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envelopeArchive), `"labels":["personal","trinh"]`) {
		t.Fatalf("envelope labels were not preserved: %s", envelopeArchive)
	}
	transcript, err := os.ReadFile(receipt.NormalizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(transcript), "Trinh: The transcript is retained and converted.") {
		t.Fatalf("normalized transcript omitted source text: %s", transcript)
	}

	duplicateReq := httptest.NewRequest(http.MethodPost, "/v1/intake/external", bytes.NewReader(body))
	duplicateReq.Header.Set("Authorization", "Bearer kernel26-secret")
	duplicateResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(duplicateResponse, duplicateReq)
	if duplicateResponse.Code != http.StatusOK || !strings.Contains(duplicateResponse.Body.String(), `"status":"duplicate"`) {
		t.Fatalf("duplicate response = %d %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}
	conflicting := envelope
	conflicting.Payload = json.RawMessage(`{"memory_id":"mem-001","numbers":[7,99]}`)
	conflictBody, err := json.Marshal(conflicting)
	if err != nil {
		t.Fatal(err)
	}
	conflictReq := httptest.NewRequest(http.MethodPost, "/v1/intake/external", bytes.NewReader(conflictBody))
	conflictReq.Header.Set("X-Pantheon-Intake-Secret", "kernel26-secret")
	conflictResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(conflictResponse, conflictReq)
	if conflictResponse.Code != http.StatusConflict {
		t.Fatalf("conflicting replay status = %d, body = %s", conflictResponse.Code, conflictResponse.Body.String())
	}

	getReq := httptest.NewRequest(http.MethodGet, "/v1/intake/external/mem-001", nil)
	getReq.Header.Set("X-Pantheon-Intake-Secret", "kernel26-secret")
	getResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(getResponse, getReq)
	if getResponse.Code != http.StatusOK || !strings.Contains(getResponse.Body.String(), receipt.ReceiptID) {
		t.Fatalf("receipt lookup = %d %s", getResponse.Code, getResponse.Body.String())
	}
}

func TestKernel26ExternalOmiIntakeRejectsUnauthorized(t *testing.T) {
	t.Setenv(externalIntakeSecretEnv, "kernel26-secret")
	d := kernel15Daemon(t, janusTestPackage("KERNEL26:REPLAY", "expire"))
	defer d.Close()
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/external", strings.NewReader(`{"source_kind":"omi_memory","source_id":"mem-unauthorized","payload":{"a":1}}`))
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusUnauthorized {
		t.Fatalf("unauthorized status = %d", response.Code)
	}
}

func TestKernel26NativeOmiWebhookPreservesMemory(t *testing.T) {
	t.Setenv(externalIntakeSecretEnv, "kernel26-omi-secret")
	d := kernel15Daemon(t, janusTestPackage("KERNEL26:OMI", "expire"))
	defer d.Close()
	d.cfg.VaultPath = filepath.Join(t.TempDir(), "vault")
	d.cfg.RawPath = filepath.Join(d.cfg.VaultPath, "80 Pantheon Raw")
	if err := os.MkdirAll(d.cfg.RawPath, 0o700); err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{"id":"memory-omi-001","started_at":"2026-08-23T10:15:00+00:00","created_at":"2026-08-23T10:16:00+00:00","finished_at":"2026-08-23T10:17:00+00:00","transcript_segments":[{"text":"Let's keep the raw Omi memory.","speaker":"SPEAKER_00","speakerId":0,"speaker_name":"Trinh","start":10.25,"end":14.5},{"text":"Agreed.","speaker":"SPEAKER_01","speakerId":1,"is_user":true,"start":15,"end":16}],"structured":{"title":"Omi Bridge Test","category":"personal"},"folder_id":"folder-001","folder_name":"Personal"}`)
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/omi?uid=omi-user-001", bytes.NewReader(payload))
	req.Header.Set("X-Pantheon-Intake-Secret", "kernel26-omi-secret")
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusCreated {
		t.Fatalf("native Omi webhook status = %d, body = %s", response.Code, response.Body.String())
	}
	var receipt externalIntakeReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(d.cfg.IntakePath, filepath.FromSlash(receipt.RawPayloadPath)))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(raw, payload) {
		t.Fatalf("native Omi payload changed: got %s", raw)
	}
	envelopeArchive, err := os.ReadFile(filepath.Join(d.cfg.IntakePath, filepath.FromSlash(receipt.EnvelopePath)))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(envelopeArchive), `"omi_uid":"omi-user-001"`) {
		t.Fatalf("Omi uid was not preserved in envelope: %s", envelopeArchive)
	}
	transcript, err := os.ReadFile(receipt.NormalizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(transcript), "[00:00:10.250] Trinh: Let's keep the raw Omi memory.") || !strings.Contains(string(transcript), "[00:00:15.000] SPEAKER_01: Agreed.") {
		t.Fatalf("native Omi transcript was not converted correctly: %s", transcript)
	}
}

func TestKernel26NativeOmiWebhookAcceptsQuerySecret(t *testing.T) {
	t.Setenv(externalIntakeSecretEnv, "kernel26-query-secret")
	d := kernel15Daemon(t, janusTestPackage("KERNEL26:OMIQUERY", "expire"))
	defer d.Close()
	payload := `{"id":"memory-omi-query","created_at":"2026-08-23T10:15:00Z","transcript_segments":[]}`
	req := httptest.NewRequest(http.MethodPost, "/v1/intake/omi?pantheon_intake_secret=kernel26-query-secret", strings.NewReader(payload))
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusAccepted {
		t.Fatalf("query-secret Omi webhook status = %d, body = %s", response.Code, response.Body.String())
	}
}
