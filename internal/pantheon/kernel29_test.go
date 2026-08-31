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

func TestKernel29ZeusDryRunClassifiesOnlyExplicitCaches(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL29", "expire"))
	defer d.Close()
	cacheRoot := filepath.Join(t.TempDir(), "pantheon-go-cache")
	unknownRoot := filepath.Join(t.TempDir(), "old-work")
	if err := os.MkdirAll(filepath.Join(cacheRoot, "nested"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(unknownRoot, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "a.o"), []byte("recreatable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cacheRoot, "nested", "b.o"), []byte("also recreatable"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(unknownRoot, "meeting.wav"), []byte("personal source"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(cacheRoot, "a.o"), filepath.Join(cacheRoot, "linked.o")); err != nil {
		t.Fatal(err)
	}
	d.cfg.ZeusEnabled = true
	d.cfg.ZeusRoots = []string{cacheRoot, unknownRoot}
	d.cfg.ZeusQuarantinePath = filepath.Join(t.TempDir(), "quarantine")

	report, err := d.runZeus(zeusModeDryRun)
	if err != nil {
		t.Fatal(err)
	}
	if report.LatestRun == nil || report.LatestRun.CandidateCount != 4 || report.LatestRun.EligibleCount != 2 || report.LatestRun.ProtectedCount != 2 {
		t.Fatalf("unexpected run: %+v", report.LatestRun)
	}
	states := map[string]string{}
	for _, candidate := range report.Candidates {
		states[candidate.RelativePath] = candidate.State
	}
	if states["a.o"] != zeusStateEligible || states["nested/b.o"] != zeusStateEligible {
		t.Fatalf("cache candidates=%v", states)
	}
	if states["meeting.wav"] != zeusStateProtected || states["linked.o"] != zeusStateProtected {
		t.Fatalf("protected candidates=%v", states)
	}
	if _, err := os.Stat(filepath.Join(unknownRoot, "meeting.wav")); err != nil {
		t.Fatalf("dry-run changed protected source: %v", err)
	}
	if _, err := os.Stat(filepath.Join(cacheRoot, "a.o")); err != nil {
		t.Fatalf("dry-run changed cache: %v", err)
	}
	var stored int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM zeus_candidates WHERE run_id = ?`, report.LatestRun.RunID).Scan(&stored); err != nil {
		t.Fatal(err)
	}
	if stored != 4 {
		t.Fatalf("durable candidate count=%d", stored)
	}
}

func TestKernel29ZeusRouteIsInspectableAndNoDelete(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL29ROUTE", "expire"))
	defer d.Close()
	root := filepath.Join(t.TempDir(), "pantheon-go-cache")
	if err := os.MkdirAll(root, 0o700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "cache.o")
	if err := os.WriteFile(path, []byte("cache"), 0o600); err != nil {
		t.Fatal(err)
	}
	d.cfg.ZeusEnabled = true
	d.cfg.ZeusRoots = []string{root}

	req := httptest.NewRequest(http.MethodPost, "/v1/zeus/dry-run", strings.NewReader(`{"mode":"dry_run"}`))
	req.Header.Set("Content-Type", "application/json")
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("Zeus dry-run status=%d body=%s", response.Code, response.Body.String())
	}
	var document zeusResponse
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.LatestRun == nil || document.LatestRun.Mode != zeusModeDryRun || document.AuditID == "" {
		t.Fatalf("response=%+v", document)
	}
	if _, err := os.Stat(path); err != nil {
		t.Fatalf("dry-run removed candidate: %v", err)
	}
	var audits int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE endpoint = 'POST /v1/zeus/inventory'`).Scan(&audits); err != nil {
		t.Fatal(err)
	}
	if audits != 1 {
		t.Fatalf("Zeus audit count=%d", audits)
	}
}

func TestKernel29SystemInventoryExposesZeusReadyState(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL29SYSTEM", "expire"))
	defer d.Close()
	d.cfg.ZeusEnabled = true
	d.cfg.ZeusRoots = []string{filepath.Join(t.TempDir(), "cache")}
	inventory, err := d.systemInventory()
	if err != nil {
		t.Fatal(err)
	}
	for _, component := range inventory.Components {
		if component.ID == "zeus" {
			if component.State != "ready" || component.InspectionRoute != "/v1/zeus" {
				t.Fatalf("Zeus component=%+v", component)
			}
			return
		}
	}
	t.Fatal("system inventory omitted Zeus")
}
