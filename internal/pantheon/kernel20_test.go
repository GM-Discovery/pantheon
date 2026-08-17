package pantheon

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestKernel20SystemInventoryIsCanonicalAndSecretFree(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL20", "expire"))
	defer d.Close()
	d.controlToken = "kernel20-control-secret"
	d.cfg.LocalModel = "local-test-model"
	d.cfg.OllamaURL = "http://127.0.0.1:11434/api/chat"

	inventory, err := d.systemInventory()
	if err != nil {
		t.Fatal(err)
	}
	if inventory.ContractVersion != systemInventoryContractVersion {
		t.Fatalf("contract version=%d", inventory.ContractVersion)
	}
	if len(inventory.Components) < 10 {
		t.Fatalf("component inventory is incomplete: %d", len(inventory.Components))
	}
	seen := map[string]bool{}
	for _, component := range inventory.Components {
		seen[component.ID] = true
	}
	for _, required := range []string{"intake", "present", "apollo", "future", "mercury", "prometheus", "rumor", "bifrost"} {
		if !seen[required] {
			t.Fatalf("component inventory omitted %q", required)
		}
	}
	encoded, err := json.Marshal(inventory)
	if err != nil {
		t.Fatal(err)
	}
	output := string(encoded)
	for _, secret := range []string{"kernel20-control-secret", "password", "refresh_token", "Authorization"} {
		if strings.Contains(output, secret) {
			t.Fatalf("inventory exposed %q: %s", secret, output)
		}
	}
	if inventory.Security["control_token"] != "omitted" || inventory.Security["secrets"] != "omitted" {
		t.Fatalf("inventory security markers=%#v", inventory.Security)
	}
}

func TestKernel20SystemInventoryRouteIsReadOnly(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL20ROUTE", "expire"))
	defer d.Close()
	req := httptest.NewRequest(http.MethodGet, "/v1/system/inventory", nil)
	response := httptest.NewRecorder()
	d.Handler().ServeHTTP(response, req)
	if response.Code != http.StatusOK {
		t.Fatalf("inventory route status=%d body=%s", response.Code, response.Body.String())
	}
	var document systemInventoryResponse
	if err := json.Unmarshal(response.Body.Bytes(), &document); err != nil {
		t.Fatal(err)
	}
	if document.ContractVersion != systemInventoryContractVersion {
		t.Fatalf("route contract version=%d", document.ContractVersion)
	}
	var auditEvents int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM audit_events WHERE endpoint = 'GET /v1/system/inventory'`).Scan(&auditEvents); err != nil {
		t.Fatal(err)
	}
	if auditEvents != 0 {
		t.Fatalf("read-only inventory unexpectedly created an audit event: %d", auditEvents)
	}
}
