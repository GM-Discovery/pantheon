package pantheon

import (
	"archive/zip"
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestKernel10TextPresentVerificationArmsRetentionAndIsIdempotent(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	source := filepath.Join(cfg.IntakePath, "Kernel 9 Test", "original.md")
	body := []byte("# Kernel 9 Test\n\nExact source body.\n")
	if err := os.MkdirAll(filepath.Dir(source), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, body, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var occurrenceID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE relative_path = 'Kernel 9 Test/original.md'`).Scan(&occurrenceID); err != nil {
		t.Fatal(err)
	}
	before, err := d.loadIntakeOccurrence(occurrenceID)
	if err != nil {
		t.Fatal(err)
	}
	if before.RetentionState != retentionAwaitingMarkdown || before.RetentionDueAt != "" {
		t.Fatalf("initial retention = %+v", before)
	}
	result, err := d.processOccurrence(context.Background(), occurrenceID, "I need a brief summary later")
	if err != nil {
		t.Fatal(err)
	}
	if result.State != processingVerified || result.Verification != "complete" || result.RetentionState != retentionArmed {
		t.Fatalf("present result = %+v", result)
	}
	artifact, err := os.ReadFile(result.NormalizedPath)
	if err != nil {
		t.Fatal(err)
	}
	_, artifactBody, err := parseFrontmatter(artifact)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(artifactBody, body) {
		t.Fatalf("artifact body changed: %q", artifactBody)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatalf("source was not preserved: %v", err)
	}
	verified, err := d.loadIntakeOccurrence(occurrenceID)
	if err != nil {
		t.Fatal(err)
	}
	verifiedAt, err := time.Parse(time.RFC3339Nano, verified.VerifiedAt)
	if err != nil {
		t.Fatal(err)
	}
	dueAt, err := time.Parse(time.RFC3339Nano, verified.RetentionDueAt)
	if err != nil || !dueAt.Equal(verifiedAt.AddDate(0, 0, 30)) {
		t.Fatalf("retention due=%q verified=%q err=%v", verified.RetentionDueAt, verified.VerifiedAt, err)
	}
	var missives int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM present_missives`).Scan(&missives); err != nil {
		t.Fatal(err)
	}
	second, err := d.processOccurrence(context.Background(), occurrenceID, "")
	if err != nil {
		t.Fatal(err)
	}
	if !second.AlreadyVerified || second.NormalizedPath != result.NormalizedPath {
		t.Fatalf("second result = %+v", second)
	}
	var afterMissives int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM present_missives`).Scan(&afterMissives); err != nil {
		t.Fatal(err)
	}
	if afterMissives != missives {
		t.Fatalf("idempotent processing created missive: before=%d after=%d", missives, afterMissives)
	}
}

func TestKernel10UnsupportedImageRemainsProtected(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(cfg.IntakePath, "photo.png")
	if err := os.WriteFile(source, []byte("not interpreted"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	result, err := d.processOccurrence(context.Background(), id, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.State != processingBlockedUnsupported || result.RetentionState != retentionAwaitingMarkdown {
		t.Fatalf("unsupported result = %+v", result)
	}
	if _, err := os.Stat(source); err != nil {
		t.Fatal(err)
	}
	var normalized sql.NullString
	if err := d.db.QueryRow(`SELECT normalized_path FROM intake_occurrences WHERE occurrence_id = ?`, id).Scan(&normalized); err != nil {
		t.Fatal(err)
	}
	if normalized.Valid {
		t.Fatalf("unsupported source received artifact: %q", normalized.String)
	}
}

func TestKernel10PresentHandlerPreservesRequestAndSupportsInspection(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(presentRequest{Request: "I need a brief summary and all action items from the latest file."})
	req := httptest.NewRequest(http.MethodPost, "/v1/present", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	d.handlePresent(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("present status=%d body=%s", rec.Code, rec.Body.String())
	}
	var response presentResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &response); err != nil {
		t.Fatal(err)
	}
	if response.Occurrence == nil || !strings.Contains(response.Result, "Reasoning is awaiting Apollo") {
		t.Fatalf("present response=%+v", response)
	}
	var originalRequest, reasoning string
	if err := d.db.QueryRow(`SELECT original_request, reasoning_state FROM present_missives WHERE missive_id = ?`, response.Occurrence.MissiveID).Scan(&originalRequest, &reasoning); err != nil {
		t.Fatal(err)
	}
	if originalRequest != "I need a brief summary and all action items from the latest file." || reasoning != "pending_apollo" {
		t.Fatalf("missive request=%q reasoning=%q", originalRequest, reasoning)
	}
	if _, err := d.presentInspection(); err != nil {
		t.Fatal(err)
	}
	if len(d.processorInspection()) == 0 {
		t.Fatal("processor inspection is empty")
	}
}

func TestKernel10PauseBlocksPresentWithoutReplay(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "notes.txt"), []byte("notes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := d.setPaused(true); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(presentRequest{ProcessNew: true})
	req := httptest.NewRequest(http.MethodPost, "/v1/present", bytes.NewReader(payload))
	rec := httptest.NewRecorder()
	d.handlePresent(rec, req)
	if rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("paused Present status=%d body=%s", rec.Code, rec.Body.String())
	}
	if err := d.setPaused(false); err != nil {
		t.Fatal(err)
	}
	var occurrences int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM intake_occurrences`).Scan(&occurrences); err != nil {
		t.Fatal(err)
	}
	if occurrences != 0 {
		t.Fatalf("paused Present performed work: %d occurrences", occurrences)
	}
}

func TestKernel10CanonicalConfigurationResolvesRawAndProcessorFields(t *testing.T) {
	data := []byte(`{"schema_version":1,"paths":{"vault":"/tmp/example-vault"},"models":{"apollo":{"model":"local","endpoint":"http://127.0.0.1:11434/api/chat"}},"ingestion":{"whisper":{"enabled":false,"binary":"/tmp/whisper","model":"tiny","language":"es"},"media":{"converter":"/tmp/ffmpeg"}}}`)
	resolution, err := parseConfigBytes("test", data)
	if err != nil {
		t.Fatal(err)
	}
	if resolution.Config.RawPath != filepath.Join("/tmp/example-vault", "80 Pantheon Raw") || resolution.Config.WhisperEnabled || resolution.Config.WhisperLanguage != "es" || resolution.Config.WhisperModel != "tiny" {
		t.Fatalf("resolved Kernel 10 config=%+v", resolution.Config)
	}
}

func TestKernel10DOCXPreservesBasicStructure(t *testing.T) {
	path := filepath.Join(t.TempDir(), "source.docx")
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	archive := zip.NewWriter(file)
	document, err := archive.Create("word/document.xml")
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.WriteString(document, `<w:document xmlns:w="wordprocessingml/2006/main"><w:body><w:p><w:pPr><w:pStyle w:val="Heading1"/></w:pPr><w:r><w:t>Heading</w:t></w:r></w:p><w:p><w:pPr><w:pStyle w:val="ListBullet"/></w:pPr><w:r><w:t>Bullet</w:t></w:r></w:p><w:tbl><w:tr><w:tc><w:p><w:r><w:t>One</w:t></w:r></w:p></w:tc><w:tc><w:p><w:r><w:t>Two</w:t></w:r></w:p></w:tc></w:tr></w:tbl></w:body></w:document>`)
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	artifact, err := normalizeDOCX(path)
	if err != nil {
		t.Fatal(err)
	}
	text := string(artifact.Body)
	if !strings.Contains(text, "# Heading") || !strings.Contains(text, "- Bullet") || !strings.Contains(text, "| One | Two |") {
		t.Fatalf("DOCX output=%q", text)
	}
}

func TestKernel10LocalWhisperHarness(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	cfg.WhisperEnabled = true
	whisper := filepath.Join(root, "whisper-test")
	script := "#!/bin/sh\n" +
		"input=\"$1\"\n" +
		"out=\"\"\n" +
		"shift\n" +
		"while [ \"$#\" -gt 0 ]; do\n" +
		"  if [ \"$1\" = \"--output_dir\" ]; then out=\"$2\"; shift 2; continue; fi\n" +
		"  shift\n" +
		"done\n" +
		"base=$(basename \"$input\")\n" +
		"base=${base%.*}\n" +
		"printf '%s\\n' 'Pantheon transcription test.' > \"$out/$base.txt\"\n"
	if err := os.WriteFile(whisper, []byte(script), 0o700); err != nil {
		t.Fatal(err)
	}
	cfg.WhisperBinary = whisper
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfg.IntakePath, "test.m4a"), []byte("audio bytes"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	result, err := d.processOccurrence(context.Background(), id, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.State != processingVerified {
		t.Fatalf("Whisper result=%+v", result)
	}
	artifact, err := os.ReadFile(result.NormalizedPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifact), "Pantheon transcription test.") {
		t.Fatalf("transcript artifact=%q", artifact)
	}
}

func TestKernel10PDFExtractionWhenLocalDependencyExists(t *testing.T) {
	if _, err := exec.LookPath("pdftotext"); err != nil {
		t.Skip("pdftotext is not installed")
	}
	stream := "BT /F1 18 Tf 72 720 Td (Kernel 10 PDF text) Tj ET"
	objects := []string{
		"<< /Type /Catalog /Pages 2 0 R >>",
		"<< /Type /Pages /Kids [3 0 R] /Count 1 >>",
		"<< /Type /Page /Parent 2 0 R /MediaBox [0 0 612 792] /Resources << /Font << /F1 5 0 R >> >> /Contents 4 0 R >>",
		fmt.Sprintf("<< /Length %d >>\nstream\n%s\nendstream", len(stream), stream),
		"<< /Type /Font /Subtype /Type1 /BaseFont /Helvetica >>",
	}
	var pdf bytes.Buffer
	pdf.WriteString("%PDF-1.4\n")
	offsets := make([]int, len(objects)+1)
	for i, object := range objects {
		offsets[i+1] = pdf.Len()
		fmt.Fprintf(&pdf, "%d 0 obj\n%s\nendobj\n", i+1, object)
	}
	xref := pdf.Len()
	fmt.Fprintf(&pdf, "xref\n0 %d\n0000000000 65535 f \n", len(objects)+1)
	for i := 1; i < len(offsets); i++ {
		fmt.Fprintf(&pdf, "%010d 00000 n \n", offsets[i])
	}
	fmt.Fprintf(&pdf, "trailer\n<< /Size %d /Root 1 0 R >>\nstartxref\n%d\n%%%%EOF\n", len(objects)+1, xref)
	path := filepath.Join(t.TempDir(), "source.pdf")
	if err := os.WriteFile(path, pdf.Bytes(), 0o600); err != nil {
		t.Fatal(err)
	}
	artifact, err := normalizePDF(context.Background(), path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(artifact.Body), "Kernel 10 PDF text") || !strings.Contains(string(artifact.Body), "source-page: 1") {
		t.Fatalf("PDF output=%q", artifact.Body)
	}
}

func TestKernel10RawPathSafetyRejectsTraversalAndSymlinkEscape(t *testing.T) {
	if _, err := rawRelativePath(intakeOccurrence{RelativePath: "../escape.txt", FirstSeenAt: time.Now().Format(time.RFC3339Nano)}); err == nil {
		t.Fatal("path traversal was accepted")
	}
	root := t.TempDir()
	target := filepath.Join(root, "target")
	link := filepath.Join(root, "raw-link")
	if err := os.MkdirAll(target, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(target, link); err != nil {
		t.Fatal(err)
	}
	cfg := testKernel9Config(t, root)
	cfg.RawPath = link
	d := testKernel9Daemon(t, cfg)
	_, err := d.occurrenceArtifactPath(intakeOccurrence{RelativePath: "note.txt", FirstSeenAt: time.Now().Format(time.RFC3339Nano)})
	if err == nil || !strings.Contains(err.Error(), "symbolic link") {
		t.Fatalf("symlink Raw path error=%v", err)
	}
}

func TestKernel10ChangedSourceIsNotAttachedToStaleOccurrence(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(cfg.IntakePath, "stale.txt")
	if err := os.WriteFile(source, []byte("original"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var id string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences`).Scan(&id); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("changed"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.processOccurrence(context.Background(), id, ""); err == nil || !strings.Contains(err.Error(), "stale") {
		t.Fatalf("stale source result error=%v", err)
	}
	var normalized sql.NullString
	if err := d.db.QueryRow(`SELECT normalized_path FROM intake_occurrences WHERE occurrence_id = ?`, id).Scan(&normalized); err != nil {
		t.Fatal(err)
	}
	if normalized.Valid {
		t.Fatalf("stale occurrence received artifact=%q", normalized.String)
	}
}

func TestKernel10FilenameCollisionHumanEditAndMissingWhisperAreSafe(t *testing.T) {
	root := t.TempDir()
	cfg := testKernel9Config(t, root)
	d := testKernel9Daemon(t, cfg)
	if err := os.MkdirAll(cfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	source := filepath.Join(cfg.IntakePath, "meeting.txt")
	if err := os.WriteFile(source, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var firstID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences`).Scan(&firstID); err != nil {
		t.Fatal(err)
	}
	first, err := d.processOccurrence(context.Background(), firstID, "")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(first.NormalizedPath, []byte("human edit"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.processOccurrence(context.Background(), firstID, ""); err == nil || !strings.Contains(err.Error(), "modified") {
		t.Fatal("human-modified artifact was silently accepted")
	}
	edited, err := os.ReadFile(first.NormalizedPath)
	if err != nil || string(edited) != "human edit" {
		t.Fatalf("human edit was overwritten: %q err=%v", edited, err)
	}
	if err := os.Remove(source); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(source, []byte("first"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := d.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var secondID string
	if err := d.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences WHERE occurrence_id != ? ORDER BY first_seen_at DESC LIMIT 1`, firstID).Scan(&secondID); err != nil {
		t.Fatal(err)
	}
	second, err := d.processOccurrence(context.Background(), secondID, "")
	if err != nil {
		t.Fatal(err)
	}
	if second.NormalizedPath == first.NormalizedPath {
		t.Fatalf("collision overwrote first artifact: first=%q second=%q", first.NormalizedPath, second.NormalizedPath)
	}
	missingCfg := testKernel9Config(t, filepath.Join(root, "missing-whisper"))
	missingCfg.WhisperEnabled = true
	missingCfg.WhisperBinary = filepath.Join(root, "does-not-exist-whisper")
	missing := testKernel9Daemon(t, missingCfg)
	if err := os.MkdirAll(missingCfg.IntakePath, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(missingCfg.IntakePath, "audio.m4a"), []byte("audio"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := missing.runIntakeScan(); err != nil {
		t.Fatal(err)
	}
	var missingID string
	if err := missing.db.QueryRow(`SELECT occurrence_id FROM intake_occurrences`).Scan(&missingID); err != nil {
		t.Fatal(err)
	}
	result, err := missing.processOccurrence(context.Background(), missingID, "")
	if err != nil {
		t.Fatal(err)
	}
	if result.State != processingBlockedProcessor || result.RetentionState != retentionAwaitingMarkdown {
		t.Fatalf("missing Whisper result=%+v", result)
	}
}
