package pantheon

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
)

const seshatManifestSchemaVersion = 1

var (
	seshatSubjectCodePattern = regexp.MustCompile(`^[A-Z][A-Z0-9]*(:[A-Z0-9_]+)*$`)
	seshatFieldPattern       = regexp.MustCompile(`^[a-z][a-z0-9_]*$`)
	seshatHeaderPattern      = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_-]*$`)
	seshatPluginNamePattern  = regexp.MustCompile(`^[a-z][a-z0-9_-]*$`)
	seshatVersionPattern     = regexp.MustCompile(`^[A-Za-z0-9][A-Za-z0-9._+-]*$`)
	seshatFixedPermission    = map[string]bool{
		"once": true, "project": true, "this_project": true, "bounded_uses": true,
		"until_revoked": true, "temporary": true, "until_timestamp": true,
	}
)

// A manifest describes a package contract; it never describes an authority grant.
type seshatManifest struct {
	SchemaVersion int                 `json:"schema_version"`
	Plugin        seshatPlugin        `json:"plugin"`
	Packages      []seshatPackageSpec `json:"packages"`
}

type seshatPlugin struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

type seshatPackageSpec struct {
	SubjectCode      string            `json:"subject_code"`
	Version          string            `json:"version"`
	Description      string            `json:"description"`
	Capability       string            `json:"capability"`
	Adapter          string            `json:"adapter"`
	RequiredFields   []string          `json:"required_fields"`
	OptionalFields   []string          `json:"optional_fields"`
	PayloadRules     map[string]string `json:"payload_rules,omitempty"`
	Returns          []string          `json:"returns"`
	PermissionClass  string            `json:"permission_class"`
	PermissionShapes []string          `json:"permission_shapes"`
	Aliases          []string          `json:"aliases,omitempty"`
	Constraints      map[string]string `json:"constraints,omitempty"`
	Examples         []string          `json:"examples,omitempty"`
	Enabled          *bool             `json:"enabled,omitempty"`
	Temporal         *seshatTemporal   `json:"temporal,omitempty"`
}

type seshatTemporal struct {
	Mode         string `json:"mode"`
	MissedPolicy string `json:"missed_policy"`
	PausePolicy  string `json:"pause_policy,omitempty"`
}

var seshatTemporalPolicies = map[string]bool{
	"expire": true, "resurface": true, "backfill": true, "recompute": true, "require_review": true,
}

type seshatPackage struct {
	seshatPackageSpec
	PluginName       string `json:"plugin_name"`
	PluginVersion    string `json:"plugin_version"`
	ManifestPath     string `json:"manifest_path"`
	AdapterAvailable bool   `json:"adapter_available"`
}

type seshatPluginInspection struct {
	Name            string `json:"name"`
	Version         string `json:"version"`
	ManifestPath    string `json:"manifest_path"`
	PackageCount    int    `json:"package_count"`
	EnabledPackages int    `json:"enabled_packages"`
}

type seshatRegistry struct {
	dir      string
	packages map[string]seshatPackage
	plugins  map[string]seshatPluginInspection
}

// The registry knows only adapter identities. It never executes an adapter.
// local-notify is the existing harmless local notification boundary; noop is a
// deterministic proof adapter used by extension manifests and tests.
var seshatAvailableAdapters = map[string]bool{
	"local-notify": true,
	"noop":         true,
}

func newSeshatRegistry(dir string) (*seshatRegistry, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	r := &seshatRegistry{dir: dir, packages: map[string]seshatPackage{}, plugins: map[string]seshatPluginInspection{}}
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	for _, entry := range entries {
		if entry.IsDir() || strings.ToLower(filepath.Ext(entry.Name())) != ".json" {
			continue
		}
		manifestPath := filepath.Join(dir, entry.Name())
		manifest, err := loadSeshatManifest(manifestPath)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", manifestPath, err)
		}
		if err := r.addManifest(manifest, manifestPath); err != nil {
			return nil, fmt.Errorf("%s: %w", manifestPath, err)
		}
	}
	return r, nil
}

func loadSeshatManifest(path string) (seshatManifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return seshatManifest{}, err
	}
	return validateSeshatManifestBytes(data)
}

func validateSeshatManifestBytes(data []byte) (seshatManifest, error) {
	var raw map[string]json.RawMessage
	if err := json.Unmarshal(data, &raw); err != nil {
		return seshatManifest{}, fmt.Errorf("manifest is malformed JSON: %w", err)
	}
	if forbiddenSeshatKey(raw) {
		return seshatManifest{}, errors.New("manifest cannot declare authority, grants, holders, or approvals")
	}
	var manifest seshatManifest
	decoder := json.NewDecoder(strings.NewReader(string(data)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&manifest); err != nil {
		return seshatManifest{}, fmt.Errorf("manifest has unsupported fields or types: %w", err)
	}
	if manifest.SchemaVersion != seshatManifestSchemaVersion {
		return seshatManifest{}, fmt.Errorf("unsupported manifest schema_version %d (supported: %d)", manifest.SchemaVersion, seshatManifestSchemaVersion)
	}
	if !seshatPluginNamePattern.MatchString(strings.TrimSpace(manifest.Plugin.Name)) {
		return seshatManifest{}, errors.New("plugin.name must be a lowercase identifier")
	}
	if !seshatVersionPattern.MatchString(strings.TrimSpace(manifest.Plugin.Version)) {
		return seshatManifest{}, errors.New("plugin.version must be a non-empty version identifier")
	}
	if len(manifest.Packages) == 0 {
		return seshatManifest{}, errors.New("manifest must declare at least one package")
	}
	seen := map[string]bool{}
	for i := range manifest.Packages {
		if err := validateSeshatPackage(manifest.Packages[i]); err != nil {
			return seshatManifest{}, fmt.Errorf("packages[%d]: %w", i, err)
		}
		code := strings.ToUpper(strings.TrimSpace(manifest.Packages[i].SubjectCode))
		if seen[code] {
			return seshatManifest{}, fmt.Errorf("duplicate subject_code %q in manifest", code)
		}
		seen[code] = true
		for _, alias := range manifest.Packages[i].Aliases {
			alias = strings.ToUpper(strings.TrimSpace(alias))
			if seen[alias] {
				return seshatManifest{}, fmt.Errorf("duplicate subject code or alias %q in manifest", alias)
			}
			seen[alias] = true
		}
	}
	return manifest, nil
}

func forbiddenSeshatKey(value any) bool {
	for key, child := range value.(map[string]json.RawMessage) {
		switch strings.ToLower(key) {
		case "authority", "authorized", "grant", "grants", "holder", "approval", "approvals":
			return true
		}
		var object map[string]json.RawMessage
		if json.Unmarshal(child, &object) == nil && forbiddenSeshatKey(object) {
			return true
		}
		var list []json.RawMessage
		if json.Unmarshal(child, &list) == nil {
			for _, item := range list {
				var nested map[string]json.RawMessage
				if json.Unmarshal(item, &nested) == nil && forbiddenSeshatKey(nested) {
					return true
				}
			}
		}
	}
	return false
}

func validateSeshatPackage(spec seshatPackageSpec) error {
	spec.SubjectCode = strings.TrimSpace(spec.SubjectCode)
	if !seshatSubjectCodePattern.MatchString(spec.SubjectCode) {
		return errors.New("subject_code must be an uppercase colon-delimited code")
	}
	if !seshatVersionPattern.MatchString(strings.TrimSpace(spec.Version)) {
		return errors.New("version must be a non-empty version identifier")
	}
	if strings.TrimSpace(spec.Description) == "" || strings.TrimSpace(spec.Capability) == "" || strings.TrimSpace(spec.Adapter) == "" {
		return errors.New("description, capability, and adapter are required")
	}
	if len(spec.RequiredFields) == 0 {
		return errors.New("required_fields must contain at least one field")
	}
	fields := map[string]bool{}
	for _, field := range append(append([]string{}, spec.RequiredFields...), spec.OptionalFields...) {
		if !seshatFieldPattern.MatchString(field) {
			return fmt.Errorf("field %q is not a lowercase identifier", field)
		}
		if fields[field] {
			return fmt.Errorf("field %q is duplicated or both required and optional", field)
		}
		fields[field] = true
	}
	for field, source := range spec.PayloadRules {
		if !fields[field] {
			return fmt.Errorf("payload_rules names undeclared field %q", field)
		}
		if strings.ToLower(source) != "body" && !seshatHeaderPattern.MatchString(source) {
			return fmt.Errorf("payload_rules source %q for %q is malformed", source, field)
		}
	}
	for _, shape := range spec.PermissionShapes {
		if !seshatFixedPermission[shape] && !regexp.MustCompile(`^[1-9][0-9]*_uses$`).MatchString(shape) {
			return fmt.Errorf("unsupported permission shape %q", shape)
		}
	}
	if spec.Temporal != nil {
		if spec.Temporal.Mode != "scheduled" && spec.Temporal.Mode != "reviewable" && spec.Temporal.Mode != "active" {
			return fmt.Errorf("unsupported temporal mode %q", spec.Temporal.Mode)
		}
		if !seshatTemporalPolicies[spec.Temporal.MissedPolicy] {
			return fmt.Errorf("unsupported temporal missed_policy %q", spec.Temporal.MissedPolicy)
		}
		if spec.Temporal.PausePolicy != "" && spec.Temporal.PausePolicy != "hold" && spec.Temporal.PausePolicy != "missed" && spec.Temporal.PausePolicy != "require_review" {
			return fmt.Errorf("unsupported temporal pause_policy %q", spec.Temporal.PausePolicy)
		}
	}
	return nil
}

func (r *seshatRegistry) addManifest(manifest seshatManifest, path string) error {
	pluginName := manifest.Plugin.Name
	if _, exists := r.plugins[pluginName]; exists {
		return fmt.Errorf("duplicate plugin %q", pluginName)
	}
	inspection := seshatPluginInspection{Name: pluginName, Version: manifest.Plugin.Version, ManifestPath: path}
	for _, spec := range manifest.Packages {
		pkg := seshatPackage{seshatPackageSpec: spec, PluginName: pluginName, PluginVersion: manifest.Plugin.Version, ManifestPath: path, AdapterAvailable: seshatAvailableAdapters[spec.Adapter]}
		if spec.Enabled == nil {
			pkg.Enabled = boolPointer(true)
		}
		for _, code := range append([]string{spec.SubjectCode}, spec.Aliases...) {
			code = strings.ToUpper(strings.TrimSpace(code))
			if _, exists := r.packages[code]; exists {
				return fmt.Errorf("duplicate or conflicting subject code %q", code)
			}
			r.packages[code] = pkg
		}
		inspection.PackageCount++
		if packageEnabled(pkg) {
			inspection.EnabledPackages++
		}
	}
	r.plugins[pluginName] = inspection
	return nil
}

func boolPointer(value bool) *bool { return &value }

func packageEnabled(pkg seshatPackage) bool { return pkg.Enabled == nil || *pkg.Enabled }

func (r *seshatRegistry) validatePath(path string) (seshatManifest, error) {
	info, err := os.Stat(path)
	if err != nil {
		return seshatManifest{}, err
	}
	if !info.Mode().IsRegular() {
		return seshatManifest{}, errors.New("manifest path must name a regular file")
	}
	return loadSeshatManifest(path)
}

func safeSeshatFilename(name string) string {
	name = strings.ToLower(strings.TrimSpace(name))
	name = strings.NewReplacer("/", "-", "\\", "-", " ", "-").Replace(name)
	return name + ".json"
}

func (r *seshatRegistry) registerPath(path string) (seshatManifest, error) {
	manifest, err := r.validatePath(path)
	if err != nil {
		return seshatManifest{}, err
	}
	for _, spec := range manifest.Packages {
		for _, code := range append([]string{spec.SubjectCode}, spec.Aliases...) {
			if _, exists := r.packages[strings.ToUpper(strings.TrimSpace(code))]; exists {
				return seshatManifest{}, fmt.Errorf("duplicate or conflicting subject code %q", code)
			}
		}
	}
	destination := filepath.Join(r.dir, safeSeshatFilename(manifest.Plugin.Name))
	if _, err := os.Stat(destination); err == nil {
		return seshatManifest{}, fmt.Errorf("plugin %q is already registered", manifest.Plugin.Name)
	} else if !os.IsNotExist(err) {
		return seshatManifest{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return seshatManifest{}, err
	}
	temporary := destination + ".tmp"
	if err := os.WriteFile(temporary, data, 0o600); err != nil {
		return seshatManifest{}, err
	}
	if err := os.Rename(temporary, destination); err != nil {
		_ = os.Remove(temporary)
		return seshatManifest{}, err
	}
	if err := r.addManifest(manifest, destination); err != nil {
		_ = os.Remove(destination)
		return seshatManifest{}, err
	}
	return manifest, nil
}

func (r *seshatRegistry) packageFor(subjectCode string) (seshatPackage, error) {
	pkg, ok := r.packages[strings.ToUpper(strings.TrimSpace(subjectCode))]
	if !ok {
		return seshatPackage{}, fmt.Errorf("unknown subject code %q", subjectCode)
	}
	if !packageEnabled(pkg) {
		return seshatPackage{}, fmt.Errorf("package %q is disabled", subjectCode)
	}
	return pkg, nil
}

func (r *seshatRegistry) packagesInspection() []seshatPackage {
	seen := map[string]bool{}
	result := make([]seshatPackage, 0, len(r.packages))
	for _, pkg := range r.packages {
		if seen[pkg.SubjectCode] {
			continue
		}
		seen[pkg.SubjectCode] = true
		result = append(result, pkg)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].SubjectCode < result[j].SubjectCode })
	return result
}

func (r *seshatRegistry) pluginsInspection() []seshatPluginInspection {
	result := make([]seshatPluginInspection, 0, len(r.plugins))
	for _, plugin := range r.plugins {
		result = append(result, plugin)
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Name < result[j].Name })
	return result
}

type seshatMissive struct {
	Type    string
	Headers map[string]string
	Body    string
}

func parseSeshatMissive(input string) (seshatMissive, error) {
	input = strings.ReplaceAll(input, "\r\n", "\n")
	input = strings.ReplaceAll(input, "\r", "\n")
	parts := strings.SplitN(input, "\n\n", 2)
	headerText := parts[0]
	body := ""
	if len(parts) == 2 {
		body = parts[1]
	}
	missive := seshatMissive{Headers: map[string]string{}, Body: body}
	for index, line := range strings.Split(headerText, "\n") {
		line = strings.TrimSuffix(line, "\r")
		colon := strings.IndexByte(line, ':')
		if colon < 1 {
			return seshatMissive{}, fmt.Errorf("missive header line %d is malformed", index+1)
		}
		key := strings.TrimSpace(line[:colon])
		value := strings.TrimSpace(line[colon+1:])
		if !seshatHeaderPattern.MatchString(key) {
			return seshatMissive{}, fmt.Errorf("missive header %q is malformed", key)
		}
		key = strings.ToUpper(key)
		if _, exists := missive.Headers[key]; exists {
			return seshatMissive{}, fmt.Errorf("missive header %q is duplicated", key)
		}
		missive.Headers[key] = value
		if key == "TYPE" {
			missive.Type = strings.ToUpper(value)
		}
	}
	if !seshatSubjectCodePattern.MatchString(missive.Type) {
		return seshatMissive{}, errors.New("missive requires an uppercase TYPE subject code")
	}
	return missive, nil
}

type seshatNormalizedPackage struct {
	SubjectCode       string            `json:"subject_code"`
	PackageVersion    string            `json:"package_version"`
	Plugin            string            `json:"plugin"`
	Capability        string            `json:"capability"`
	Adapter           string            `json:"adapter"`
	AdapterAvailable  bool              `json:"adapter_available"`
	Fields            map[string]string `json:"fields"`
	Returns           []string          `json:"returns"`
	PermissionClass   string            `json:"permission_class"`
	PermissionShapes  []string          `json:"permission_shapes"`
	SourceMissiveHash string            `json:"source_missive_sha256"`
}

func constructSeshatPackage(pkg seshatPackage, input string) (seshatNormalizedPackage, error) {
	missive, err := parseSeshatMissive(input)
	if err != nil {
		return seshatNormalizedPackage{}, err
	}
	if missive.Type != strings.ToUpper(pkg.SubjectCode) {
		return seshatNormalizedPackage{}, fmt.Errorf("missive TYPE %q does not match package %q", missive.Type, pkg.SubjectCode)
	}
	fields := map[string]string{}
	allFields := append(append([]string{}, pkg.RequiredFields...), pkg.OptionalFields...)
	for _, field := range allFields {
		source := pkg.PayloadRules[field]
		if source == "" {
			source = strings.ToUpper(strings.ReplaceAll(field, "_", "-"))
		}
		var value string
		if strings.EqualFold(source, "body") {
			value = missive.Body
		} else {
			value = missive.Headers[strings.ToUpper(source)]
		}
		if value == "" {
			for _, required := range pkg.RequiredFields {
				if required == field {
					return seshatNormalizedPackage{}, fmt.Errorf("missing required field %q", field)
				}
			}
			continue
		}
		fields[field] = value
	}
	digest := sha256.Sum256([]byte(input))
	return seshatNormalizedPackage{
		SubjectCode: pkg.SubjectCode, PackageVersion: pkg.Version, Plugin: pkg.PluginName,
		Capability: pkg.Capability, Adapter: pkg.Adapter, AdapterAvailable: pkg.AdapterAvailable,
		Fields: fields, Returns: append([]string{}, pkg.Returns...), PermissionClass: pkg.PermissionClass,
		PermissionShapes: append([]string{}, pkg.PermissionShapes...), SourceMissiveHash: hex.EncodeToString(digest[:]),
	}, nil
}

func seshatApolloIndex(packages []seshatPackage) []map[string]any {
	result := make([]map[string]any, 0, len(packages))
	for _, pkg := range packages {
		if !packageEnabled(pkg) {
			continue
		}
		result = append(result, map[string]any{
			"subject_code": pkg.SubjectCode, "description": pkg.Description,
			"fields":          append(append([]string{}, pkg.RequiredFields...), pkg.OptionalFields...),
			"required_fields": pkg.RequiredFields, "optional_fields": pkg.OptionalFields,
			"package_version": pkg.Version,
		})
	}
	return result
}

type seshatPathRequest struct {
	Path string `json:"path"`
}
type seshatPackageRequest struct {
	SubjectCode string `json:"subject_code"`
	Missive     string `json:"missive"`
}

func (d *daemon) handleSeshatPlugins(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, d.seshat.pluginsInspection())
}

func (d *daemon) handleSeshatPackages(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, d.seshat.packagesInspection())
}

func (d *daemon) handleSeshatPackage(w http.ResponseWriter, r *http.Request) {
	subject := r.PathValue("subject")
	pkg, err := d.seshat.packageFor(subject)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, pkg)
}

func (d *daemon) handleSeshatIndex(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, seshatApolloIndex(d.seshat.packagesInspection()))
}

func (d *daemon) handleSeshatHelp(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"result": seshatHelpText(d.seshat.dir)})
}

func (d *daemon) handleSeshatSchema(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, seshatManifestSchema())
}

func (d *daemon) handleSeshatValidate(w http.ResponseWriter, r *http.Request) {
	var request seshatPathRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Path) == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	manifest, err := d.seshat.validatePath(request.Path)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"valid": true, "plugin": manifest.Plugin, "packages": manifest.Packages, "authority": "none"})
}

func (d *daemon) handleSeshatRegister(w http.ResponseWriter, r *http.Request) {
	if r.Header.Get(controlTokenHeader) != d.controlToken {
		writeError(w, http.StatusForbidden, errors.New("local CLI control token required"))
		return
	}
	var request seshatPathRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Path) == "" {
		writeError(w, http.StatusBadRequest, errors.New("path is required"))
		return
	}
	manifest, err := d.seshat.registerPath(request.Path)
	if err != nil {
		writeError(w, http.StatusConflict, err)
		return
	}
	writeJSON(w, http.StatusCreated, map[string]any{"registered": true, "plugin": manifest.Plugin, "packages": manifest.Packages, "authority": "none", "registry_path": d.seshat.dir})
}

func (d *daemon) handleSeshatPackageMissive(w http.ResponseWriter, r *http.Request) {
	var request seshatPackageRequest
	if err := decodeJSON(r, &request); err != nil || strings.TrimSpace(request.Missive) == "" {
		writeError(w, http.StatusBadRequest, errors.New("missive is required"))
		return
	}
	missive, err := parseSeshatMissive(request.Missive)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	subject := request.SubjectCode
	if strings.TrimSpace(subject) == "" {
		subject = missive.Type
	}
	pkg, err := d.seshat.packageFor(subject)
	if err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	normalized, err := constructSeshatPackage(pkg, request.Missive)
	if err != nil {
		writeError(w, http.StatusUnprocessableEntity, err)
		return
	}
	if !normalized.AdapterAvailable {
		writeJSON(w, http.StatusServiceUnavailable, map[string]any{"error": "adapter unavailable", "package": normalized})
		return
	}
	writeJSON(w, http.StatusOK, normalized)
}

func seshatManifestSchema() map[string]any {
	return map[string]any{
		"schema_version":    seshatManifestSchemaVersion,
		"manifest_path":     "config/plugins/manifest-template.json",
		"required":          map[string]any{"plugin": []string{"name", "version"}, "package": []string{"subject_code", "version", "description", "capability", "adapter", "required_fields", "returns", "permission_class", "permission_shapes"}},
		"permission_shapes": []string{"once", "project", "this_project", "bounded_uses", "until_revoked", "temporary", "until_timestamp", "N_uses"},
		"mapping_rules":     "field name to a missive header (for example TO) or the literal body",
		"temporal":          map[string]any{"mode": "scheduled|reviewable|active", "missed_policy": []string{"expire", "resurface", "backfill", "recompute", "require_review"}, "pause_policy": []string{"hold", "missed", "require_review"}},
		"authority":         "registration and package construction grant no operational authority",
	}
}

func seshatHelpText(registryDir string) string {
	return fmt.Sprintf("Seshat teaches Pantheon how to package small subject codes.\n\nManifest workflow:\n  pantheon plugin schema\n  pantheon plugin validate <path>\n  pantheon plugin register <path>\n  pantheon show plugins\n  pantheon show packages\n  pantheon show package <subject-code>\n\nManifests are JSON files copied into %s and loaded immediately. Edit enabled:false and restart to disable a registered package. Registration grants no authority; Kernel 13 grants remain authoritative. A manifest describes a package contract, but a genuinely new adapter still requires bounded implementation code. See config/plugins/manifest-template.json and docs/seshat-plugin-extension.md.", registryDir)
}
