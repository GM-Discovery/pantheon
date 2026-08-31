package pantheon

import (
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

const (
	reportAssemblyEvent  = "event"
	reportAssemblyDaily  = "daily"
	reportAssemblyWeekly = "weekly"
	reportAssemblyReady  = "ready_for_apollo"
)

type reportAssemblyRequest struct {
	Kind       string   `json:"kind"`
	PeriodDate string   `json:"period_date"`
	SourceRefs []string `json:"source_refs,omitempty"`
}

type reportAssemblySource struct {
	Path       string `json:"path"`
	Citation   string `json:"citation"`
	Kind       string `json:"kind"`
	SHA256     string `json:"sha256"`
	Bytes      int64  `json:"bytes"`
	ModifiedAt string `json:"modified_at"`
}

type reportAssemblyReceipt struct {
	AssemblyID   string                 `json:"assembly_id"`
	Kind         string                 `json:"kind"`
	PeriodDate   string                 `json:"period_date"`
	CreatedAt    string                 `json:"created_at"`
	ManifestPath string                 `json:"manifest_path"`
	BodySHA256   string                 `json:"body_sha256"`
	SourceCount  int                    `json:"source_count"`
	SourceRefs   []string               `json:"source_refs"`
	Sources      []reportAssemblySource `json:"sources,omitempty"`
	Status       string                 `json:"status"`
	Duplicate    bool                   `json:"duplicate,omitempty"`
	AuditID      string                 `json:"audit_id,omitempty"`
}

func initializeReportAssemblySchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS report_assemblies (
		assembly_id TEXT PRIMARY KEY, kind TEXT NOT NULL, period_date TEXT NOT NULL,
		assembly_key TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL, manifest_path TEXT NOT NULL,
		body_sha256 TEXT NOT NULL, source_count INTEGER NOT NULL, source_refs_json TEXT NOT NULL,
		status TEXT NOT NULL
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS report_assemblies_period_idx ON report_assemblies(kind, period_date, created_at DESC)`)
	return err
}

func validateReportAssemblyRequest(request reportAssemblyRequest) (string, error) {
	request.Kind = strings.TrimSpace(request.Kind)
	if request.Kind != reportAssemblyEvent && request.Kind != reportAssemblyDaily && request.Kind != reportAssemblyWeekly {
		return "", errors.New("kind must be event, daily, or weekly")
	}
	period := strings.TrimSpace(request.PeriodDate)
	if _, err := time.Parse("2006-01-02", period); err != nil {
		return "", errors.New("period_date must be YYYY-MM-DD")
	}
	if request.Kind == reportAssemblyEvent && len(request.SourceRefs) == 0 {
		return "", errors.New("event assembly requires at least one source_refs entry")
	}
	return period, nil
}

func reportAssemblyPeriodForTimestamp(value string, location *time.Location) string {
	parsed, err := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
	if err != nil {
		return ""
	}
	return parsed.In(location).Format("2006-01-02")
}

func (d *daemon) reportAssemblySourcePath(ref string) (string, string, error) {
	if strings.TrimSpace(d.cfg.VaultPath) == "" {
		return "", "", errors.New("report assembly requires a configured Vault")
	}
	ref = strings.TrimSpace(strings.Trim(ref, "\""))
	if ref == "" || strings.IndexByte(ref, 0) >= 0 {
		return "", "", errors.New("source_refs cannot be empty or contain NUL bytes")
	}
	clean := filepath.Clean(filepath.FromSlash(ref))
	if !filepath.IsAbs(clean) {
		clean = filepath.Join(d.cfg.VaultPath, clean)
	}
	clean = filepath.Clean(clean)
	if !pathWithin(d.cfg.VaultPath, clean) {
		return "", "", fmt.Errorf("source reference escapes configured Vault: %s", ref)
	}
	if err := ensureNoSymlinkComponents(clean); err != nil {
		return "", "", err
	}
	relative, err := filepath.Rel(d.cfg.VaultPath, clean)
	if err != nil || relative == "." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", "", fmt.Errorf("source reference is not Vault-relative: %s", ref)
	}
	return clean, filepath.ToSlash(relative), nil
}

func reportAssemblySourceKind(relative string) string {
	switch {
	case strings.HasPrefix(relative, "80 Pantheon Raw/"):
		return "transcript_or_present_artifact"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Apollo/"):
		return "apollo_report"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Integrated Events/"):
		return "integrated_event_report"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Weekly/"):
		return "weekly_report"
	default:
		return "vault_source"
	}
}

func (d *daemon) inspectReportAssemblySource(ref string) (reportAssemblySource, error) {
	absolute, relative, err := d.reportAssemblySourcePath(ref)
	if err != nil {
		return reportAssemblySource{}, err
	}
	file, err := openVaultMarkdown(absolute)
	if err != nil {
		return reportAssemblySource{}, fmt.Errorf("open report source %s: %w", relative, err)
	}
	body, readErr := io.ReadAll(file)
	closeErr := file.Close()
	if readErr != nil {
		return reportAssemblySource{}, readErr
	}
	if closeErr != nil {
		return reportAssemblySource{}, closeErr
	}
	info, err := os.Stat(absolute)
	if err != nil {
		return reportAssemblySource{}, err
	}
	return reportAssemblySource{
		Path:       relative,
		Citation:   apolloCitation(d.cfg.VaultPath, absolute),
		Kind:       reportAssemblySourceKind(relative),
		SHA256:     sha256Bytes(body),
		Bytes:      int64(len(body)),
		ModifiedAt: info.ModTime().UTC().Format(time.RFC3339Nano),
	}, nil
}

func addReportAssemblyRef(ref string, refs map[string]bool) error {
	ref = filepath.ToSlash(filepath.Clean(filepath.FromSlash(strings.TrimSpace(ref))))
	if ref == "." || ref == "" || strings.HasPrefix(ref, "../") || filepath.IsAbs(ref) {
		return fmt.Errorf("invalid Vault-relative report source: %s", ref)
	}
	refs[ref] = true
	return nil
}

func (d *daemon) autoDailyReportRefs(period string) ([]string, error) {
	location, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return nil, err
	}
	date, err := time.ParseInLocation("2006-01-02", period, location)
	if err != nil {
		return nil, err
	}
	return d.autoReportRefsBetween(date, date)
}

func (d *daemon) autoReportRefsBetween(start, end time.Time) ([]string, error) {
	return d.autoReportRefsBetweenWithRecipe(start, end, libraReportRecipe{SourceFamilies: []string{"all"}})
}

func libraSourceFamily(relative string) string {
	relative = filepath.ToSlash(relative)
	switch {
	case strings.HasPrefix(relative, "80 Pantheon Raw/"):
		return "transcripts"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Apollo/"):
		return "apollo_reports"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Integrated Events/"):
		return "integrated_events"
	case strings.HasPrefix(relative, "70 Pantheon Reports/Weekly/"):
		return "weekly_reports"
	case strings.HasPrefix(relative, "10 Operations/Situation Briefs/"):
		return "situation_briefs"
	default:
		return "vault"
	}
}

func libraRecipeAllowsPath(recipe libraReportRecipe, relative string) bool {
	family := libraSourceFamily(relative)
	// Generated reports are useful when deliberately requested, but default
	// "all" evidence must not feed yesterday's Apollo output back into today's
	// report. Otherwise an empty intake window can still produce a confident
	// recursive summary of an older summary.
	generated := family == "apollo_reports" || family == "weekly_reports" || family == "situation_briefs"
	if generated && (len(recipe.SourceFamilies) == 0 || containsLibraString(recipe.SourceFamilies, "all")) && !containsLibraString(recipe.SourceFamilies, family) {
		return false
	}
	if len(recipe.SourceFamilies) > 0 && !containsLibraString(recipe.SourceFamilies, "all") {
		if !containsLibraString(recipe.SourceFamilies, family) {
			return false
		}
	}
	if len(recipe.IncludePaths) > 0 {
		relative = filepath.ToSlash(filepath.Clean(filepath.FromSlash(relative)))
		matched := false
		for _, prefix := range recipe.IncludePaths {
			prefix = filepath.ToSlash(filepath.Clean(filepath.FromSlash(prefix)))
			if relative == prefix || strings.HasPrefix(relative, strings.TrimSuffix(prefix, "/")+"/") {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	return true
}

func (d *daemon) autoReportRefsBetweenWithRecipe(start, end time.Time, recipe libraReportRecipe) ([]string, error) {
	location := start.Location()
	startDate, endDate := start.In(location).Format("2006-01-02"), end.In(location).Format("2006-01-02")
	refs := map[string]bool{}
	inRange := func(value string) bool {
		parsed, parseErr := time.Parse(time.RFC3339Nano, strings.TrimSpace(value))
		if parseErr != nil {
			return false
		}
		date := parsed.In(location).Format("2006-01-02")
		return date >= startDate && date <= endDate
	}
	rows, err := d.db.Query(`SELECT normalized_path, occurred_at FROM external_intake_receipts WHERE normalized_path IS NOT NULL AND normalized_path != ''`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var path, occurredAt string
		if err := rows.Scan(&path, &occurredAt); err != nil {
			rows.Close()
			return nil, err
		}
		if !inRange(occurredAt) {
			continue
		}
		if absolute, relative, pathErr := d.reportAssemblySourcePath(path); pathErr == nil {
			_ = absolute
			if err := addReportAssemblyRef(relative, refs); err != nil {
				rows.Close()
				return nil, err
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	rows, err = d.db.Query(`SELECT report_path, created_at FROM apollo_reports WHERE report_path IS NOT NULL AND report_path != ''`)
	if err != nil {
		return nil, err
	}
	for rows.Next() {
		var path, createdAt string
		if err := rows.Scan(&path, &createdAt); err != nil {
			rows.Close()
			return nil, err
		}
		if !inRange(createdAt) {
			continue
		}
		if _, relative, pathErr := d.reportAssemblySourcePath(path); pathErr == nil {
			if err := addReportAssemblyRef(relative, refs); err != nil {
				rows.Close()
				return nil, err
			}
		}
	}
	if err := rows.Close(); err != nil {
		return nil, err
	}
	for _, root := range []string{
		filepath.Join(d.cfg.VaultPath, "70 Pantheon Reports", "Integrated Events"),
		filepath.Join(d.cfg.VaultPath, "10 Operations", "Situation Briefs"),
	} {
		if err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
			if errors.Is(walkErr, os.ErrNotExist) {
				return filepath.SkipDir
			}
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || strings.ToLower(filepath.Ext(entry.Name())) != ".md" {
				return nil
			}
			nameDate := strings.TrimSpace(entry.Name()[:reportMinInt(len(entry.Name()), len("2006-01-02"))])
			if _, parseErr := time.Parse("2006-01-02", nameDate); parseErr != nil || nameDate < startDate || nameDate > endDate {
				return nil
			}
			_, relative, pathErr := d.reportAssemblySourcePath(path)
			if pathErr != nil {
				return pathErr
			}
			return addReportAssemblyRef(relative, refs)
		}); err != nil && !errors.Is(err, os.ErrNotExist) {
			return nil, err
		}
	}
	result := make([]string, 0, len(refs))
	for ref := range refs {
		if libraRecipeAllowsPath(recipe, ref) {
			result = append(result, ref)
		}
	}
	sort.Strings(result)
	return result, nil
}

func reportMinInt(left, right int) int {
	if left < right {
		return left
	}
	return right
}

func reportAssemblyKey(kind, period string, refs []string) string {
	return sha256Bytes([]byte(kind + "\x00" + period + "\x00" + strings.Join(refs, "\x00")))
}

func reportAssemblySuffix(key string) string {
	key = strings.TrimPrefix(key, "sha256:")
	if len(key) > 12 {
		return key[:12]
	}
	return key
}

func reportAssemblyManifestPath(vault, kind, period, key string) string {
	if kind == reportAssemblyEvent {
		return filepath.Join(vault, "70 Pantheon Reports", "Integrated Events", period+"-assembled-event-"+reportAssemblySuffix(key)+".md")
	}
	if kind == reportAssemblyWeekly {
		return filepath.Join(vault, "70 Pantheon Reports", "Weekly", period+" Weekly Report — Evidence-"+reportAssemblySuffix(key)+".md")
	}
	return filepath.Join(vault, "10 Operations", "Situation Briefs", period+" Situation Brief — Evidence-"+reportAssemblySuffix(key)+".md")
}

func renderReportAssemblyManifest(receipt reportAssemblyReceipt) ([]byte, string) {
	var content strings.Builder
	if receipt.Kind == reportAssemblyEvent {
		fmt.Fprintf(&content, "# Event Evidence Archive — %s\n\n", receipt.PeriodDate)
	} else if receipt.Kind == reportAssemblyWeekly {
		fmt.Fprintf(&content, "# Weekly Report Evidence — %s\n\n", receipt.PeriodDate)
	} else {
		fmt.Fprintf(&content, "# Situation Brief Evidence — %s\n\n", receipt.PeriodDate)
	}
	content.WriteString("This is a deterministic report-assembly receipt. It records which verified Vault sources were selected for Apollo. It does not infer importance, create authority, or execute calendar, email, or notification actions.\n\n")
	content.WriteString("## Authorized source set\n\n")
	if len(receipt.Sources) == 0 {
		content.WriteString("No verified Vault sources were available for this assembly.\n")
	} else {
		for index, source := range receipt.Sources {
			fmt.Fprintf(&content, "%d. `%s` — %s\n", index+1, source.Citation, source.Kind)
			fmt.Fprintf(&content, "   - Path: `%s`\n   - SHA-256: `%s`\n   - Bytes: %d\n   - Modified: `%s`\n", source.Path, source.SHA256, source.Bytes, source.ModifiedAt)
		}
	}
	content.WriteString("\n## Handoff boundary\n\n")
	content.WriteString("Apollo may use the full citation tokens listed above and must name every concrete request, action, commitment, decision, or proposal in plain language. The source set is evidence, not authorization.\n")
	body := content.String()
	bodyHash := sha256Bytes([]byte(body))
	var frontmatter strings.Builder
	frontmatter.WriteString("---\n")
	fmt.Fprintf(&frontmatter, "pantheon:\n  kind: %s\n  assembly_id: %s\n  period_date: %s\n  status: %s\n  source_count: %d\n  body_sha256: %s\n  source_refs:\n", yamlQuote("report-assembly"), yamlQuote(receipt.AssemblyID), yamlQuote(receipt.PeriodDate), yamlQuote(receipt.Status), receipt.SourceCount, yamlQuote(bodyHash))
	for _, source := range receipt.Sources {
		fmt.Fprintf(&frontmatter, "    - %s\n", yamlQuote(source.Path))
	}
	frontmatter.WriteString("---\n\n")
	return []byte(frontmatter.String() + body), bodyHash
}

func writeReportAssemblyFile(path string, data []byte) error {
	if err := ensureNoSymlinkComponents(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if _, err := os.Lstat(path); err == nil {
		return errors.New("report assembly manifest already exists and will not be overwritten")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".report-assembly-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		tmp.Close()
		return err
	}
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func (d *daemon) existingReportAssembly(key string) (reportAssemblyReceipt, error) {
	var receipt reportAssemblyReceipt
	var refsJSON string
	err := d.db.QueryRow(`SELECT assembly_id, kind, period_date, created_at, manifest_path, body_sha256, source_count, source_refs_json, status FROM report_assemblies WHERE assembly_key = ?`, key).Scan(
		&receipt.AssemblyID, &receipt.Kind, &receipt.PeriodDate, &receipt.CreatedAt, &receipt.ManifestPath, &receipt.BodySHA256, &receipt.SourceCount, &refsJSON, &receipt.Status)
	if errors.Is(err, sql.ErrNoRows) {
		return reportAssemblyReceipt{}, nil
	}
	if err != nil {
		return reportAssemblyReceipt{}, err
	}
	if err := json.Unmarshal([]byte(refsJSON), &receipt.SourceRefs); err != nil {
		return reportAssemblyReceipt{}, err
	}
	receipt.Duplicate = true
	return receipt, nil
}

func (d *daemon) assembleReport(request reportAssemblyRequest) (reportAssemblyReceipt, error) {
	period, err := validateReportAssemblyRequest(request)
	if err != nil {
		return reportAssemblyReceipt{}, err
	}
	request.Kind = strings.TrimSpace(request.Kind)
	request.PeriodDate = period
	if request.Kind == reportAssemblyDaily && len(request.SourceRefs) == 0 {
		request.SourceRefs, err = d.autoDailyReportRefs(period)
		if err != nil {
			return reportAssemblyReceipt{}, err
		}
	}
	refs := map[string]bool{}
	for _, ref := range request.SourceRefs {
		absolute, relative, pathErr := d.reportAssemblySourcePath(ref)
		if pathErr != nil {
			return reportAssemblyReceipt{}, pathErr
		}
		_ = absolute
		if err := addReportAssemblyRef(relative, refs); err != nil {
			return reportAssemblyReceipt{}, err
		}
	}
	request.SourceRefs = request.SourceRefs[:0]
	for ref := range refs {
		request.SourceRefs = append(request.SourceRefs, ref)
	}
	sort.Strings(request.SourceRefs)
	if len(request.SourceRefs) == 0 {
		return reportAssemblyReceipt{}, fmt.Errorf("no verified Vault sources found for %s assembly on %s", request.Kind, period)
	}
	key := reportAssemblyKey(request.Kind, period, request.SourceRefs)
	if existing, err := d.existingReportAssembly(key); err != nil {
		return reportAssemblyReceipt{}, err
	} else if existing.AssemblyID != "" {
		return existing, nil
	}
	sources := make([]reportAssemblySource, 0, len(request.SourceRefs))
	for _, ref := range request.SourceRefs {
		source, sourceErr := d.inspectReportAssemblySource(ref)
		if sourceErr != nil {
			return reportAssemblyReceipt{}, sourceErr
		}
		sources = append(sources, source)
	}
	assemblyID, err := newLocalID("assembly-")
	if err != nil {
		return reportAssemblyReceipt{}, err
	}
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	receipt := reportAssemblyReceipt{AssemblyID: assemblyID, Kind: request.Kind, PeriodDate: period, CreatedAt: createdAt, SourceCount: len(sources), SourceRefs: request.SourceRefs, Sources: sources, Status: reportAssemblyReady}
	manifestPath := reportAssemblyManifestPath(d.cfg.VaultPath, request.Kind, period, key)
	receipt.ManifestPath = manifestPath
	data, bodyHash := renderReportAssemblyManifest(receipt)
	receipt.BodySHA256 = bodyHash
	if err := writeReportAssemblyFile(manifestPath, data); err != nil {
		return reportAssemblyReceipt{}, err
	}
	refsJSON, err := json.Marshal(request.SourceRefs)
	if err != nil {
		return reportAssemblyReceipt{}, err
	}
	if _, err := d.db.Exec(`INSERT INTO report_assemblies (assembly_id, kind, period_date, assembly_key, created_at, manifest_path, body_sha256, source_count, source_refs_json, status) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, assemblyID, request.Kind, period, key, createdAt, manifestPath, bodyHash, len(sources), string(refsJSON), reportAssemblyReady); err != nil {
		return reportAssemblyReceipt{}, err
	}
	return receipt, nil
}

func (d *daemon) reportAssembliesInspection(limit int) ([]reportAssemblyReceipt, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT assembly_id, kind, period_date, created_at, manifest_path, body_sha256, source_count, source_refs_json, status FROM report_assemblies ORDER BY created_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []reportAssemblyReceipt
	for rows.Next() {
		var receipt reportAssemblyReceipt
		var refsJSON string
		if err := rows.Scan(&receipt.AssemblyID, &receipt.Kind, &receipt.PeriodDate, &receipt.CreatedAt, &receipt.ManifestPath, &receipt.BodySHA256, &receipt.SourceCount, &refsJSON, &receipt.Status); err != nil {
			return nil, err
		}
		if err := json.Unmarshal([]byte(refsJSON), &receipt.SourceRefs); err != nil {
			return nil, err
		}
		result = append(result, receipt)
	}
	return result, rows.Err()
}

func (d *daemon) handleReportAssembly(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if r.Method == http.MethodPost {
		if r.Header.Get(controlTokenHeader) == "" || r.Header.Get(controlTokenHeader) != d.controlToken {
			writeError(w, http.StatusForbidden, errors.New("report assembly requires the local CLI control boundary"))
			return
		}
		if d.blockIfPaused(w, "POST /v1/reports/assemble", d.cfg.VaultPath, "report.assemble") {
			return
		}
		var request reportAssemblyRequest
		if err := decodeJSON(r, &request); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		receipt, err := d.assembleReport(request)
		if err != nil {
			id := d.audit("POST /v1/reports/assemble", request.PeriodDate, "report.assemble", "", "failed", "", err.Error(), time.Since(started))
			_ = id
			writeError(w, http.StatusBadRequest, err)
			return
		}
		receipt.AuditID = d.audit("POST /v1/reports/assemble", receipt.PeriodDate, "report.assemble", "", "completed", receipt.AssemblyID, "", time.Since(started))
		if receipt.Duplicate {
			writeJSON(w, http.StatusOK, receipt)
			return
		}
		writeJSON(w, http.StatusCreated, receipt)
		return
	}
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if _, err := fmt.Sscanf(rawLimit, "%d", &limit); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("limit must be an integer"))
			return
		}
	}
	assemblies, err := d.reportAssembliesInspection(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"assemblies": assemblies})
}

func (d *daemon) handleShowReportAssembly(w http.ResponseWriter, r *http.Request) {
	var receipt reportAssemblyReceipt
	var refsJSON string
	err := d.db.QueryRow(`SELECT assembly_id, kind, period_date, created_at, manifest_path, body_sha256, source_count, source_refs_json, status FROM report_assemblies WHERE assembly_id = ?`, r.PathValue("id")).Scan(
		&receipt.AssemblyID, &receipt.Kind, &receipt.PeriodDate, &receipt.CreatedAt, &receipt.ManifestPath, &receipt.BodySHA256, &receipt.SourceCount, &refsJSON, &receipt.Status)
	if errors.Is(err, sql.ErrNoRows) {
		writeError(w, http.StatusNotFound, errors.New("report assembly not found"))
		return
	}
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := json.Unmarshal([]byte(refsJSON), &receipt.SourceRefs); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, receipt)
}
