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
	zeusModeInventory = "inventory"
	zeusModeDryRun    = "dry_run"

	zeusStateProtected = "protected"
	zeusStateEligible  = "eligible"
	zeusStateFailed    = "failed"
	zeusStateSkipped   = "skipped"

	zeusActionQuarantine = "quarantine"
)

type zeusRunRequest struct {
	Mode string `json:"mode,omitempty"`
}

type zeusRoot struct {
	Path      string `json:"path"`
	Category  string `json:"category"`
	Owner     string `json:"owner"`
	State     string `json:"state"`
	FileCount int    `json:"file_count"`
	Bytes     int64  `json:"bytes"`
	Error     string `json:"error,omitempty"`
}

type zeusCandidate struct {
	CandidateID     string `json:"candidate_id"`
	RunID           string `json:"run_id"`
	PolicyID        string `json:"policy_id"`
	Category        string `json:"category"`
	Owner           string `json:"owner"`
	Root            string `json:"root"`
	RelativePath    string `json:"relative_path"`
	AbsolutePath    string `json:"absolute_path"`
	Bytes           int64  `json:"bytes"`
	ModifiedAt      string `json:"modified_at"`
	ContentSHA256   string `json:"content_sha256,omitempty"`
	SourceReceiptID string `json:"source_receipt_id,omitempty"`
	ReplacementPath string `json:"replacement_path,omitempty"`
	EligibleAt      string `json:"eligible_at,omitempty"`
	State           string `json:"state"`
	ReasonCode      string `json:"reason_code"`
	ReasonDetail    string `json:"reason_detail,omitempty"`
	Action          string `json:"action,omitempty"`
	DiscoveredAt    string `json:"discovered_at"`
	UpdatedAt       string `json:"updated_at"`
}

type zeusRun struct {
	RunID                string `json:"run_id"`
	Mode                 string `json:"mode"`
	PolicySnapshotSHA256 string `json:"policy_snapshot_sha256"`
	CandidateCount       int    `json:"candidate_count"`
	EligibleCount        int    `json:"eligible_count"`
	ProtectedCount       int    `json:"protected_count"`
	NeedsYouCount        int    `json:"needs_you_count"`
	ActionCount          int    `json:"action_count"`
	State                string `json:"state"`
	CreatedAt            string `json:"created_at"`
	CompletedAt          string `json:"completed_at,omitempty"`
	Error                string `json:"error,omitempty"`
}

type zeusResponse struct {
	Enabled        bool            `json:"enabled"`
	QuarantinePath string          `json:"quarantine_path,omitempty"`
	Roots          []zeusRoot      `json:"roots"`
	LatestRun      *zeusRun        `json:"latest_run,omitempty"`
	CandidateTotal int             `json:"candidate_total"`
	Candidates     []zeusCandidate `json:"candidates,omitempty"`
	CandidateLimit int             `json:"candidate_limit,omitempty"`
	Mode           string          `json:"mode,omitempty"`
	AuditID        string          `json:"audit_id,omitempty"`
}

func initializeZeusSchema(db *sql.DB) error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS zeus_runs (
			run_id TEXT PRIMARY KEY, mode TEXT NOT NULL, policy_snapshot_sha256 TEXT NOT NULL,
			candidate_count INTEGER NOT NULL, eligible_count INTEGER NOT NULL,
			protected_count INTEGER NOT NULL, needs_you_count INTEGER NOT NULL,
			action_count INTEGER NOT NULL, state TEXT NOT NULL, created_at TEXT NOT NULL,
			completed_at TEXT, error TEXT
		)`,
		`CREATE TABLE IF NOT EXISTS zeus_candidates (
			candidate_id TEXT PRIMARY KEY, run_id TEXT NOT NULL, policy_id TEXT NOT NULL,
			category TEXT NOT NULL, owner TEXT NOT NULL, root TEXT NOT NULL,
			relative_path TEXT NOT NULL, absolute_path TEXT NOT NULL, bytes INTEGER NOT NULL,
			modified_at TEXT NOT NULL, content_sha256 TEXT, source_receipt_id TEXT,
			replacement_path TEXT, eligible_at TEXT, state TEXT NOT NULL,
			reason_code TEXT NOT NULL, reason_detail TEXT, action TEXT,
			discovered_at TEXT NOT NULL, updated_at TEXT NOT NULL,
			FOREIGN KEY(run_id) REFERENCES zeus_runs(run_id)
		)`,
		`CREATE INDEX IF NOT EXISTS zeus_runs_created_idx ON zeus_runs(created_at DESC)`,
		`CREATE INDEX IF NOT EXISTS zeus_candidates_run_idx ON zeus_candidates(run_id, state, relative_path)`,
	}
	for _, statement := range statements {
		if _, err := db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

type zeusPolicy struct {
	ID       string
	Category string
	Owner    string
	Root     string
	Action   string
}

func zeusPolicyForRoot(root string) zeusPolicy {
	root = filepath.Clean(root)
	base := strings.ToLower(filepath.Base(root))
	switch {
	case base == "pantheon-go-cache" || strings.Contains(base, "go-cache"):
		return zeusPolicy{ID: "zeus.go_build_cache.v1", Category: "go_build_cache", Owner: "pantheon-build", Root: root, Action: zeusActionQuarantine}
	case base == "__pycache__":
		return zeusPolicy{ID: "zeus.python_bytecode_cache.v1", Category: "python_bytecode_cache", Owner: "pantheon-scripts", Root: root, Action: zeusActionQuarantine}
	default:
		return zeusPolicy{ID: "zeus.unknown_root.v1", Category: "unknown_machine_root", Owner: "unknown", Root: root}
	}
}

func (d *daemon) zeusPolicies() []zeusPolicy {
	roots := append([]string(nil), d.cfg.ZeusRoots...)
	seen := map[string]bool{}
	policies := make([]zeusPolicy, 0, len(roots))
	for _, root := range roots {
		root = filepath.Clean(strings.TrimSpace(root))
		if root == "." || root == "" || seen[root] {
			continue
		}
		seen[root] = true
		policies = append(policies, zeusPolicyForRoot(root))
	}
	sort.Slice(policies, func(i, j int) bool { return policies[i].Root < policies[j].Root })
	return policies
}

func zeusPolicySnapshot(policies []zeusPolicy) (string, error) {
	data, err := json.Marshal(policies)
	if err != nil {
		return "", err
	}
	return sha256Bytes(data), nil
}

func zeusCandidateID(runID, path string) string {
	return "zeus-candidate-" + strings.TrimPrefix(sha256Bytes([]byte(runID+"\x00"+path)), "sha256:")[:32]
}

func zeusCandidateForFailure(policy zeusPolicy, runID, absolute, relative, reason, detail, now string) zeusCandidate {
	return zeusCandidate{
		CandidateID: zeusCandidateID(runID, absolute), RunID: runID, PolicyID: policy.ID,
		Category: policy.Category, Owner: policy.Owner, Root: policy.Root,
		RelativePath: relative, AbsolutePath: absolute, State: zeusStateProtected,
		ReasonCode: reason, ReasonDetail: detail, DiscoveredAt: now, UpdatedAt: now,
	}
}

func inspectZeusPolicy(policy zeusPolicy, runID string) (zeusRoot, []zeusCandidate) {
	rootRecord := zeusRoot{Path: policy.Root, Category: policy.Category, Owner: policy.Owner, State: "ready"}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	if !filepath.IsAbs(policy.Root) || strings.IndexByte(policy.Root, 0) >= 0 {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = "configured root must be an absolute path without NUL bytes"
		return rootRecord, []zeusCandidate{zeusCandidateForFailure(policy, runID, policy.Root, ".", "invalid_root", rootRecord.Error, now)}
	}
	if err := ensureNoSymlinkComponents(policy.Root); err != nil {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = err.Error()
		return rootRecord, []zeusCandidate{zeusCandidateForFailure(policy, runID, policy.Root, ".", "root_symlink", rootRecord.Error, now)}
	}
	info, err := os.Lstat(policy.Root)
	if errors.Is(err, os.ErrNotExist) {
		rootRecord.State = "missing"
		rootRecord.Error = "configured root does not exist"
		return rootRecord, nil
	}
	if err != nil {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = err.Error()
		return rootRecord, nil
	}
	if info.Mode()&os.ModeSymlink != 0 {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = "configured root is a symbolic link"
		return rootRecord, []zeusCandidate{zeusCandidateForFailure(policy, runID, policy.Root, ".", "root_symlink", rootRecord.Error, now)}
	}
	if !info.IsDir() {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = "configured root is not a directory"
		return rootRecord, []zeusCandidate{zeusCandidateForFailure(policy, runID, policy.Root, ".", "root_not_directory", rootRecord.Error, now)}
	}

	candidates := []zeusCandidate{}
	walkErr := filepath.WalkDir(policy.Root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == policy.Root {
			return nil
		}
		relative, err := filepath.Rel(policy.Root, path)
		if err != nil || !pathWithin(policy.Root, path) {
			return fmt.Errorf("path escaped configured Zeus root: %s", path)
		}
		relative = filepath.ToSlash(relative)
		fileInfo, err := os.Lstat(path)
		if err != nil {
			candidates = append(candidates, zeusCandidateForFailure(policy, runID, path, relative, "stat_failed", err.Error(), now))
			return nil
		}
		if fileInfo.Mode()&os.ModeSymlink != 0 {
			rootRecord.FileCount++
			candidates = append(candidates, zeusCandidateForFailure(policy, runID, path, relative, "symlink_protected", "symbolic links are never followed or quarantined", now))
			if entry.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}
		if fileInfo.IsDir() {
			return nil
		}
		rootRecord.FileCount++
		rootRecord.Bytes += fileInfo.Size()
		candidate := zeusCandidate{
			CandidateID: zeusCandidateID(runID, path), RunID: runID, PolicyID: policy.ID,
			Category: policy.Category, Owner: policy.Owner, Root: policy.Root,
			RelativePath: relative, AbsolutePath: path, Bytes: fileInfo.Size(),
			ModifiedAt: fileInfo.ModTime().UTC().Format(time.RFC3339Nano), DiscoveredAt: now, UpdatedAt: now,
		}
		if !fileInfo.Mode().IsRegular() {
			candidate.State = zeusStateProtected
			candidate.ReasonCode = "unsupported_file_type"
			candidate.ReasonDetail = "only regular files in an explicitly recognized cache root may be quarantined"
			candidates = append(candidates, candidate)
			return nil
		}
		hashed, err := hashStableIntakeFile(path)
		if err != nil {
			candidate.State = zeusStateFailed
			candidate.ReasonCode = "hash_failed"
			candidate.ReasonDetail = err.Error()
			candidates = append(candidates, candidate)
			return nil
		}
		candidate.ContentSHA256 = hashed.ContentID
		if policy.Category == "unknown_machine_root" {
			candidate.State = zeusStateProtected
			candidate.ReasonCode = "unknown_ownership"
			candidate.ReasonDetail = "configured root has no deterministic Zeus policy"
		} else {
			candidate.State = zeusStateEligible
			candidate.EligibleAt = now
			candidate.ReasonCode = "recreatable_cache"
			candidate.ReasonDetail = "explicit machine-owned cache root; contents are reproducible"
			candidate.Action = policy.Action
		}
		candidates = append(candidates, candidate)
		return nil
	})
	if walkErr != nil {
		rootRecord.State = zeusStateProtected
		rootRecord.Error = walkErr.Error()
	}
	sort.Slice(candidates, func(i, j int) bool { return candidates[i].RelativePath < candidates[j].RelativePath })
	return rootRecord, candidates
}

func (d *daemon) runZeus(mode string) (zeusResponse, error) {
	if mode == "" {
		mode = zeusModeDryRun
	}
	if mode != zeusModeInventory && mode != zeusModeDryRun {
		return zeusResponse{}, errors.New("Zeus mode must be inventory or dry_run")
	}
	if !d.cfg.ZeusEnabled {
		return zeusResponse{}, errors.New("Zeus is disabled in configuration")
	}
	policies := d.zeusPolicies()
	if len(policies) == 0 {
		return zeusResponse{}, errors.New("Zeus has no configured machine-owned roots")
	}
	policyHash, err := zeusPolicySnapshot(policies)
	if err != nil {
		return zeusResponse{}, err
	}
	runID := fmt.Sprintf("zeus-run-%d", time.Now().UnixNano())
	createdAt := time.Now().UTC().Format(time.RFC3339Nano)
	roots := make([]zeusRoot, 0, len(policies))
	candidates := []zeusCandidate{}
	for _, policy := range policies {
		root, found := inspectZeusPolicy(policy, runID)
		roots = append(roots, root)
		candidates = append(candidates, found...)
	}
	completedAt := time.Now().UTC().Format(time.RFC3339Nano)
	run := zeusRun{RunID: runID, Mode: mode, PolicySnapshotSHA256: policyHash, CandidateCount: len(candidates), CreatedAt: createdAt, CompletedAt: completedAt, State: "completed"}
	for _, candidate := range candidates {
		switch candidate.State {
		case zeusStateEligible:
			run.EligibleCount++
		case zeusStateProtected:
			run.ProtectedCount++
		case "needs_you":
			run.NeedsYouCount++
		case zeusStateFailed:
			run.State = "completed_with_errors"
		}
	}
	run.ActionCount = run.EligibleCount
	tx, err := d.db.Begin()
	if err != nil {
		return zeusResponse{}, err
	}
	defer tx.Rollback()
	_, err = tx.Exec(`INSERT INTO zeus_runs (run_id, mode, policy_snapshot_sha256, candidate_count, eligible_count, protected_count, needs_you_count, action_count, state, created_at, completed_at, error) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`, run.RunID, run.Mode, run.PolicySnapshotSHA256, run.CandidateCount, run.EligibleCount, run.ProtectedCount, run.NeedsYouCount, run.ActionCount, run.State, run.CreatedAt, run.CompletedAt, run.Error)
	if err != nil {
		return zeusResponse{}, err
	}
	for _, candidate := range candidates {
		_, err = tx.Exec(`INSERT INTO zeus_candidates (candidate_id, run_id, policy_id, category, owner, root, relative_path, absolute_path, bytes, modified_at, content_sha256, source_receipt_id, replacement_path, eligible_at, state, reason_code, reason_detail, action, discovered_at, updated_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
			candidate.CandidateID, candidate.RunID, candidate.PolicyID, candidate.Category, candidate.Owner, candidate.Root, candidate.RelativePath, candidate.AbsolutePath, candidate.Bytes, candidate.ModifiedAt, candidate.ContentSHA256, candidate.SourceReceiptID, candidate.ReplacementPath, candidate.EligibleAt, candidate.State, candidate.ReasonCode, candidate.ReasonDetail, candidate.Action, candidate.DiscoveredAt, candidate.UpdatedAt)
		if err != nil {
			return zeusResponse{}, err
		}
	}
	if err := tx.Commit(); err != nil {
		return zeusResponse{}, err
	}
	return zeusResponse{Enabled: true, QuarantinePath: d.cfg.ZeusQuarantinePath, Roots: roots, LatestRun: &run, CandidateTotal: len(candidates), Candidates: zeusCandidatePreview(candidates, 100), CandidateLimit: 100, Mode: mode}, nil
}

func zeusCandidatePreview(candidates []zeusCandidate, limit int) []zeusCandidate {
	if limit <= 0 || len(candidates) <= limit {
		return candidates
	}
	return candidates[:limit]
}

func (d *daemon) latestZeusRun() (*zeusRun, error) {
	var run zeusRun
	var completedAt, runError sql.NullString
	err := d.db.QueryRow(`SELECT run_id, mode, policy_snapshot_sha256, candidate_count, eligible_count, protected_count, needs_you_count, action_count, state, created_at, completed_at, error FROM zeus_runs ORDER BY created_at DESC LIMIT 1`).Scan(&run.RunID, &run.Mode, &run.PolicySnapshotSHA256, &run.CandidateCount, &run.EligibleCount, &run.ProtectedCount, &run.NeedsYouCount, &run.ActionCount, &run.State, &run.CreatedAt, &completedAt, &runError)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	run.CompletedAt, run.Error = completedAt.String, runError.String
	return &run, nil
}

func (d *daemon) zeusInspection(limit int) (zeusResponse, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	policies := d.zeusPolicies()
	roots := make([]zeusRoot, 0, len(policies))
	for _, policy := range policies {
		root := zeusRoot{Path: policy.Root, Category: policy.Category, Owner: policy.Owner, State: "configured"}
		if _, err := os.Lstat(policy.Root); errors.Is(err, os.ErrNotExist) {
			root.State = "missing"
		} else if err != nil {
			root.State, root.Error = zeusStateProtected, err.Error()
		} else {
			root.State = "ready"
		}
		roots = append(roots, root)
	}
	run, err := d.latestZeusRun()
	if err != nil {
		return zeusResponse{}, err
	}
	response := zeusResponse{Enabled: d.cfg.ZeusEnabled, QuarantinePath: d.cfg.ZeusQuarantinePath, Roots: roots, LatestRun: run, CandidateLimit: limit}
	if run == nil {
		return response, nil
	}
	response.CandidateTotal = run.CandidateCount
	rows, err := d.db.Query(`SELECT candidate_id, run_id, policy_id, category, owner, root, relative_path, absolute_path, bytes, modified_at, content_sha256, source_receipt_id, replacement_path, eligible_at, state, reason_code, reason_detail, action, discovered_at, updated_at FROM zeus_candidates WHERE run_id = ? ORDER BY relative_path LIMIT ?`, run.RunID, limit)
	if err != nil {
		return zeusResponse{}, err
	}
	defer rows.Close()
	for rows.Next() {
		var candidate zeusCandidate
		var contentHash, sourceReceipt, replacement, eligible, detail, action sql.NullString
		if err := rows.Scan(&candidate.CandidateID, &candidate.RunID, &candidate.PolicyID, &candidate.Category, &candidate.Owner, &candidate.Root, &candidate.RelativePath, &candidate.AbsolutePath, &candidate.Bytes, &candidate.ModifiedAt, &contentHash, &sourceReceipt, &replacement, &eligible, &candidate.State, &candidate.ReasonCode, &detail, &action, &candidate.DiscoveredAt, &candidate.UpdatedAt); err != nil {
			return zeusResponse{}, err
		}
		candidate.ContentSHA256, candidate.SourceReceiptID, candidate.ReplacementPath, candidate.EligibleAt, candidate.ReasonDetail, candidate.Action = contentHash.String, sourceReceipt.String, replacement.String, eligible.String, detail.String, action.String
		response.Candidates = append(response.Candidates, candidate)
	}
	return response, rows.Err()
}

func (d *daemon) handleZeusInspection(w http.ResponseWriter, r *http.Request) {
	limit := 100
	if raw := r.URL.Query().Get("limit"); raw != "" {
		var parsed int
		if _, err := fmt.Sscanf(raw, "%d", &parsed); err != nil || parsed < 1 || parsed > 500 {
			writeError(w, http.StatusBadRequest, errors.New("limit must be an integer between 1 and 500"))
			return
		}
		limit = parsed
	}
	response, err := d.zeusInspection(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	writeJSON(w, http.StatusOK, response)
}

func (d *daemon) handleZeusInventory(w http.ResponseWriter, r *http.Request) {
	request := zeusRunRequest{Mode: zeusModeDryRun}
	if r.Body != nil {
		if err := decodeJSON(r, &request); err != nil && !errors.Is(err, io.EOF) {
			writeError(w, http.StatusBadRequest, err)
			return
		}
	}
	if request.Mode == "" {
		request.Mode = zeusModeDryRun
	}
	started := time.Now()
	response, err := d.runZeus(request.Mode)
	if err != nil {
		id := d.audit("POST /v1/zeus/inventory", request.Mode, "zeus.inventory", "", "failed", "", err.Error(), time.Since(started))
		writeJSON(w, http.StatusConflict, map[string]any{"error": err.Error(), "audit_id": id})
		return
	}
	response.AuditID = d.audit("POST /v1/zeus/inventory", request.Mode, "zeus.inventory", "", "completed", fmt.Sprintf("mode=%s candidates=%d eligible=%d protected=%d", response.Mode, response.LatestRun.CandidateCount, response.LatestRun.EligibleCount, response.LatestRun.ProtectedCount), "", time.Since(started))
	writeJSON(w, http.StatusOK, response)
}
