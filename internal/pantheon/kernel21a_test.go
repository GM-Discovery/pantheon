package pantheon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestKernel21APanopticonAcceptancePresentationContracts(t *testing.T) {
	page, err := panopticonAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	if !strings.Contains(html, `current-work-panel`) || !strings.Contains(html, `class="work-board"`) {
		t.Fatal("Home does not make Current Work a dedicated primary surface")
	}
	if strings.Index(html, `current-work-panel`) > strings.Index(html, `class="cell-grid"`) {
		t.Fatal("Current Work appears below the generic component inventory")
	}
	for _, required := range []string{`fates-surface`, `class="fates-group"`, `>Present<`, `>Past<`, `>Future<`, `class="weave-tool"`, `TOOL USED BY THE FATES`} {
		if !strings.Contains(html, required) {
			t.Fatalf("Fates presentation omitted %q", required)
		}
	}
	if strings.Count(html, `class="fate-card"`) != 3 {
		t.Fatalf("expected exactly three grouped Fate cards, found %d", strings.Count(html, `class="fate-card"`))
	}
}

func TestKernel21ANullCapabilityResponseHasBoundedFallbackContract(t *testing.T) {
	asset, err := panopticonAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	source := string(asset)
	for _, required := range []string{
		"Array.isArray(body)",
		"body && Array.isArray(body.capabilities)",
		"Capability information unavailable",
		"console.warn(\"[Panopticon] Capability information unavailable\"",
		"id=\"retry-capabilities\"",
	} {
		if !strings.Contains(source, required) {
			t.Fatalf("capability fallback contract omitted %q", required)
		}
	}
	if strings.Contains(source, "body.capabilities || body || []") {
		t.Fatal("capability rendering still dereferences a nullable response")
	}
}

func TestKernel21AWorkContextExplainsBlockedRecords(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL21AWORK", "expire"))
	defer d.Close()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`INSERT INTO orders (order_id, created_at, origin, request_text, status, last_error) VALUES (?, ?, ?, ?, ?, ?)`, "blocked-kernel21a", now, "test", "A blocked test order", orderStatusUnavailable, "no safe local adapter"); err != nil {
		t.Fatal(err)
	}
	request := httptest.NewRequest(http.MethodGet, "/v1/gui/work?category=blocked", nil)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, request)
	if response.Code != http.StatusOK {
		t.Fatalf("work context status=%d body=%s", response.Code, response.Body.String())
	}
	var work guiWorkResponse
	if err := json.Unmarshal(response.Body.Bytes(), &work); err != nil {
		t.Fatal(err)
	}
	if work.Counts["blocked"] != 1 || len(work.Items) != 1 {
		t.Fatalf("blocked work context=%#v", work)
	}
	item := work.Items[0]
	if item.ID != "blocked-kernel21a" || item.NextAction == "" || !strings.Contains(item.Explanation, "adapter") {
		t.Fatalf("blocked work item lacks explanation/action=%#v", item)
	}
}

func TestKernel21AWorkContextIsReachableFromHome(t *testing.T) {
	page, err := panopticonAssets.ReadFile("web/index.html")
	if err != nil {
		t.Fatal(err)
	}
	html := string(page)
	for _, required := range []string{"How to act on this work", "work-context", "data-work-category", "execution-filter", "Blocked work", "Needs decision"} {
		if !strings.Contains(html, required) {
			t.Fatalf("Home omitted work context affordance %q", required)
		}
	}
	asset, err := panopticonAssets.ReadFile("web/app.js")
	if err != nil {
		t.Fatal(err)
	}
	source := string(asset)
	for _, required := range []string{"/v1/gui/work?category=", "View records and next step", "Next safe step:", "No safe execution action is available"} {
		if !strings.Contains(source, required) {
			t.Fatalf("work context omitted %q", required)
		}
	}
}
