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

func kernel14Manifest(t *testing.T, dir, name, subject, adapter string) string {
	t.Helper()
	manifest := seshatManifest{
		SchemaVersion: 1,
		Plugin:        seshatPlugin{Name: name, Version: "1.0.0"},
		Packages: []seshatPackageSpec{{
			SubjectCode: subject, Version: "1.0.0", Description: "test package",
			Capability: "test.action", Adapter: adapter,
			RequiredFields: []string{"target", "body"}, OptionalFields: []string{"note"},
			PayloadRules: map[string]string{"target": "TARGET", "body": "body", "note": "NOTE"},
			Returns:      []string{"result_id", "status"}, PermissionClass: "bounded_test_action",
			PermissionShapes: []string{"once", "bounded_uses"},
		}},
	}
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, name+".json")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestKernel14ManifestValidationRejectsUnsupportedAndSelfAuthority(t *testing.T) {
	valid := `{"schema_version":1,"plugin":{"name":"example","version":"1"},"packages":[{"subject_code":"EXAMPLE:ACTION","version":"1","description":"test","capability":"example.action","adapter":"noop","required_fields":["body"],"optional_fields":[],"payload_rules":{"body":"body"},"returns":["status"],"permission_class":"bounded","permission_shapes":["once"]}]}`
	if _, err := validateSeshatManifestBytes([]byte(strings.Replace(valid, `"schema_version":1`, `"schema_version":99`, 1))); err == nil || !strings.Contains(err.Error(), "unsupported manifest schema") {
		t.Fatalf("unsupported schema error = %v", err)
	}
	if _, err := validateSeshatManifestBytes([]byte(strings.Replace(valid, `"subject_code"`, `"authority":"person","subject_code"`, 1))); err == nil || !strings.Contains(err.Error(), "cannot declare authority") {
		t.Fatalf("self-authority error = %v", err)
	}
	if _, err := validateSeshatManifestBytes([]byte(strings.Replace(valid, `"permission_shapes":["once"]`, `"permission_shapes":["arbitrary"]`, 1))); err == nil || !strings.Contains(err.Error(), "permission shape") {
		t.Fatalf("permission shape error = %v", err)
	}
	malformedMapping := strings.Replace(valid, `"payload_rules":{"body":"body"}`, `"payload_rules":{"missing":"body"}`, 1)
	if _, err := validateSeshatManifestBytes([]byte(malformedMapping)); err == nil || !strings.Contains(err.Error(), "undeclared field") {
		t.Fatalf("mapping error = %v", err)
	}
	malformedField := strings.Replace(valid, `"required_fields":["body"]`, `"required_fields":["Body"]`, 1)
	if _, err := validateSeshatManifestBytes([]byte(malformedField)); err == nil || !strings.Contains(err.Error(), "lowercase identifier") {
		t.Fatalf("field error = %v", err)
	}
}

func TestKernel14ExtensionRegistrationConstructionAndPersistence(t *testing.T) {
	dir := t.TempDir()
	registryDir := filepath.Join(dir, "registered")
	registry, err := newSeshatRegistry(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	manifestPath := kernel14Manifest(t, dir, "test-extension", "TEST:ACTION", "noop")
	if _, err := registry.validatePath(manifestPath); err != nil {
		t.Fatal(err)
	}
	manifest, err := registry.registerPath(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	if manifest.Plugin.Name != "test-extension" {
		t.Fatalf("registered plugin = %q", manifest.Plugin.Name)
	}
	pkg, err := registry.packageFor("TEST:ACTION")
	if err != nil {
		t.Fatal(err)
	}
	if pkg.Capability != "test.action" || pkg.PluginName != "test-extension" || !pkg.AdapterAvailable {
		t.Fatalf("registered package = %#v", pkg)
	}
	index := seshatApolloIndex(registry.packagesInspection())
	indexJSON, _ := json.Marshal(index)
	if !strings.Contains(string(indexJSON), "TEST:ACTION") || strings.Contains(string(indexJSON), "noop") {
		t.Fatalf("Apollo-safe index leaked adapter details: %s", indexJSON)
	}
	normalized, err := constructSeshatPackage(pkg, "TYPE: TEST:ACTION\nTARGET: Person#17\nNOTE: keep context\n\nThe substantive body stays exactly here.\n")
	if err != nil {
		t.Fatal(err)
	}
	if normalized.Fields["target"] != "Person#17" || normalized.Fields["body"] != "The substantive body stays exactly here.\n" || normalized.Fields["note"] != "keep context" {
		t.Fatalf("normalized package fields = %#v", normalized.Fields)
	}
	if normalized.Capability != "test.action" || normalized.Adapter != "noop" || len(normalized.Returns) != 2 || normalized.PermissionClass != "bounded_test_action" {
		t.Fatalf("normalized package contract = %#v", normalized)
	}
	if normalized.SourceMissiveHash == "" {
		t.Fatal("normalized package omitted source missive identity")
	}
	if _, err := constructSeshatPackage(pkg, "TYPE: TEST:ACTION\n\nbody"); err == nil || !strings.Contains(err.Error(), "missing required field") {
		t.Fatalf("missing required field error = %v", err)
	}
	if _, err := registry.packageFor("UNKNOWN:ACTION"); err == nil {
		t.Fatal("unknown subject code was accepted")
	}
	reopened, err := newSeshatRegistry(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := reopened.packageFor("TEST:ACTION"); err != nil {
		t.Fatalf("registered package did not persist: %v", err)
	}
}

func TestKernel14PackageSafetyAndInspectionEndpoints(t *testing.T) {
	dir := t.TempDir()
	registryDir := filepath.Join(dir, "registered")
	registry, err := newSeshatRegistry(registryDir)
	if err != nil {
		t.Fatal(err)
	}
	path := kernel14Manifest(t, dir, "endpoint-extension", "ENDPOINT:ACTION", "noop")
	if _, err := registry.registerPath(path); err != nil {
		t.Fatal(err)
	}
	d := &daemon{seshat: registry, controlToken: "control"}
	handler := d.Handler()
	for _, test := range []struct {
		name string
		path string
	}{
		{"plugins", "/v1/seshat/plugins"},
		{"packages", "/v1/seshat/packages"},
		{"index", "/v1/seshat/index"},
		{"help", "/v1/seshat/help"},
		{"schema", "/v1/seshat/schema"},
	} {
		t.Run(test.name, func(t *testing.T) {
			rr := httptest.NewRecorder()
			handler.ServeHTTP(rr, httptest.NewRequest(http.MethodGet, test.path, nil))
			if rr.Code != http.StatusOK {
				t.Fatalf("%s status = %d, body=%s", test.path, rr.Code, rr.Body.String())
			}
		})
	}
	request := newJSONRequest(t, http.MethodPost, "/v1/seshat/package", seshatPackageRequest{Missive: "TYPE: ENDPOINT:ACTION\nTARGET: local\n\nbody"})
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, request)
	if rr.Code != http.StatusOK {
		t.Fatalf("package response status=%d body=%s", rr.Code, rr.Body.String())
	}
	if strings.Contains(rr.Body.String(), `"authority"`) {
		t.Fatalf("package construction should not invent authority: %s", rr.Body.String())
	}
	unknown := httptest.NewRecorder()
	handler.ServeHTTP(unknown, newJSONRequest(t, http.MethodPost, "/v1/seshat/package", seshatPackageRequest{Missive: "TYPE: UNKNOWN:ACTION\nTARGET: local\n\nbody"}))
	if unknown.Code != http.StatusNotFound {
		t.Fatalf("unknown package status = %d", unknown.Code)
	}
	unauthorized := httptest.NewRecorder()
	handler.ServeHTTP(unauthorized, newJSONRequest(t, http.MethodPost, "/v1/seshat/plugins/register", seshatPathRequest{Path: path}))
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("unauthorized registration status = %d", unauthorized.Code)
	}
}

func TestKernel14DuplicateDisabledAndUnavailablePackagesFailBoringly(t *testing.T) {
	dir := t.TempDir()
	registry, err := newSeshatRegistry(filepath.Join(dir, "registered"))
	if err != nil {
		t.Fatal(err)
	}
	first := kernel14Manifest(t, dir, "first", "DUPLICATE:ACTION", "noop")
	if _, err := registry.registerPath(first); err != nil {
		t.Fatal(err)
	}
	second := kernel14Manifest(t, dir, "second", "DUPLICATE:ACTION", "noop")
	if _, err := registry.registerPath(second); err == nil || !strings.Contains(err.Error(), "duplicate") {
		t.Fatalf("duplicate registration error = %v", err)
	}
	disabled := kernel14Manifest(t, dir, "disabled", "DISABLED:ACTION", "noop")
	data, err := os.ReadFile(disabled)
	if err != nil {
		t.Fatal(err)
	}
	var disabledManifest seshatManifest
	if err := json.Unmarshal(data, &disabledManifest); err != nil {
		t.Fatal(err)
	}
	falseValue := false
	disabledManifest.Packages[0].Enabled = &falseValue
	data, err = json.Marshal(disabledManifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(disabled, data, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.registerPath(disabled); err != nil {
		t.Fatal(err)
	}
	if _, err := registry.packageFor("DISABLED:ACTION"); err == nil || !strings.Contains(err.Error(), "disabled") {
		t.Fatalf("disabled package error = %v", err)
	}
	unavailable := kernel14Manifest(t, dir, "unavailable", "UNAVAILABLE:ACTION", "missing-adapter")
	if _, err := registry.registerPath(unavailable); err != nil {
		t.Fatal(err)
	}
	pkg, err := registry.packageFor("UNAVAILABLE:ACTION")
	if err != nil {
		t.Fatal(err)
	}
	normalized, err := constructSeshatPackage(pkg, "TYPE: UNAVAILABLE:ACTION\nTARGET: local\n\nbody")
	if err != nil || normalized.AdapterAvailable {
		t.Fatalf("unavailable adapter construction = %#v, err=%v", normalized, err)
	}
}
