package pantheon

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestKernel21PanopticonServesEmbeddedLocalSurface(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL21SURFACE", "expire"))
	defer d.Close()

	page := httptest.NewRecorder()
	d.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "Panopticon") {
		t.Fatalf("Panopticon page status=%d body=%q", page.Code, page.Body.String())
	}
	if page.Header().Get("Content-Security-Policy") == "" || page.Header().Get("X-Frame-Options") != "DENY" {
		t.Fatalf("embedded page security headers=%v", page.Header())
	}

	asset := httptest.NewRecorder()
	d.Handler().ServeHTTP(asset, httptest.NewRequest(http.MethodGet, "/assets/app.js", nil))
	if asset.Code != http.StatusOK || !strings.Contains(asset.Body.String(), "/v1/gui/prometheus/") {
		t.Fatalf("embedded app asset status=%d body prefix=%q", asset.Code, asset.Body.String()[:minInt(80, asset.Body.Len())])
	}
	if strings.Contains(asset.Body.String(), `json("/v1/status")`) {
		t.Fatal("Panopticon must use the read-only inventory contract rather than polling the auditing status route")
	}
	missing := httptest.NewRecorder()
	d.Handler().ServeHTTP(missing, httptest.NewRequest(http.MethodGet, "/assets/not-present.js", nil))
	if missing.Code != http.StatusNotFound {
		t.Fatalf("missing asset status=%d", missing.Code)
	}
}

func TestKernel21PanopticonSessionControlsPrometheusWithoutReplay(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL21PROMETHEUS", "expire"))
	defer d.Close()

	session := httptest.NewRecorder()
	d.Handler().ServeHTTP(session, httptest.NewRequest(http.MethodPost, "/v1/gui/session", nil))
	if session.Code != http.StatusOK || len(session.Result().Cookies()) != 1 {
		t.Fatalf("GUI session status=%d cookies=%v body=%s", session.Code, session.Result().Cookies(), session.Body.String())
	}
	cookie := session.Result().Cookies()[0]
	control := func(path string) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, nil)
		request.AddCookie(cookie)
		response := httptest.NewRecorder()
		d.Handler().ServeHTTP(response, request)
		return response
	}
	paused := control("/v1/gui/prometheus/pause")
	if paused.Code != http.StatusOK || !strings.Contains(paused.Body.String(), "will not replay") {
		t.Fatalf("pause status=%d body=%s", paused.Code, paused.Body.String())
	}
	if active, err := d.paused(); err != nil || !active {
		t.Fatalf("paused=%v err=%v", active, err)
	}
	resumed := control("/v1/gui/prometheus/resume")
	if resumed.Code != http.StatusOK || !strings.Contains(resumed.Body.String(), "remains skipped") {
		t.Fatalf("resume status=%d body=%s", resumed.Code, resumed.Body.String())
	}
	if active, err := d.paused(); err != nil || active {
		t.Fatalf("paused after resume=%v err=%v", active, err)
	}
}

func TestKernel21PanopticonConfigurationValidateApplyAndProtectsSecrets(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL21CONFIG", "expire"))
	defer d.Close()

	canonical, err := marshalCanonicalConfig(d.cfg)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(d.configPath, canonical, 0o600); err != nil {
		t.Fatal(err)
	}
	original, err := os.ReadFile(d.configPath)
	if err != nil {
		t.Fatal(err)
	}
	session := httptest.NewRecorder()
	d.Handler().ServeHTTP(session, httptest.NewRequest(http.MethodPost, "/v1/gui/session", nil))
	if session.Code != http.StatusOK {
		t.Fatalf("session status=%d", session.Code)
	}
	cookie := session.Result().Cookies()[0]
	changes := map[string]json.RawMessage{"rumor.poll_seconds": json.RawMessage("7")}
	requestBody, err := json.Marshal(guiConfigurationRequest{Changes: changes})
	if err != nil {
		t.Fatal(err)
	}
	call := func(path string, withCookie bool) *httptest.ResponseRecorder {
		request := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(requestBody))
		request.Header.Set("Content-Type", "application/json")
		if withCookie {
			request.AddCookie(cookie)
		}
		response := httptest.NewRecorder()
		d.Handler().ServeHTTP(response, request)
		return response
	}
	validated := call("/v1/gui/configuration/validate", false)
	if validated.Code != http.StatusOK || !strings.Contains(validated.Body.String(), "has not been written") {
		t.Fatalf("validate status=%d body=%s", validated.Code, validated.Body.String())
	}
	unchanged, err := os.ReadFile(d.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(original, unchanged) {
		t.Fatal("configuration validation wrote the file")
	}
	applied := call("/v1/gui/configuration/apply", true)
	if applied.Code != http.StatusOK || !strings.Contains(applied.Body.String(), `"applied":true`) {
		t.Fatalf("apply status=%d body=%s", applied.Code, applied.Body.String())
	}
	updated, err := os.ReadFile(d.configPath)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Equal(original, updated) || !bytes.Contains(updated, []byte(`"poll_seconds": 7`)) {
		t.Fatalf("configuration was not updated: %s", updated)
	}
	if strings.Contains(string(updated), "control_token") || strings.Contains(applied.Body.String(), d.controlToken) {
		t.Fatal("GUI configuration exposed a secret")
	}
}

func TestKernel21PanopticonRequiresLocalOriginForSession(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL21ORIGIN", "expire"))
	defer d.Close()
	request := httptest.NewRequest(http.MethodPost, "/v1/gui/session", nil)
	request.Host = "127.0.0.1:8765"
	request.Header.Set("Origin", "https://127.0.0.1:8765")
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusForbidden {
		t.Fatalf("non-local origin status=%d body=%s", response.Code, response.Body.String())
	}
}

func minInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}
