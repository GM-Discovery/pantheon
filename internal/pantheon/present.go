package pantheon

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const (
	processingUnprocessed        = "unprocessed"
	processingProcessing         = "processing"
	processingVerified           = "verified"
	processingBlockedUnsupported = "blocked_unsupported"
	processingBlockedProcessor   = "blocked_missing_processor"
	processingFailed             = "failed"
	retentionArmed               = "retention_armed"
)

type presentRequest struct {
	Request      string `json:"request,omitempty"`
	OccurrenceID string `json:"occurrence_id,omitempty"`
	ProcessNew   bool   `json:"process_new,omitempty"`
}

type presentResult struct {
	OccurrenceID    string   `json:"occurrence_id,omitempty"`
	SourcePath      string   `json:"source_path,omitempty"`
	SourceType      string   `json:"source_type,omitempty"`
	NormalizedPath  string   `json:"normalized_path,omitempty"`
	Processor       string   `json:"processor,omitempty"`
	State           string   `json:"processing_state"`
	Verification    string   `json:"verification_state"`
	RetentionState  string   `json:"retention_state"`
	RetentionDueAt  string   `json:"retention_due_at,omitempty"`
	MissiveID       string   `json:"missive_id,omitempty"`
	Warnings        []string `json:"warnings,omitempty"`
	AlreadyVerified bool     `json:"already_verified,omitempty"`
	Error           string   `json:"error,omitempty"`
}

type presentResponse struct {
	Result     string             `json:"result"`
	Request    string             `json:"request,omitempty"`
	Resolution string             `json:"resolution,omitempty"`
	Candidates []presentCandidate `json:"candidates,omitempty"`
	Processed  []presentResult    `json:"processed,omitempty"`
	Occurrence *presentResult     `json:"occurrence,omitempty"`
}

type presentCandidate struct {
	OccurrenceID string `json:"occurrence_id"`
	Path         string `json:"path"`
	Type         string `json:"type,omitempty"`
	ObservedAt   string `json:"observed_at"`
}

type normalizedArtifact struct {
	Body             []byte
	Processor        string
	ProcessorVersion string
	Warnings         []string
}

type processorStatus struct {
	Name     string `json:"name"`
	State    string `json:"state"`
	Binary   string `json:"binary,omitempty"`
	Model    string `json:"model,omitempty"`
	Language string `json:"language,omitempty"`
	Detail   string `json:"detail,omitempty"`
}

func ensureIntakeColumn(db *sql.DB, table, column, definition string) error {
	rows, err := db.Query("PRAGMA table_info(" + table + ")")
	if err != nil {
		return err
	}
	defer rows.Close()
	for rows.Next() {
		var cid int
		var name, typ string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &typ, &notNull, &defaultValue, &pk); err != nil {
			return err
		}
		if name == column {
			return nil
		}
	}
	if err := rows.Err(); err != nil {
		return err
	}
	_, err = db.Exec("ALTER TABLE " + table + " ADD COLUMN " + column + " " + definition)
	return err
}

func effectiveRawPath(cfg config) string {
	if cfg.RawPath != "" {
		return cfg.RawPath
	}
	if cfg.VaultPath == "" {
		return ""
	}
	return filepath.Join(cfg.VaultPath, "80 Pantheon Raw")
}

func sha256Bytes(data []byte) string {
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}

func safeRelativePath(relative string) (string, error) {
	if relative == "" || strings.IndexByte(relative, 0) >= 0 {
		return "", errors.New("relative path is empty or contains a NUL byte")
	}
	relative = strings.ReplaceAll(relative, "\\", "/")
	if filepath.IsAbs(relative) {
		return "", errors.New("relative path must not be absolute")
	}
	parts := strings.Split(relative, "/")
	for _, part := range parts {
		if part == ".." {
			return "", errors.New("relative path traversal is not allowed")
		}
	}
	clean := filepath.Clean(filepath.FromSlash(relative))
	if clean == "." || clean == ".." || strings.HasPrefix(clean, ".."+string(filepath.Separator)) {
		return "", errors.New("relative path escapes its root")
	}
	return clean, nil
}

func pathWithin(root, target string) bool {
	rel, err := filepath.Rel(filepath.Clean(root), filepath.Clean(target))
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) && !filepath.IsAbs(rel)
}

func sourcePathForOccurrence(cfg config, occurrence intakeOccurrence) (string, error) {
	relative, err := safeRelativePath(occurrence.RelativePath)
	if err != nil {
		return "", err
	}
	if cfg.IntakePath == "" {
		return "", errors.New("Intake path is unavailable")
	}
	path := filepath.Join(cfg.IntakePath, relative)
	if !pathWithin(cfg.IntakePath, path) {
		return "", errors.New("source path escapes configured Intake")
	}
	return path, nil
}

func rawRelativePath(occurrence intakeOccurrence) (string, error) {
	relative, err := safeRelativePath(occurrence.RelativePath)
	if err != nil {
		return "", err
	}
	ext := strings.ToLower(filepath.Ext(relative))
	classification := ""
	switch ext {
	case ".md", ".txt":
		if strings.Contains(strings.ToLower(relative), "transcript") {
			classification = "Transcripts"
		} else {
			classification = "Text"
		}
	case ".pdf", ".docx":
		classification = "Documents"
	case ".wav", ".mp3", ".m4a", ".flac", ".ogg", ".mp4", ".mov":
		classification = "Transcripts"
	default:
		return "", fmt.Errorf("unsupported source extension %q", ext)
	}
	observed, err := time.Parse(time.RFC3339Nano, occurrence.FirstSeenAt)
	if err != nil {
		return "", fmt.Errorf("invalid occurrence timestamp: %w", err)
	}
	base := strings.TrimSuffix(relative, filepath.Ext(relative)) + ".md"
	return filepath.Join(classification, observed.Format("2006"), observed.Format("01"), base), nil
}

func (d *daemon) loadIntakeOccurrence(id string) (intakeOccurrence, error) {
	var item intakeOccurrence
	err := d.db.QueryRow(`SELECT o.occurrence_id, o.content_id, o.relative_path, o.original_filename, c.size, c.type, c.extension,
        o.first_seen_at, o.last_seen_at, o.state, o.father_time_year, o.processing_state,
        o.retention_policy_days, o.retention_state, COALESCE(o.retention_due_at, ''), o.chronicle_state,
        COALESCE(o.processing_started_at, ''), COALESCE(o.normalized_at, ''), COALESCE(o.verified_at, ''),
        COALESCE(o.normalized_path, ''), COALESCE(o.artifact_sha256, ''), COALESCE(o.artifact_body_sha256, ''),
        COALESCE(o.processor, ''), COALESCE(o.processor_version, ''), COALESCE(o.processing_error, ''), COALESCE(o.warnings, '')
        FROM intake_occurrences o JOIN intake_contents c ON c.content_id = o.content_id WHERE o.occurrence_id = ?`, id).Scan(
		&item.OccurrenceID, &item.ContentID, &item.RelativePath, &item.OriginalFilename, &item.Size, &item.Type, &item.Extension,
		&item.FirstSeenAt, &item.LastSeenAt, &item.State, &item.FatherTimeYear, &item.ProcessingState,
		&item.RetentionDays, &item.RetentionState, &item.RetentionDueAt, &item.ChronicleState, &item.ProcessingStartedAt, &item.NormalizedAt,
		&item.VerifiedAt, &item.NormalizedPath, &item.ArtifactSHA256, &item.ArtifactBodySHA256, &item.Processor,
		&item.ProcessorVersion, &item.ProcessingError, &item.Warnings)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return intakeOccurrence{}, fmt.Errorf("unknown Intake occurrence %q", id)
		}
		return intakeOccurrence{}, err
	}
	return item, nil
}

func (d *daemon) updateProcessingFailure(id, state, message string) error {
	_, err := d.db.Exec(`UPDATE intake_occurrences SET processing_state = ?, processing_error = ? WHERE occurrence_id = ?`, state, message, id)
	return err
}

func (d *daemon) occurrenceArtifactPath(occurrence intakeOccurrence) (string, error) {
	root := effectiveRawPath(d.cfg)
	if root == "" {
		return "", errors.New("Raw Vault path is unavailable; configure paths.vault or paths.raw")
	}
	if err := ensureNoSymlinkComponents(root); err != nil {
		return "", err
	}
	relative, err := rawRelativePath(occurrence)
	if err != nil {
		return "", err
	}
	if occurrence.NormalizedPath != "" {
		candidate := occurrence.NormalizedPath
		if !filepath.IsAbs(candidate) {
			candidate = filepath.Join(root, candidate)
		}
		if !pathWithin(root, candidate) {
			return "", errors.New("recorded normalized path escapes Raw Vault")
		}
		if err := ensureNoSymlinkComponents(candidate); err != nil {
			return "", err
		}
		return candidate, nil
	}
	candidate := filepath.Join(root, relative)
	if !pathWithin(root, candidate) {
		return "", errors.New("normalized path escapes Raw Vault")
	}
	if _, err := os.Lstat(candidate); err == nil {
		var owner string
		_ = d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE normalized_path = ? LIMIT 1`, candidate).Scan(&owner)
		if owner == occurrence.OccurrenceID {
			return candidate, nil
		}
		stamp, parseErr := time.Parse(time.RFC3339Nano, occurrence.FirstSeenAt)
		if parseErr != nil {
			return "", parseErr
		}
		stem := strings.TrimSuffix(candidate, filepath.Ext(candidate))
		candidate = fmt.Sprintf("%s--%s%s", stem, stamp.Format("2006-01-02-1504"), filepath.Ext(candidate))
		if _, err := os.Lstat(candidate); err == nil {
			candidate = strings.TrimSuffix(candidate, filepath.Ext(candidate)) + "--" + occurrence.OccurrenceID[len("intake-"):len("intake-")+8] + filepath.Ext(candidate)
		}
	}
	if err := ensureNoSymlinkComponents(candidate); err != nil {
		return "", err
	}
	return candidate, nil
}

func sourceTypeForOccurrence(occurrence intakeOccurrence) string {
	if occurrence.Type != "" && occurrence.Type != "unknown" {
		return occurrence.Type
	}
	return "application/octet-stream"
}

func frontmatterFor(occurrence intakeOccurrence, body []byte, processor, version, normalizedAt string, warnings []string) []byte {
	var b strings.Builder
	fmt.Fprintf(&b, "---\npantheon:\n  kind: raw-source\n  occurrence_id: %s\n  content_id: %s\n  source_path: %s\n  original_filename: %s\n  source_type: %s\n  observed_at: %s\n  normalized_at: %s\n  processor: %s\n  processor_version: %s\n  body_sha256: %s\n", yamlQuote(occurrence.OccurrenceID), yamlQuote(occurrence.ContentID), yamlQuote(occurrence.RelativePath), yamlQuote(occurrence.OriginalFilename), yamlQuote(sourceTypeForOccurrence(occurrence)), yamlQuote(occurrence.FirstSeenAt), yamlQuote(normalizedAt), yamlQuote(processor), yamlQuote(version), yamlQuote(sha256Bytes(body)))
	if len(warnings) > 0 {
		b.WriteString("  warnings:\n")
		for _, warning := range warnings {
			fmt.Fprintf(&b, "    - %s\n", yamlQuote(warning))
		}
	}
	b.WriteString("---\n")
	return []byte(b.String())
}

func yamlQuote(value string) string { return strconv.Quote(value) }

func writeArtifact(path string, artifact []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if err := ensureNoSymlinkComponents(path); err != nil {
		return err
	}
	if info, err := os.Lstat(path); err == nil {
		if info.Mode()&os.ModeSymlink != 0 || !info.Mode().IsRegular() {
			return errors.New("normalized artifact is not a regular file")
		}
		return errors.New("normalized artifact already exists and will not be overwritten")
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pantheon-present-*.tmp")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath)
	if err := tmp.Chmod(0o600); err != nil {
		_ = tmp.Close()
		return err
	}
	if _, err := tmp.Write(artifact); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Sync(); err != nil {
		_ = tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpPath, path)
}

func parseFrontmatter(data []byte) (map[string]string, []byte, error) {
	if !bytes.HasPrefix(data, []byte("---\n")) {
		return nil, nil, errors.New("artifact is missing YAML frontmatter")
	}
	end := bytes.Index(data[4:], []byte("\n---\n"))
	if end < 0 {
		return nil, nil, errors.New("artifact frontmatter is unterminated")
	}
	end += 4
	values := make(map[string]string)
	for _, line := range strings.Split(string(data[4:end]), "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, "") || !strings.Contains(line, ":") {
			continue
		}
		parts := strings.SplitN(line, ":", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if key == "pantheon" || key == "kind" || key == "warnings" || strings.HasPrefix(line, "-") {
			continue
		}
		if unquoted, err := strconv.Unquote(value); err == nil {
			value = unquoted
		}
		values[key] = value
	}
	body := data[end+len("\n---\n"):]
	return values, body, nil
}

func verifyArtifact(path string, occurrence intakeOccurrence, expectedBodyHash string) (string, string, error) {
	if path == "" || !pathWithin(filepath.Dir(path), path) {
		return "", "", errors.New("invalid artifact path")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return "", "", err
	}
	metadata, body, err := parseFrontmatter(data)
	if err != nil {
		return "", "", err
	}
	checks := map[string]string{"occurrence_id": occurrence.OccurrenceID, "content_id": occurrence.ContentID, "source_path": occurrence.RelativePath}
	for key, expected := range checks {
		if metadata[key] != expected {
			return "", "", fmt.Errorf("artifact provenance %s does not match occurrence", key)
		}
	}
	bodyHash := sha256Bytes(body)
	if metadata["body_sha256"] != bodyHash || (expectedBodyHash != "" && expectedBodyHash != bodyHash) {
		return "", "", errors.New("normalized artifact body hash does not verify")
	}
	return sha256Bytes(data), bodyHash, nil
}

func (d *daemon) normalizeOccurrence(ctx context.Context, sourcePath string, occurrence intakeOccurrence) (normalizedArtifact, error) {
	ext := strings.ToLower(filepath.Ext(occurrence.RelativePath))
	switch ext {
	case ".md", ".txt":
		body, err := os.ReadFile(sourcePath)
		if err != nil {
			return normalizedArtifact{}, err
		}
		return normalizedArtifact{Body: body, Processor: "pantheon-text", ProcessorVersion: "1", Warnings: nil}, nil
	case ".pdf":
		return normalizePDF(ctx, sourcePath)
	case ".docx":
		return normalizeDOCX(sourcePath)
	case ".wav", ".mp3", ".m4a", ".flac", ".ogg", ".mp4", ".mov":
		return d.normalizeAudio(ctx, sourcePath, occurrence, ext == ".mp4" || ext == ".mov")
	default:
		return normalizedArtifact{}, fmt.Errorf("unsupported source type %q", ext)
	}
}

func normalizePDF(ctx context.Context, sourcePath string) (normalizedArtifact, error) {
	binary, err := exec.LookPath("pdftotext")
	if err != nil {
		return normalizedArtifact{}, fmt.Errorf("pdftotext is not available")
	}
	cmd := exec.CommandContext(ctx, binary, "-layout", sourcePath, "-")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	output, err := cmd.Output()
	if err != nil {
		return normalizedArtifact{}, fmt.Errorf("PDF text extraction failed: %s", strings.TrimSpace(stderr.String()))
	}
	if len(bytes.TrimSpace(output)) == 0 {
		return normalizedArtifact{}, errors.New("PDF contains no extractable text; OCR is not enabled")
	}
	page := 1
	var body strings.Builder
	for _, part := range bytes.Split(output, []byte("\f")) {
		if len(bytes.TrimSpace(part)) == 0 {
			continue
		}
		fmt.Fprintf(&body, "<!-- source-page: %d -->\n\n%s\n\n", page, strings.TrimSpace(string(part)))
		page++
	}
	return normalizedArtifact{Body: []byte(body.String()), Processor: "pdftotext", ProcessorVersion: filepath.Base(binary), Warnings: nil}, nil
}

type docxParagraph struct {
	Style string
	Text  string
}

type docxTable struct{ Rows [][]string }

func normalizeDOCX(sourcePath string) (normalizedArtifact, error) {
	reader, err := zip.OpenReader(sourcePath)
	if err != nil {
		return normalizedArtifact{}, fmt.Errorf("open DOCX: %w", err)
	}
	defer reader.Close()
	var document io.ReadCloser
	for _, file := range reader.File {
		if file.Name == "word/document.xml" {
			document, err = file.Open()
			break
		}
	}
	if err != nil || document == nil {
		return normalizedArtifact{}, errors.New("DOCX has no word/document.xml")
	}
	defer document.Close()
	relationships := map[string]string{}
	for _, file := range reader.File {
		if file.Name != "word/_rels/document.xml.rels" {
			continue
		}
		rel, openErr := file.Open()
		if openErr != nil {
			continue
		}
		relationships = parseDOCXRelationships(rel)
		_ = rel.Close()
	}
	paragraphs, tables, err := parseDOCX(document, relationships)
	if err != nil {
		return normalizedArtifact{}, err
	}
	var body strings.Builder
	for _, paragraph := range paragraphs {
		line := renderDOCXParagraph(paragraph)
		if line != "" {
			body.WriteString(line)
			body.WriteString("\n\n")
		}
	}
	for _, table := range tables {
		if len(table.Rows) == 0 {
			continue
		}
		for rowIndex, row := range table.Rows {
			body.WriteString("| ")
			body.WriteString(strings.Join(escapeTableCells(row), " | "))
			body.WriteString(" |\n")
			if rowIndex == 0 {
				body.WriteString("| ")
				separators := make([]string, len(row))
				for i := range separators {
					separators[i] = "---"
				}
				body.WriteString(strings.Join(separators, " | "))
				body.WriteString(" |\n")
			}
		}
		body.WriteString("\n")
	}
	if strings.TrimSpace(body.String()) == "" {
		return normalizedArtifact{}, errors.New("DOCX contained no recoverable text")
	}
	return normalizedArtifact{Body: []byte(body.String()), Processor: "pantheon-docx", ProcessorVersion: "1", Warnings: nil}, nil
}

func parseDOCXRelationships(reader io.Reader) map[string]string {
	decoder := xml.NewDecoder(reader)
	result := map[string]string{}
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) || err != nil {
			break
		}
		if start, ok := token.(xml.StartElement); ok && start.Name.Local == "Relationship" {
			var id, target string
			for _, attr := range start.Attr {
				switch attr.Name.Local {
				case "Id":
					id = attr.Value
				case "Target":
					target = attr.Value
				}
			}
			if id != "" && target != "" {
				result[id] = target
			}
		}
	}
	return result
}

func parseDOCX(reader io.Reader, relationships map[string]string) ([]docxParagraph, []docxTable, error) {
	decoder := xml.NewDecoder(reader)
	var paragraphs []docxParagraph
	var tables []docxTable
	var current *docxParagraph
	var currentCell *strings.Builder
	var currentRow []string
	var currentTable *docxTable
	var hyperlink bool
	var hyperlinkText strings.Builder
	var hyperlinkTarget string
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, nil, err
		}
		switch item := token.(type) {
		case xml.StartElement:
			switch item.Name.Local {
			case "p":
				current = &docxParagraph{}
			case "pStyle":
				if current != nil {
					for _, attr := range item.Attr {
						if attr.Name.Local == "val" {
							current.Style = attr.Value
						}
					}
				}
			case "tbl":
				currentTable = &docxTable{}
			case "tr":
				currentRow = nil
			case "tc":
				currentCell = &strings.Builder{}
			case "hyperlink":
				hyperlink = true
				hyperlinkText.Reset()
				hyperlinkTarget = ""
				for _, attr := range item.Attr {
					if attr.Name.Local == "id" {
						hyperlinkTarget = relationships[attr.Value]
						if hyperlinkTarget == "" {
							hyperlinkTarget = attr.Value
						}
					}
				}
			case "tab":
				if current != nil {
					current.Text += "\t"
				}
			case "br", "cr":
				if current != nil {
					current.Text += "\n"
				}
			}
		case xml.CharData:
			if current != nil {
				text := string(item)
				current.Text += text
				if hyperlink {
					hyperlinkText.WriteString(text)
				}
			}
		case xml.EndElement:
			switch item.Name.Local {
			case "hyperlink":
				hyperlink = false
				if current != nil && hyperlinkTarget != "" && hyperlinkText.Len() > 0 {
					current.Text = strings.TrimSuffix(current.Text, hyperlinkText.String()) + "[" + hyperlinkText.String() + "](" + hyperlinkTarget + ")"
				}
			case "p":
				if current != nil {
					text := strings.TrimRight(current.Text, "\n")
					if currentCell != nil {
						currentCell.WriteString(text)
						currentCell.WriteString(" ")
					} else {
						paragraphs = append(paragraphs, *current)
					}
				}
				current = nil
			case "tc":
				if currentCell != nil {
					currentRow = append(currentRow, strings.TrimSpace(currentCell.String()))
				}
				currentCell = nil
			case "tr":
				if currentTable != nil {
					currentTable.Rows = append(currentTable.Rows, currentRow)
				}
				currentRow = nil
			case "tbl":
				if currentTable != nil {
					tables = append(tables, *currentTable)
				}
				currentTable = nil
			}
		}
	}
	return paragraphs, tables, nil
}

func renderDOCXParagraph(paragraph docxParagraph) string {
	text := strings.TrimSpace(paragraph.Text)
	if text == "" {
		return ""
	}
	style := strings.ToLower(paragraph.Style)
	if strings.Contains(style, "heading") {
		for _, n := range []string{"1", "2", "3", "4", "5", "6"} {
			if strings.Contains(style, n) {
				return strings.Repeat("#", int(n[0]-'0')) + " " + text
			}
		}
		return "# " + text
	}
	if strings.Contains(style, "bullet") {
		return "- " + text
	}
	if strings.Contains(style, "number") {
		return "1. " + text
	}
	return text
}

func escapeTableCells(row []string) []string {
	result := make([]string, len(row))
	for i, cell := range row {
		result[i] = strings.ReplaceAll(cell, "|", "\\|")
	}
	return result
}

func (d *daemon) normalizeAudio(ctx context.Context, sourcePath string, occurrence intakeOccurrence, video bool) (normalizedArtifact, error) {
	if !d.cfg.WhisperEnabled {
		return normalizedArtifact{}, errors.New("local Whisper processor is disabled")
	}
	binary, err := configuredCommand(d.cfg.WhisperBinary, "whisper")
	if err != nil {
		return normalizedArtifact{}, errors.New("local Whisper processor is not configured")
	}
	workDir, err := os.MkdirTemp("", "pantheon-present-")
	if err != nil {
		return normalizedArtifact{}, err
	}
	defer os.RemoveAll(workDir)
	inputPath := sourcePath
	// The installed local processor is whisper.cpp's whisper-cli, which only
	// accepts decoded audio (wav/flac/mp3/ogg) and uses its own CLI contract.
	// Keep the older Python-Whisper argument contract available for configured
	// test/dev binaries that are not named whisper-cli.
	whisperCLI := strings.Contains(strings.ToLower(filepath.Base(binary)), "whisper-cli")
	if video || (whisperCLI && strings.ToLower(filepath.Ext(sourcePath)) != ".wav") {
		converter, convErr := configuredCommand(d.cfg.MediaConverter, "ffmpeg")
		if convErr != nil {
			return normalizedArtifact{}, errors.New("local media converter is not configured")
		}
		converted := filepath.Join(workDir, "audio.wav")
		cmd := exec.CommandContext(ctx, converter, "-y", "-i", sourcePath, "-vn", "-ar", "16000", "-ac", "1", "-c:a", "pcm_s16le", converted)
		if output, runErr := cmd.CombinedOutput(); runErr != nil {
			return normalizedArtifact{}, fmt.Errorf("media conversion failed: %s", strings.TrimSpace(string(output)))
		}
		inputPath = converted
	}
	outputBase := filepath.Join(workDir, strings.TrimSuffix(filepath.Base(inputPath), filepath.Ext(inputPath)))
	var args []string
	if whisperCLI {
		args = []string{"-m", d.cfg.WhisperModel, "-f", inputPath, "--output-txt", "--output-file", outputBase, "--no-prints", "--no-timestamps"}
		if vad := strings.TrimSpace(d.cfg.WhisperVADModel); vad != "" {
			if info, statErr := os.Stat(vad); statErr != nil || !info.Mode().IsRegular() {
				return normalizedArtifact{}, errors.New("configured Whisper VAD model is unavailable")
			}
			args = append(args, "--vad", "--vad-model", vad, "--no-fallback", "--temperature", "0", "--max-context", "0", "--split-on-word", "--no-speech-thold", "0.60", "--vad-max-speech-duration-s", "30")
		}
		if lang := strings.TrimSpace(d.cfg.WhisperLanguage); lang != "" {
			args = append(args, "-l", lang)
		}
	} else {
		args = []string{inputPath, "--output_format", "txt", "--output_dir", workDir}
		if d.cfg.WhisperModel != "" {
			args = append(args, "--model", d.cfg.WhisperModel)
		}
		if lang := strings.TrimSpace(d.cfg.WhisperLanguage); lang != "" && lang != "auto" {
			args = append(args, "--language", lang)
		}
	}
	cmd := exec.CommandContext(ctx, binary, args...)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	stdout, runErr := cmd.Output()
	if runErr != nil {
		return normalizedArtifact{}, fmt.Errorf("Whisper failed: %s", strings.TrimSpace(stderr.String()))
	}
	outputPath := outputBase + ".txt"
	transcript, readErr := os.ReadFile(outputPath)
	if readErr != nil {
		transcript = stdout
	}
	if len(bytes.TrimSpace(transcript)) == 0 {
		return normalizedArtifact{}, errors.New("Whisper produced no transcript")
	}
	if repeated, count := transcriptRepeatedLine(transcript); count >= 25 {
		return normalizedArtifact{}, fmt.Errorf("Whisper transcript quality gate rejected repeated line %q (%d occurrences)", repeated, count)
	}
	title := strings.TrimSuffix(occurrence.OriginalFilename, filepath.Ext(occurrence.OriginalFilename))
	body := fmt.Sprintf("# %s\n\n**Source:** `%s`\n\n## Transcript\n\n%s", title, occurrence.RelativePath, strings.TrimSpace(string(transcript)))
	return normalizedArtifact{Body: []byte(body + "\n"), Processor: "pantheon-whisper", ProcessorVersion: filepath.Base(binary), Warnings: nil}, nil
}

func transcriptRepeatedLine(data []byte) (string, int) {
	counts := map[string]int{}
	var repeated string
	max := 0
	for _, line := range strings.Split(string(data), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		key := strings.ToLower(strings.Join(strings.Fields(line), " "))
		counts[key]++
		if counts[key] > max {
			max, repeated = counts[key], line
		}
	}
	return repeated, max
}

func configuredCommand(configured, fallback string) (string, error) {
	if configured != "" {
		info, err := os.Stat(configured)
		if err != nil || !info.Mode().IsRegular() || info.Mode()&0111 == 0 {
			return "", errors.New("configured command is unavailable")
		}
		return configured, nil
	}
	return exec.LookPath(fallback)
}

func (d *daemon) verifyAndArm(occurrence intakeOccurrence, artifactPath string, artifact normalizedArtifact, normalizedAt, request string) (presentResult, error) {
	frontmatter := frontmatterFor(occurrence, artifact.Body, artifact.Processor, artifact.ProcessorVersion, normalizedAt, artifact.Warnings)
	data := append(frontmatter, artifact.Body...)
	if err := writeArtifact(artifactPath, data); err != nil {
		return presentResult{}, err
	}
	artifactHash, bodyHash, err := verifyArtifact(artifactPath, occurrence, sha256Bytes(artifact.Body))
	if err != nil {
		return presentResult{}, err
	}
	due := time.Time{}
	verifiedTime, err := time.Parse(time.RFC3339Nano, normalizedAt)
	if err != nil {
		return presentResult{}, err
	}
	due = verifiedTime.AddDate(0, 0, occurrence.RetentionDays)
	dueAt := due.Format(time.RFC3339Nano)
	missiveID, err := newLocalID("missive-")
	if err != nil {
		return presentResult{}, err
	}
	if _, err := d.db.Exec(`UPDATE intake_occurrences SET processing_state = ?, normalized_at = ?, verified_at = ?, normalized_path = ?, artifact_sha256 = ?, artifact_body_sha256 = ?, processor = ?, processor_version = ?, processing_error = '', warnings = ?, retention_state = ?, retention_eligible_at = ?, retention_due_at = ? WHERE occurrence_id = ?`, processingVerified, normalizedAt, normalizedAt, artifactPath, artifactHash, bodyHash, artifact.Processor, artifact.ProcessorVersion, strings.Join(artifact.Warnings, "\n"), retentionArmed, normalizedAt, dueAt, occurrence.OccurrenceID); err != nil {
		return presentResult{}, err
	}
	reasoningState := "not_requested"
	if strings.TrimSpace(request) != "" {
		reasoningState = "pending_apollo"
	}
	if _, err := d.db.Exec(`INSERT INTO present_missives (missive_id, created_at, original_request, occurrence_id, content_id, source_path, source_type, normalized_path, processor, processor_version, warnings, verification_state, verified_at, retention_state, retention_due_at, reasoning_state) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 'verified', ?, ?, ?, ?)`, missiveID, normalizedAt, request, occurrence.OccurrenceID, occurrence.ContentID, occurrence.RelativePath, sourceTypeForOccurrence(occurrence), artifactPath, artifact.Processor, artifact.ProcessorVersion, strings.Join(artifact.Warnings, "\n"), normalizedAt, retentionArmed, dueAt, reasoningState); err != nil {
		return presentResult{}, err
	}
	return presentResult{OccurrenceID: occurrence.OccurrenceID, SourcePath: occurrence.RelativePath, SourceType: sourceTypeForOccurrence(occurrence), NormalizedPath: artifactPath, Processor: artifact.Processor, State: processingVerified, Verification: "complete", RetentionState: retentionArmed, RetentionDueAt: dueAt, MissiveID: missiveID, Warnings: artifact.Warnings}, nil
}

func (d *daemon) processOccurrence(ctx context.Context, id, request string) (presentResult, error) {
	occurrence, err := d.loadIntakeOccurrence(id)
	if err != nil {
		return presentResult{}, err
	}
	sourcePath, err := sourcePathForOccurrence(d.cfg, occurrence)
	if err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, err
	}
	if _, err := os.Lstat(sourcePath); err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, fmt.Errorf("source unavailable: %w", err)
	}
	if err := ensureNoSymlinkComponents(sourcePath); err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, err
	}
	info, err := os.Lstat(sourcePath)
	if err != nil || !info.Mode().IsRegular() {
		message := "source is not an ordinary regular file"
		if err != nil {
			message = err.Error()
		}
		_ = d.updateProcessingFailure(id, processingFailed, message)
		return presentResult{}, errors.New(message)
	}
	hashed, err := hashStableIntakeFile(sourcePath)
	if err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, err
	}
	if hashed.ContentID != occurrence.ContentID {
		return presentResult{}, errors.New("source changed; occurrence identity is stale and was not processed")
	}
	if occurrence.ProcessingState == processingVerified && occurrence.NormalizedPath != "" {
		if _, _, verifyErr := verifyArtifact(occurrence.NormalizedPath, occurrence, occurrence.ArtifactBodySHA256); verifyErr == nil {
			return presentResult{OccurrenceID: id, SourcePath: occurrence.RelativePath, SourceType: sourceTypeForOccurrence(occurrence), NormalizedPath: occurrence.NormalizedPath, Processor: occurrence.Processor, State: processingVerified, Verification: "complete", RetentionState: occurrence.RetentionState, RetentionDueAt: occurrence.RetentionDueAt, AlreadyVerified: true}, nil
		}
		return presentResult{}, errors.New("existing normalized artifact was modified or cannot be verified; it was preserved")
	}
	loc, err := fatherTimeLocation(d.cfg)
	if err != nil {
		return presentResult{}, err
	}
	started := time.Now().In(loc).Format(time.RFC3339Nano)
	if _, err := d.db.Exec(`UPDATE intake_occurrences SET processing_state = ?, processing_started_at = ?, processing_error = '' WHERE occurrence_id = ?`, processingProcessing, started, id); err != nil {
		return presentResult{}, err
	}
	artifact, normalizeErr := d.normalizeOccurrence(ctx, sourcePath, occurrence)
	if normalizeErr != nil {
		state := processingFailed
		if strings.Contains(normalizeErr.Error(), "unsupported source type") || strings.Contains(normalizeErr.Error(), "no extractable text") {
			state = processingBlockedUnsupported
		}
		if strings.Contains(normalizeErr.Error(), "not configured") || strings.Contains(normalizeErr.Error(), "not available") {
			state = processingBlockedProcessor
		}
		_ = d.updateProcessingFailure(id, state, normalizeErr.Error())
		return presentResult{OccurrenceID: id, SourcePath: occurrence.RelativePath, SourceType: sourceTypeForOccurrence(occurrence), State: state, Verification: "not_verified", RetentionState: occurrence.RetentionState, Error: normalizeErr.Error()}, nil
	}
	artifactPath, err := d.occurrenceArtifactPath(occurrence)
	if err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, err
	}
	if _, statErr := os.Lstat(artifactPath); statErr == nil && occurrence.NormalizedPath == "" {
		artifactPath, err = d.collisionArtifactPath(artifactPath, occurrence)
		if err != nil {
			_ = d.updateProcessingFailure(id, processingFailed, err.Error())
			return presentResult{}, err
		}
	}
	normalizedAt := time.Now().In(loc).Format(time.RFC3339Nano)
	result, err := d.verifyAndArm(occurrence, artifactPath, artifact, normalizedAt, request)
	if err != nil {
		_ = d.updateProcessingFailure(id, processingFailed, err.Error())
		return presentResult{}, err
	}
	return result, nil
}

func (d *daemon) collisionArtifactPath(path string, occurrence intakeOccurrence) (string, error) {
	stamp, err := time.Parse(time.RFC3339Nano, occurrence.FirstSeenAt)
	if err != nil {
		return "", err
	}
	ext := filepath.Ext(path)
	base := strings.TrimSuffix(path, ext)
	candidate := fmt.Sprintf("%s--%s%s", base, stamp.Format("2006-01-02-1504"), ext)
	if _, err := os.Lstat(candidate); errors.Is(err, os.ErrNotExist) {
		return candidate, nil
	}
	return fmt.Sprintf("%s--%s%s", strings.TrimSuffix(candidate, ext), occurrence.OccurrenceID[len("intake-"):len("intake-")+8], ext), nil
}

func (d *daemon) processNew(ctx context.Context, request string) ([]presentResult, error) {
	if _, err := d.runIntakeScan(); err != nil {
		return nil, err
	}
	rows, err := d.db.Query(`SELECT occurrence_id FROM intake_occurrences WHERE state = ? AND processing_state = ? ORDER BY first_seen_at`, occurrencePresent, processingUnprocessed)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var ids []string
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}
	results := make([]presentResult, 0, len(ids))
	for _, id := range ids {
		result, processErr := d.processOccurrence(ctx, id, request)
		if processErr != nil {
			continue
		}
		results = append(results, result)
	}
	return results, nil
}

func (d *daemon) resolvePresentOccurrence(request string) ([]intakeOccurrence, error) {
	lower := strings.ToLower(request)
	pathFilter, extFilter := "", ""
	if strings.Contains(lower, "pdf") {
		extFilter = ".pdf"
	}
	if strings.Contains(lower, "recording") || strings.Contains(lower, "audio") || strings.Contains(lower, "video") {
		pathFilter = ""
	}
	all, err := d.queryIntakeOccurrences(100, pathFilter, extFilter, "")
	if err != nil {
		return nil, err
	}
	var candidates []intakeOccurrence
	for _, item := range all {
		if item.State != occurrencePresent {
			continue
		}
		ext := strings.ToLower(item.Extension)
		if (strings.Contains(lower, "recording") || strings.Contains(lower, "audio") || strings.Contains(lower, "video")) && !isRecordingExtension(ext) {
			continue
		}
		if strings.Contains(lower, "document") && !isDocumentExtension(ext) {
			continue
		}
		candidates = append(candidates, item)
	}
	var named []intakeOccurrence
	for _, item := range candidates {
		if strings.Contains(lower, strings.ToLower(item.RelativePath)) || strings.Contains(lower, strings.ToLower(item.OriginalFilename)) {
			named = append(named, item)
		}
	}
	if len(named) > 0 {
		candidates = named
	}
	if len(candidates) > 0 && strings.Contains(lower, "latest") {
		top := candidates[0]
		var ties []intakeOccurrence
		for _, item := range candidates {
			if item.FirstSeenAt == top.FirstSeenAt {
				ties = append(ties, item)
			}
		}
		if len(ties) > 1 {
			return ties, nil
		}
		return []intakeOccurrence{top}, nil
	}
	if len(candidates) > 0 {
		return []intakeOccurrence{candidates[0]}, nil
	}
	return nil, errors.New("no matching present Intake occurrence found")
}

func isRecordingExtension(ext string) bool {
	switch ext {
	case ".wav", ".mp3", ".m4a", ".flac", ".ogg", ".mp4", ".mov":
		return true
	}
	return false
}
func isDocumentExtension(ext string) bool {
	switch ext {
	case ".pdf", ".docx", ".txt", ".md":
		return true
	}
	return false
}

func (d *daemon) presentInspection() (map[string]any, error) {
	var pending, verified, blocked, failed int
	for state, destination := range map[string]*int{processingUnprocessed: &pending, processingVerified: &verified, processingBlockedUnsupported: &blocked, processingBlockedProcessor: &blocked, processingFailed: &failed} {
		var count int
		if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences WHERE processing_state = ?`, state).Scan(&count); err != nil {
			return nil, err
		}
		*destination += count
	}
	rows, err := d.db.Query(`SELECT missive_id, created_at, occurrence_id, source_path, normalized_path, processor, verification_state, retention_state, retention_due_at, reasoning_state FROM present_missives ORDER BY created_at DESC LIMIT 20`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	type missive struct {
		ID             string `json:"missive_id"`
		CreatedAt      string `json:"created_at"`
		OccurrenceID   string `json:"occurrence_id"`
		SourcePath     string `json:"source_path"`
		NormalizedPath string `json:"normalized_path"`
		Processor      string `json:"processor"`
		Verification   string `json:"verification_state"`
		Retention      string `json:"retention_state"`
		DueAt          string `json:"retention_due_at,omitempty"`
		Reasoning      string `json:"reasoning_state"`
	}
	var missives []missive
	for rows.Next() {
		var item missive
		if err := rows.Scan(&item.ID, &item.CreatedAt, &item.OccurrenceID, &item.SourcePath, &item.NormalizedPath, &item.Processor, &item.Verification, &item.Retention, &item.DueAt, &item.Reasoning); err != nil {
			return nil, err
		}
		missives = append(missives, item)
	}
	return map[string]any{"raw_path": effectiveRawPath(d.cfg), "pending": pending, "verified": verified, "blocked": blocked, "failed": failed, "missives": missives}, rows.Err()
}

func (d *daemon) processorInspection() map[string]any {
	text := processorStatus{Name: "text", State: "ready", Detail: "Markdown and plain text"}
	pdf := processorStatus{Name: "pdf", State: "missing", Detail: "pdftotext"}
	if binary, err := exec.LookPath("pdftotext"); err == nil {
		pdf.State, pdf.Binary = "ready", binary
	}
	docx := processorStatus{Name: "docx", State: "ready", Detail: "local DOCX XML extraction"}
	whisper := processorStatus{Name: "whisper", State: "disabled", Binary: d.cfg.WhisperBinary, Model: d.cfg.WhisperModel, Language: d.cfg.WhisperLanguage}
	if d.cfg.WhisperEnabled {
		whisper.State = "missing"
		if binary, err := configuredCommand(d.cfg.WhisperBinary, "whisper"); err == nil {
			whisper.State, whisper.Binary = "ready", binary
		} else {
			whisper.Detail = "configure a local Whisper binary"
		}
	}
	media := processorStatus{Name: "media conversion", State: "missing", Binary: d.cfg.MediaConverter}
	if binary, err := configuredCommand(d.cfg.MediaConverter, "ffmpeg"); err == nil {
		media.State, media.Binary = "ready", binary
	}
	return map[string]any{"text": text, "pdf": pdf, "docx": docx, "whisper": whisper, "media_conversion": media}
}

func presentResultText(result presentResult, request string) string {
	if result.Error != "" {
		return fmt.Sprintf("Present could not complete %s: %s\nSource remains untouched. Retention has not been armed.", result.SourcePath, result.Error)
	}
	if result.AlreadyVerified {
		return fmt.Sprintf("Already verified: %s\nNormalized: %s\nRetention: %s", result.SourcePath, result.NormalizedPath, result.RetentionState)
	}
	text := fmt.Sprintf("Present prepared the source.\n\nSource:\n  %s\n\nNormalized:\n  %s\n\nVerification:\n  %s\nRetention:\n  %s", result.SourcePath, result.NormalizedPath, result.Verification, result.RetentionState)
	if result.RetentionDueAt != "" {
		text += "\nDue date:\n  " + result.RetentionDueAt
	}
	if len(result.Warnings) > 0 {
		text += "\nWarnings:\n  " + strings.Join(result.Warnings, "\n  ")
	}
	if strings.TrimSpace(request) != "" {
		text += "\n\nRequested reasoning:\n  " + request + "\n\nReasoning is awaiting Apollo."
	}
	return text
}

func (d *daemon) handlePresent(w http.ResponseWriter, r *http.Request) {
	var request presentRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if !request.ProcessNew && strings.TrimSpace(request.OccurrenceID) == "" && strings.TrimSpace(request.Request) == "" {
		writeError(w, http.StatusBadRequest, errors.New("present requires a request, --occurrence ID, or --new"))
		return
	}
	if d.blockIfPaused(w, "POST /v1/present", request.Request, "present") {
		return
	}
	started := time.Now()
	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Minute)
	defer cancel()
	response := presentResponse{Request: request.Request}
	if request.ProcessNew {
		results, err := d.processNew(ctx, request.Request)
		if err != nil {
			id := d.audit("POST /v1/present", request.Request, "present", "", "failed", "", err.Error(), time.Since(started))
			writeJSON(w, http.StatusInternalServerError, apiError{fmt.Sprintf("Present failed (audit %s): %v", id, err)})
			return
		}
		response.Processed = results
		response.Result = fmt.Sprintf("Present processed %d new occurrence(s).", len(results))
		for _, result := range results {
			if result.Error != "" {
				response.Result += "\n" + result.Error
			}
		}
		d.audit("POST /v1/present", request.Request, "present", "", "completed", response.Result, "", time.Since(started))
		writeJSON(w, http.StatusOK, response)
		return
	}
	if _, err := d.runIntakeScan(); err != nil {
		id := d.audit("POST /v1/present", request.Request, "present", "", "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusInternalServerError, apiError{fmt.Sprintf("Present reconciliation failed (audit %s): %v", id, err)})
		return
	}
	id := request.OccurrenceID
	var candidates []intakeOccurrence
	if id != "" {
		occurrence, err := d.loadIntakeOccurrence(id)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		candidates = []intakeOccurrence{occurrence}
	} else {
		var err error
		candidates, err = d.resolvePresentOccurrence(request.Request)
		if err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
	}
	if len(candidates) > 1 {
		response.Resolution = "ambiguous"
		for _, candidate := range candidates {
			response.Candidates = append(response.Candidates, presentCandidate{OccurrenceID: candidate.OccurrenceID, Path: candidate.RelativePath, Type: candidate.Type, ObservedAt: candidate.FirstSeenAt})
		}
		response.Result = "I found multiple equally plausible Intake occurrences; use a filename or occurrence ID."
		d.audit("POST /v1/present", request.Request, "present", "", "ambiguous", response.Result, "", time.Since(started))
		writeJSON(w, http.StatusConflict, response)
		return
	}
	result, err := d.processOccurrence(ctx, candidates[0].OccurrenceID, request.Request)
	if err != nil {
		id := d.audit("POST /v1/present", request.Request, "present", "", "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusInternalServerError, apiError{fmt.Sprintf("Present failed (audit %s): %v", id, err)})
		return
	}
	response.Occurrence = &result
	response.Result = presentResultText(result, request.Request)
	d.audit("POST /v1/present", request.Request, "present", result.Processor, "completed", response.Result, result.Error, time.Since(started))
	writeJSON(w, http.StatusOK, response)
}

func (d *daemon) handleShowPresent(w http.ResponseWriter, r *http.Request) {
	inspection, err := d.presentInspection()
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, inspection)
}

func (d *daemon) handleShowProcessors(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, d.processorInspection())
}
