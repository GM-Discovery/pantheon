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

func TestKernel27ReportAssemblyCreatesCitedManifestAndIsIdempotent(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("KERNEL27:ASSEMBLY", "expire"))
	defer d.Close()
	vault := filepath.Join(t.TempDir(), "vault")
	d.cfg.VaultPath = vault
	d.cfg.RawPath = filepath.Join(vault, "80 Pantheon Raw")
	transcript := filepath.Join(vault, "80 Pantheon Raw", "Audio Transcripts", "2026", "08", "event.md")
	eventReport := filepath.Join(vault, "70 Pantheon Reports", "Integrated Events", "2026-08-23-existing.md")
	for _, path := range []string{transcript, eventReport} {
		if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte("# Evidence\n\nA named request appears here.\n"), 0o600); err != nil {
			t.Fatal(err)
		}
	}

	requestBody, err := json.Marshal(reportAssemblyRequest{Kind: reportAssemblyDaily, PeriodDate: "2026-08-23", SourceRefs: []string{
		"80 Pantheon Raw/Audio Transcripts/2026/08/event.md",
		"70 Pantheon Reports/Integrated Events/2026-08-23-existing.md",
	}})
	if err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodPost, "/v1/reports/assemble", bytes.NewReader(requestBody))
	request.Header.Set(controlTokenHeader, d.controlToken)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusCreated {
		t.Fatalf("assembly status = %d, body = %s", response.Code, response.Body.String())
	}
	var receipt reportAssemblyReceipt
	if err := json.Unmarshal(response.Body.Bytes(), &receipt); err != nil {
		t.Fatal(err)
	}
	if receipt.Status != reportAssemblyReady || receipt.SourceCount != 2 || receipt.ManifestPath == "" || receipt.BodySHA256 == "" {
		t.Fatalf("unexpected assembly receipt: %+v", receipt)
	}
	manifest, err := os.ReadFile(receipt.ManifestPath)
	if err != nil {
		t.Fatal(err)
	}
	manifestText := string(manifest)
	if !strings.Contains(manifestText, "[[80 Pantheon Raw/Audio Transcripts/2026/08/event.md]]") || !strings.Contains(manifestText, "[[70 Pantheon Reports/Integrated Events/2026-08-23-existing.md]]") {
		t.Fatalf("manifest omitted exact citations: %s", manifestText)
	}
	if !strings.Contains(manifestText, "does not infer importance") || !strings.Contains(manifestText, "execute calendar") {
		t.Fatalf("manifest did not state the execution boundary: %s", manifestText)
	}

	duplicateRequest := httptest.NewRequest(http.MethodPost, "/v1/reports/assemble", bytes.NewReader(requestBody))
	duplicateRequest.Header.Set(controlTokenHeader, d.controlToken)
	duplicateResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(duplicateResponse, duplicateRequest)
	if duplicateResponse.Code != http.StatusOK || !strings.Contains(duplicateResponse.Body.String(), `"duplicate":true`) {
		t.Fatalf("duplicate assembly response = %d %s", duplicateResponse.Code, duplicateResponse.Body.String())
	}

	showRequest := httptest.NewRequest(http.MethodGet, "/v1/reports/assemblies/"+receipt.AssemblyID, nil)
	showResponse := httptest.NewRecorder()
	d.Handler().ServeHTTP(showResponse, showRequest)
	if showResponse.Code != http.StatusOK || !strings.Contains(showResponse.Body.String(), receipt.AssemblyID) {
		t.Fatalf("assembly inspection = %d %s", showResponse.Code, showResponse.Body.String())
	}
}

func TestKernel27EventAssemblyRejectsVaultEscape(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("KERNEL27:BOUNDARY", "expire"))
	defer d.Close()
	d.cfg.VaultPath = filepath.Join(t.TempDir(), "vault")
	request := reportAssemblyRequest{Kind: reportAssemblyEvent, PeriodDate: "2026-08-23", SourceRefs: []string{"../outside.md"}}
	if _, err := d.assembleReport(request); err == nil || !strings.Contains(err.Error(), "escapes configured Vault") {
		t.Fatalf("Vault escape was not rejected: %v", err)
	}
}
