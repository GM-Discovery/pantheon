package pantheon

import (
	"bytes"
	"crypto/subtle"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

const (
	externalIntakeSecretEnv = "PANTHEON_INTAKE_BRIDGE_SECRET"
	externalIntakeMaxBytes  = 8 << 20
)

var externalIntakeUnsafeChars = regexp.MustCompile(`[^a-zA-Z0-9._-]+`)

type externalIntakeEnvelope struct {
	SourceKind    string          `json:"source_kind"`
	SourceID      string          `json:"source_id"`
	SourceEventID string          `json:"source_event_id,omitempty"`
	ReceivedAt    string          `json:"received_at,omitempty"`
	OccurredAt    string          `json:"occurred_at,omitempty"`
	ContentSHA256 string          `json:"content_sha256,omitempty"`
	PayloadFormat string          `json:"payload_format,omitempty"`
	Payload       json.RawMessage `json:"payload"`
	Transcript    string          `json:"transcript,omitempty"`
	Labels        []string        `json:"labels,omitempty"`
	Metadata      json.RawMessage `json:"metadata,omitempty"`
}

type externalIntakeReceipt struct {
	ReceiptID              string `json:"receipt_id"`
	SourceKind             string `json:"source_kind"`
	SourceID               string `json:"source_id"`
	SourceEventID          string `json:"source_event_id,omitempty"`
	ReceivedAt             string `json:"received_at"`
	OccurredAt             string `json:"occurred_at"`
	ContentSHA256          string `json:"content_sha256"`
	TranscriptSHA256       string `json:"transcript_sha256,omitempty"`
	RawPayloadPath         string `json:"raw_payload_path"`
	EnvelopePath           string `json:"envelope_path,omitempty"`
	TranscriptPath         string `json:"transcript_path,omitempty"`
	RawOccurrenceID        string `json:"raw_occurrence_id,omitempty"`
	TranscriptOccurrenceID string `json:"transcript_occurrence_id,omitempty"`
	NormalizedPath         string `json:"normalized_path,omitempty"`
	Status                 string `json:"status"`
	Error                  string `json:"error,omitempty"`
	CreatedAt              string `json:"created_at"`
	UpdatedAt              string `json:"updated_at"`
}

type omiMemoryWebhook struct {
	ID                 string              `json:"id"`
	CreatedAt          string              `json:"created_at,omitempty"`
	StartedAt          string              `json:"started_at,omitempty"`
	FinishedAt         string              `json:"finished_at,omitempty"`
	TranscriptSegments []omiTranscriptPart `json:"transcript_segments,omitempty"`
	Structured         omiStructuredMemory `json:"structured,omitempty"`
	FolderID           string              `json:"folder_id,omitempty"`
	FolderName         string              `json:"folder_name,omitempty"`
}

type omiTranscriptPart struct {
	Text        string  `json:"text,omitempty"`
	Speaker     string  `json:"speaker,omitempty"`
	SpeakerID   *int    `json:"speakerId,omitempty"`
	SpeakerName string  `json:"speaker_name,omitempty"`
	Start       float64 `json:"start,omitempty"`
	End         float64 `json:"end,omitempty"`
	IsUser      bool    `json:"is_user,omitempty"`
}

type omiStructuredMemory struct {
	Title    string `json:"title,omitempty"`
	Category string `json:"category,omitempty"`
}

func initializeExternalIntakeSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS external_intake_receipts (
		receipt_id TEXT PRIMARY KEY, source_kind TEXT NOT NULL, source_id TEXT NOT NULL,
		source_event_id TEXT, received_at TEXT NOT NULL, occurred_at TEXT NOT NULL,
		content_sha256 TEXT NOT NULL, transcript_sha256 TEXT, raw_payload_path TEXT NOT NULL,
		transcript_path TEXT, raw_occurrence_id TEXT, transcript_occurrence_id TEXT,
		normalized_path TEXT, status TEXT NOT NULL, error TEXT, created_at TEXT NOT NULL,
		updated_at TEXT NOT NULL, envelope_path TEXT, UNIQUE(source_kind, source_id)
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS external_intake_receipts_updated_idx ON external_intake_receipts(updated_at DESC)`)
	if err != nil {
		return err
	}
	if _, err := db.Exec(`ALTER TABLE external_intake_receipts ADD COLUMN envelope_path TEXT`); err != nil && !strings.Contains(strings.ToLower(err.Error()), "duplicate column") {
		return err
	}
	return nil
}

func (d *daemon) loadExternalIntakeReceipt(sourceKind, sourceID string) (externalIntakeReceipt, error) {
	var receipt externalIntakeReceipt
	err := d.db.QueryRow(`SELECT receipt_id, source_kind, source_id, COALESCE(source_event_id, ''), received_at,
		occurred_at, content_sha256, COALESCE(transcript_sha256, ''), raw_payload_path, COALESCE(envelope_path, ''), COALESCE(transcript_path, ''),
		COALESCE(raw_occurrence_id, ''), COALESCE(transcript_occurrence_id, ''), COALESCE(normalized_path, ''),
		status, COALESCE(error, ''), created_at, updated_at FROM external_intake_receipts WHERE source_kind = ? AND source_id = ?`, sourceKind, sourceID).Scan(
		&receipt.ReceiptID, &receipt.SourceKind, &receipt.SourceID, &receipt.SourceEventID, &receipt.ReceivedAt,
		&receipt.OccurredAt, &receipt.ContentSHA256, &receipt.TranscriptSHA256, &receipt.RawPayloadPath, &receipt.EnvelopePath, &receipt.TranscriptPath,
		&receipt.RawOccurrenceID, &receipt.TranscriptOccurrenceID, &receipt.NormalizedPath, &receipt.Status, &receipt.Error,
		&receipt.CreatedAt, &receipt.UpdatedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return externalIntakeReceipt{}, nil
	}
	return receipt, err
}

func (d *daemon) listExternalIntakeReceipts(limit int) ([]externalIntakeReceipt, error) {
	if limit < 1 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT receipt_id, source_kind, source_id, COALESCE(source_event_id, ''), received_at,
		occurred_at, content_sha256, COALESCE(transcript_sha256, ''), raw_payload_path, COALESCE(envelope_path, ''), COALESCE(transcript_path, ''),
		COALESCE(raw_occurrence_id, ''), COALESCE(transcript_occurrence_id, ''), COALESCE(normalized_path, ''),
		status, COALESCE(error, ''), created_at, updated_at FROM external_intake_receipts ORDER BY updated_at DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []externalIntakeReceipt
	for rows.Next() {
		var receipt externalIntakeReceipt
		if err := rows.Scan(&receipt.ReceiptID, &receipt.SourceKind, &receipt.SourceID, &receipt.SourceEventID, &receipt.ReceivedAt,
			&receipt.OccurredAt, &receipt.ContentSHA256, &receipt.TranscriptSHA256, &receipt.RawPayloadPath, &receipt.EnvelopePath, &receipt.TranscriptPath,
			&receipt.RawOccurrenceID, &receipt.TranscriptOccurrenceID, &receipt.NormalizedPath, &receipt.Status, &receipt.Error,
			&receipt.CreatedAt, &receipt.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, receipt)
	}
	return result, rows.Err()
}

func validateExternalIntakeEnvelope(request *externalIntakeEnvelope) (time.Time, string, string, error) {
	request.SourceKind = strings.TrimSpace(request.SourceKind)
	request.SourceID = strings.TrimSpace(request.SourceID)
	if request.SourceKind == "" || len(request.SourceKind) > 80 {
		return time.Time{}, "", "", errors.New("source_kind is required and must be at most 80 characters")
	}
	if request.SourceID == "" || len(request.SourceID) > 240 {
		return time.Time{}, "", "", errors.New("source_id is required and must be at most 240 characters")
	}
	if len(request.SourceEventID) > 240 {
		return time.Time{}, "", "", errors.New("source_event_id must be at most 240 characters")
	}
	if len(request.Payload) == 0 || !json.Valid(request.Payload) {
		return time.Time{}, "", "", errors.New("payload must be valid, non-empty JSON")
	}
	if len(request.Transcript) > externalIntakeMaxBytes {
		return time.Time{}, "", "", errors.New("transcript is too large")
	}
	if request.PayloadFormat == "" {
		request.PayloadFormat = "application/json"
	}
	receivedAt := time.Now().UTC()
	if request.ReceivedAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.ReceivedAt)
		if err != nil {
			return time.Time{}, "", "", fmt.Errorf("received_at must be RFC3339: %w", err)
		}
		receivedAt = parsed
	}
	occurredAt := receivedAt
	if request.OccurredAt != "" {
		parsed, err := time.Parse(time.RFC3339Nano, request.OccurredAt)
		if err != nil {
			return time.Time{}, "", "", fmt.Errorf("occurred_at must be RFC3339: %w", err)
		}
		occurredAt = parsed
	}
	contentHash := sha256Bytes(request.Payload)
	if request.ContentSHA256 != "" && request.ContentSHA256 != contentHash {
		return time.Time{}, "", "", errors.New("content_sha256 does not match payload")
	}
	request.ReceivedAt = receivedAt.UTC().Format(time.RFC3339Nano)
	request.OccurredAt = occurredAt.UTC().Format(time.RFC3339Nano)
	request.ContentSHA256 = contentHash
	return occurredAt, contentHash, sha256Bytes([]byte(request.Transcript)), nil
}

func externalIntakeSlug(value string) string {
	slug := externalIntakeUnsafeChars.ReplaceAllString(strings.TrimSpace(value), "-")
	slug = strings.Trim(slug, ".-")
	if len(slug) > 120 {
		slug = slug[:120]
	}
	if slug == "" {
		return "event"
	}
	return slug
}

func externalIntakeAuthorized(r *http.Request) bool {
	expected := strings.TrimSpace(os.Getenv(externalIntakeSecretEnv))
	if expected == "" {
		return false
	}
	provided := strings.TrimSpace(r.Header.Get("X-Pantheon-Intake-Secret"))
	if provided == "" {
		authorization := strings.TrimSpace(r.Header.Get("Authorization"))
		if strings.HasPrefix(authorization, "Bearer ") {
			provided = strings.TrimSpace(strings.TrimPrefix(authorization, "Bearer "))
		}
	}
	return len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func externalIntakeSecretMatches(provided string) bool {
	expected := strings.TrimSpace(os.Getenv(externalIntakeSecretEnv))
	provided = strings.TrimSpace(provided)
	return expected != "" && len(provided) == len(expected) && subtle.ConstantTimeCompare([]byte(provided), []byte(expected)) == 1
}

func omiWebhookAuthorized(r *http.Request) bool {
	if externalIntakeAuthorized(r) {
		return true
	}
	return externalIntakeSecretMatches(r.URL.Query().Get("pantheon_intake_secret"))
}

func omiTimestamp(value string) string {
	if value == "" {
		return ""
	}
	if _, err := time.Parse(time.RFC3339Nano, value); err != nil {
		return ""
	}
	return value
}

func omiTranscriptTimestamp(seconds float64) string {
	if seconds < 0 {
		seconds = 0
	}
	hours := int(seconds) / 3600
	minutes := (int(seconds) % 3600) / 60
	wholeSeconds := int(seconds) % 60
	milliseconds := int((seconds - float64(int(seconds))) * 1000)
	if milliseconds > 999 {
		milliseconds = 999
	}
	return fmt.Sprintf("%02d:%02d:%02d.%03d", hours, minutes, wholeSeconds, milliseconds)
}

func omiTranscriptText(parts []omiTranscriptPart) string {
	var body strings.Builder
	for _, part := range parts {
		text := strings.TrimSpace(part.Text)
		if text == "" {
			continue
		}
		speaker := strings.TrimSpace(part.SpeakerName)
		if speaker == "" {
			speaker = strings.TrimSpace(part.Speaker)
		}
		if speaker == "" && part.SpeakerID != nil {
			speaker = fmt.Sprintf("SPEAKER_%02d", *part.SpeakerID)
		}
		if speaker == "" {
			speaker = "UNKNOWN_SPEAKER"
		}
		fmt.Fprintf(&body, "[%s] %s: %s\n", omiTranscriptTimestamp(part.Start), speaker, text)
	}
	return strings.TrimSpace(body.String())
}

func omiWebhookEnvelope(raw []byte, query url.Values) (externalIntakeEnvelope, error) {
	var memory omiMemoryWebhook
	if err := json.Unmarshal(raw, &memory); err != nil {
		return externalIntakeEnvelope{}, fmt.Errorf("invalid Omi memory JSON: %w", err)
	}
	if strings.TrimSpace(memory.ID) == "" {
		return externalIntakeEnvelope{}, errors.New("Omi memory payload is missing id")
	}
	occurredAt := omiTimestamp(memory.StartedAt)
	if occurredAt == "" {
		occurredAt = omiTimestamp(memory.CreatedAt)
	}
	labels := make([]string, 0, 2)
	if label := strings.TrimSpace(memory.FolderName); label != "" {
		labels = append(labels, label)
	}
	if label := strings.TrimSpace(memory.Structured.Category); label != "" && !containsString(labels, label) {
		labels = append(labels, label)
	}
	metadata := map[string]string{}
	if uid := strings.TrimSpace(query.Get("uid")); uid != "" {
		metadata["omi_uid"] = uid
	}
	if memory.FinishedAt != "" {
		metadata["finished_at"] = memory.FinishedAt
	}
	if memory.FolderID != "" {
		metadata["folder_id"] = memory.FolderID
	}
	metadataBytes, err := json.Marshal(metadata)
	if err != nil {
		return externalIntakeEnvelope{}, err
	}
	return externalIntakeEnvelope{
		SourceKind:    "omi_memory",
		SourceID:      memory.ID,
		ReceivedAt:    time.Now().UTC().Format(time.RFC3339Nano),
		OccurredAt:    occurredAt,
		PayloadFormat: "application/json",
		Payload:       json.RawMessage(raw),
		Transcript:    omiTranscriptText(memory.TranscriptSegments),
		Labels:        labels,
		Metadata:      metadataBytes,
	}, nil
}

func writeExternalIntakeFile(path string, data []byte) error {
	if err := ensureNoSymlinkComponents(filepath.Dir(path)); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	if existing, err := os.ReadFile(path); err == nil {
		if bytes.Equal(existing, data) {
			return nil
		}
		return fmt.Errorf("refusing to overwrite existing Intake file %s with different content", path)
	} else if !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".pantheon-intake-*")
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

func (d *daemon) findIntakeOccurrenceByPath(relativePath string) (intakeOccurrence, error) {
	rows, err := d.queryIntakeOccurrences(10, relativePath, "", "")
	if err != nil {
		return intakeOccurrence{}, err
	}
	for _, occurrence := range rows {
		if occurrence.State == occurrencePresent {
			return d.loadIntakeOccurrence(occurrence.OccurrenceID)
		}
	}
	return intakeOccurrence{}, fmt.Errorf("Intake occurrence for %q was not found", relativePath)
}

func (d *daemon) persistExternalIntakeReceipt(receipt externalIntakeReceipt) error {
	_, err := d.db.Exec(`INSERT INTO external_intake_receipts
		(receipt_id, source_kind, source_id, source_event_id, received_at, occurred_at, content_sha256,
		 transcript_sha256, raw_payload_path, envelope_path, transcript_path, raw_occurrence_id, transcript_occurrence_id,
		 normalized_path, status, error, created_at, updated_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, receipt.ReceiptID, receipt.SourceKind, receipt.SourceID,
		receipt.SourceEventID, receipt.ReceivedAt, receipt.OccurredAt, receipt.ContentSHA256, receipt.TranscriptSHA256,
		receipt.RawPayloadPath, receipt.EnvelopePath, receipt.TranscriptPath, receipt.RawOccurrenceID, receipt.TranscriptOccurrenceID,
		receipt.NormalizedPath, receipt.Status, receipt.Error, receipt.CreatedAt, receipt.UpdatedAt)
	return err
}

func (d *daemon) updateExternalIntakeReceipt(receipt externalIntakeReceipt) error {
	_, err := d.db.Exec(`UPDATE external_intake_receipts SET raw_occurrence_id = ?, transcript_occurrence_id = ?, normalized_path = ?, status = ?, error = ?, updated_at = ? WHERE receipt_id = ?`,
		receipt.RawOccurrenceID, receipt.TranscriptOccurrenceID, receipt.NormalizedPath, receipt.Status, receipt.Error, receipt.UpdatedAt, receipt.ReceiptID)
	return err
}

func (d *daemon) handleExternalIntake(w http.ResponseWriter, r *http.Request) {
	started := time.Now()
	if !externalIntakeAuthorized(r) {
		writeError(w, http.StatusUnauthorized, errors.New("external Intake authentication failed"))
		return
	}
	if d.blockIfPaused(w, "POST /v1/intake/external", d.cfg.IntakePath, "intake.external") {
		return
	}
	var request externalIntakeEnvelope
	if err := json.NewDecoder(io.LimitReader(r.Body, externalIntakeMaxBytes)).Decode(&request); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("invalid external Intake envelope: %w", err))
		return
	}
	occurredAt, contentHash, transcriptHash, err := validateExternalIntakeEnvelope(&request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := initializeExternalIntakeSchema(d.db); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	existing, err := d.loadExternalIntakeReceipt(request.SourceKind, request.SourceID)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if existing.ReceiptID != "" {
		if existing.ContentSHA256 != contentHash || existing.TranscriptSHA256 != transcriptHash {
			writeError(w, http.StatusConflict, errors.New("source_id already exists with different content"))
			return
		}
		if existing.Status == "completed" {
			if err := d.enqueueLibraIngestionJobs(existing); err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Errorf("queue Libra ingestion report retry: %w", err))
				return
			}
			if existing.TranscriptOccurrenceID != "" {
				if err := d.enqueueActionJob(existing.SourceKind, existing.SourceID, existing.TranscriptOccurrenceID); err != nil {
					writeError(w, http.StatusInternalServerError, fmt.Errorf("queue Omi action reasoning retry: %w", err))
					return
				}
			}
		}
		existing.Status = "duplicate"
		writeJSON(w, http.StatusOK, existing)
		return
	}
	if strings.TrimSpace(d.cfg.IntakePath) == "" {
		writeError(w, http.StatusInternalServerError, errors.New("Intake path is not configured"))
		return
	}
	if err := ensureNoSymlinkComponents(d.cfg.IntakePath); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	datePath := occurredAt.UTC().Format("2006/01")
	base := externalIntakeSlug(request.SourceKind + "-" + request.SourceID)
	rawRelative := filepath.Join("Omi", datePath, base+".json")
	envelopeRelative := filepath.Join("Omi", datePath, base+".envelope.json")
	transcriptRelative := filepath.Join("Transcripts", datePath, base+".md")
	rawPath := filepath.Join(d.cfg.IntakePath, rawRelative)
	envelopePath := filepath.Join(d.cfg.IntakePath, envelopeRelative)
	transcriptPath := filepath.Join(d.cfg.IntakePath, transcriptRelative)
	if !pathWithin(d.cfg.IntakePath, rawPath) || !pathWithin(d.cfg.IntakePath, envelopePath) || !pathWithin(d.cfg.IntakePath, transcriptPath) {
		writeError(w, http.StatusBadRequest, errors.New("external Intake path escapes configured Intake"))
		return
	}
	if err := writeExternalIntakeFile(rawPath, request.Payload); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	envelopeBytes, err := json.Marshal(request)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if err := writeExternalIntakeFile(envelopePath, envelopeBytes); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if strings.TrimSpace(request.Transcript) != "" {
		var body strings.Builder
		fmt.Fprintf(&body, "---\npantheon:\n  kind: omi-transcript\n  source_kind: %s\n  source_id: %s\n  source_event_id: %s\n  content_sha256: %s\n  transcript_sha256: %s\n  occurred_at: %s\n  labels:\n", yamlQuote(request.SourceKind), yamlQuote(request.SourceID), yamlQuote(request.SourceEventID), yamlQuote(contentHash), yamlQuote(transcriptHash), yamlQuote(request.OccurredAt))
		for _, label := range request.Labels {
			fmt.Fprintf(&body, "    - %s\n", yamlQuote(label))
		}
		body.WriteString("---\n\n# Omi Transcript — ")
		body.WriteString(request.SourceID)
		body.WriteString("\n\n")
		body.WriteString(request.Transcript)
		body.WriteString("\n")
		if err := writeExternalIntakeFile(transcriptPath, []byte(body.String())); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
	}
	if _, err := d.runIntakeScan(); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	rawOccurrence, err := d.findIntakeOccurrenceByPath(rawRelative)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	receiptID, err := newLocalID("omi-receipt-")
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	receipt := externalIntakeReceipt{ReceiptID: receiptID, SourceKind: request.SourceKind, SourceID: request.SourceID, SourceEventID: request.SourceEventID,
		ReceivedAt: request.ReceivedAt, OccurredAt: request.OccurredAt, ContentSHA256: contentHash, TranscriptSHA256: transcriptHash,
		RawPayloadPath: rawRelative, EnvelopePath: envelopeRelative, Status: "accepted", RawOccurrenceID: rawOccurrence.OccurrenceID, CreatedAt: now, UpdatedAt: now}
	if strings.TrimSpace(request.Transcript) == "" {
		receipt.Status = "accepted_pending_transcript"
	} else {
		receipt.TranscriptPath = transcriptRelative
		transcriptOccurrence, findErr := d.findIntakeOccurrenceByPath(transcriptRelative)
		if findErr != nil {
			receipt.Status = "accepted_pending_conversion"
			receipt.Error = findErr.Error()
		} else {
			receipt.TranscriptOccurrenceID = transcriptOccurrence.OccurrenceID
			result, processErr := d.processOccurrence(r.Context(), transcriptOccurrence.OccurrenceID, "Kernel 26 Omi transcript conversion")
			if processErr != nil || result.State != processingVerified {
				receipt.Status = "accepted_pending_conversion"
				if processErr != nil {
					receipt.Error = processErr.Error()
				} else {
					receipt.Error = result.Verification
				}
			} else {
				receipt.Status = "completed"
				receipt.NormalizedPath = result.NormalizedPath
			}
		}
	}
	if err := d.persistExternalIntakeReceipt(receipt); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if receipt.Status == "completed" {
		if err := d.enqueueLibraIngestionJobs(receipt); err != nil {
			writeError(w, http.StatusInternalServerError, fmt.Errorf("queue Libra ingestion report: %w", err))
			return
		}
		if receipt.TranscriptOccurrenceID != "" {
			if err := d.enqueueActionJob(receipt.SourceKind, receipt.SourceID, receipt.TranscriptOccurrenceID); err != nil {
				writeError(w, http.StatusInternalServerError, fmt.Errorf("queue Omi action reasoning: %w", err))
				return
			}
		}
	}
	status := http.StatusAccepted
	if receipt.Status == "completed" {
		status = http.StatusCreated
	}
	d.audit("POST /v1/intake/external", request.SourceID, "intake.external", "", receipt.Status, receipt.ReceiptID, receipt.Error, time.Since(started))
	writeJSON(w, status, receipt)
}

func (d *daemon) handleOmiWebhook(w http.ResponseWriter, r *http.Request) {
	if !omiWebhookAuthorized(r) {
		writeError(w, http.StatusUnauthorized, errors.New("Omi webhook authentication failed"))
		return
	}
	raw, err := io.ReadAll(io.LimitReader(r.Body, externalIntakeMaxBytes+1))
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("could not read Omi webhook: %w", err))
		return
	}
	if len(raw) > externalIntakeMaxBytes {
		writeError(w, http.StatusRequestEntityTooLarge, errors.New("Omi webhook payload is too large"))
		return
	}
	envelope, err := omiWebhookEnvelope(raw, r.URL.Query())
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	body, err := json.Marshal(envelope)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	forwarded := r.Clone(r.Context())
	forwarded.URL = &url.URL{Path: "/v1/intake/external"}
	forwarded.Body = io.NopCloser(bytes.NewReader(body))
	forwarded.Header = r.Header.Clone()
	forwarded.Header.Set("X-Pantheon-Intake-Secret", strings.TrimSpace(os.Getenv(externalIntakeSecretEnv)))
	d.handleExternalIntake(w, forwarded)
}

func (d *daemon) handleExternalIntakeReceipts(w http.ResponseWriter, r *http.Request) {
	if !externalIntakeAuthorized(r) {
		writeError(w, http.StatusUnauthorized, errors.New("external Intake authentication failed"))
		return
	}
	if sourceID := r.PathValue("source_id"); sourceID != "" {
		sourceKind := r.URL.Query().Get("source_kind")
		if sourceKind == "" {
			sourceKind = "omi_memory"
		}
		receipt, err := d.loadExternalIntakeReceipt(sourceKind, sourceID)
		if err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		if receipt.ReceiptID == "" {
			writeError(w, http.StatusNotFound, errors.New("external Intake receipt not found"))
			return
		}
		writeJSON(w, http.StatusOK, receipt)
		return
	}
	limit := 20
	if rawLimit := r.URL.Query().Get("limit"); rawLimit != "" {
		if _, err := fmt.Sscanf(rawLimit, "%d", &limit); err != nil {
			writeError(w, http.StatusBadRequest, errors.New("limit must be an integer"))
			return
		}
	}
	result, err := d.listExternalIntakeReceipts(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"receipts": result})
}
