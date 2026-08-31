package pantheon

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
)

const (
	actionJobPending = "pending"
	actionJobRunning = "running"
	actionJobRetry   = "retry"
	actionJobDone    = "completed"
	actionJobFailed  = "failed"
)

type actionJob struct {
	JobID          string `json:"job_id"`
	SourceKind     string `json:"source_kind"`
	SourceID       string `json:"source_id"`
	OccurrenceID   string `json:"occurrence_id"`
	State          string `json:"state"`
	AttemptCount   int    `json:"attempt_count"`
	ApolloOrderID  string `json:"apollo_order_id,omitempty"`
	ApolloReportID string `json:"apollo_report_id,omitempty"`
	LastError      string `json:"last_error,omitempty"`
	NextAttemptAt  string `json:"next_attempt_at,omitempty"`
	CreatedAt      string `json:"created_at"`
	UpdatedAt      string `json:"updated_at"`
}

var actionJobMu sync.Mutex

func initializeActionJobSchema(db *sql.DB) error {
	_, err := db.Exec(`CREATE TABLE IF NOT EXISTS action_jobs (
		job_id TEXT PRIMARY KEY, source_kind TEXT NOT NULL, source_id TEXT NOT NULL,
		occurrence_id TEXT NOT NULL, state TEXT NOT NULL, attempt_count INTEGER NOT NULL,
		apollo_order_id TEXT, apollo_report_id TEXT, last_error TEXT, next_attempt_at TEXT,
		created_at TEXT NOT NULL, updated_at TEXT NOT NULL, UNIQUE(source_kind, source_id)
	)`)
	if err != nil {
		return err
	}
	_, err = db.Exec(`CREATE INDEX IF NOT EXISTS action_jobs_due_idx ON action_jobs(state, next_attempt_at, updated_at)`)
	return err
}

func (d *daemon) enqueueActionJob(sourceKind, sourceID, occurrenceID string) error {
	if err := initializeActionJobSchema(d.db); err != nil {
		return err
	}
	sourceKind, sourceID, occurrenceID = strings.TrimSpace(sourceKind), strings.TrimSpace(sourceID), strings.TrimSpace(occurrenceID)
	if sourceKind == "" || sourceID == "" || occurrenceID == "" {
		return errors.New("action job requires source kind, source ID, and occurrence ID")
	}
	now := time.Now().UTC().Format(time.RFC3339Nano)
	jobID, err := newLocalID("action-job-")
	if err != nil {
		return err
	}
	_, err = d.db.Exec(`INSERT INTO action_jobs (job_id, source_kind, source_id, occurrence_id, state, attempt_count, created_at, updated_at) VALUES (?, ?, ?, ?, ?, 0, ?, ?) ON CONFLICT(source_kind, source_id) DO UPDATE SET occurrence_id = excluded.occurrence_id, state = CASE WHEN action_jobs.state = ? THEN action_jobs.state ELSE ? END, last_error = CASE WHEN action_jobs.state = ? THEN action_jobs.last_error ELSE '' END, next_attempt_at = '', updated_at = excluded.updated_at`, jobID, sourceKind, sourceID, occurrenceID, actionJobPending, now, now, actionJobDone, actionJobPending, actionJobDone)
	return err
}

func (d *daemon) loadActionJob(id string) (actionJob, error) {
	if err := initializeActionJobSchema(d.db); err != nil {
		return actionJob{}, err
	}
	var job actionJob
	var orderID, reportID, lastError, nextAttempt sql.NullString
	err := d.db.QueryRow(`SELECT job_id, source_kind, source_id, occurrence_id, state, attempt_count, COALESCE(apollo_order_id, ''), COALESCE(apollo_report_id, ''), COALESCE(last_error, ''), COALESCE(next_attempt_at, ''), created_at, updated_at FROM action_jobs WHERE job_id = ?`, id).Scan(&job.JobID, &job.SourceKind, &job.SourceID, &job.OccurrenceID, &job.State, &job.AttemptCount, &orderID, &reportID, &lastError, &nextAttempt, &job.CreatedAt, &job.UpdatedAt)
	if err != nil {
		return actionJob{}, err
	}
	job.ApolloOrderID, job.ApolloReportID, job.LastError, job.NextAttemptAt = orderID.String, reportID.String, lastError.String, nextAttempt.String
	return job, nil
}

func (d *daemon) listActionJobs(limit int) ([]actionJob, error) {
	if err := initializeActionJobSchema(d.db); err != nil {
		return nil, err
	}
	if limit < 1 || limit > 200 {
		limit = 100
	}
	rows, err := d.db.Query(`SELECT job_id FROM action_jobs ORDER BY updated_at DESC, job_id DESC LIMIT ?`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []actionJob
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, err
		}
		job, err := d.loadActionJob(id)
		if err != nil {
			return nil, err
		}
		result = append(result, job)
	}
	return result, rows.Err()
}

func (d *daemon) actionJobInspection() (map[string]any, error) {
	jobs, err := d.listActionJobs(100)
	if err != nil {
		return nil, err
	}
	counts := map[string]int{}
	for _, job := range jobs {
		counts[job.State]++
	}
	return map[string]any{"state_counts": counts, "jobs": jobs, "automatic_local_apollo": true, "external_execution": false}, nil
}

func (d *daemon) claimActionJob() (actionJob, bool, error) {
	if err := initializeActionJobSchema(d.db); err != nil {
		return actionJob{}, false, err
	}
	actionJobMu.Lock()
	defer actionJobMu.Unlock()
	now := time.Now().UTC().Format(time.RFC3339Nano)
	var id string
	// Omi audio needs digest-level treatment, not one Apollo action report per
	// cutting. Preserve the action jobs so a later digest worker can consume
	// them, but keep this per-occurrence worker off the Omi queue entirely.
	err := d.db.QueryRow(`SELECT j.job_id FROM action_jobs j WHERE j.state IN (?, ?) AND (j.next_attempt_at IS NULL OR j.next_attempt_at = '' OR j.next_attempt_at <= ?) AND j.source_kind <> 'rumor' ORDER BY j.updated_at, j.job_id LIMIT 1`, actionJobPending, actionJobRetry, now).Scan(&id)
	if errors.Is(err, sql.ErrNoRows) {
		return actionJob{}, false, nil
	}
	if err != nil {
		return actionJob{}, false, err
	}
	result, err := d.db.Exec(`UPDATE action_jobs SET state = ?, attempt_count = attempt_count + 1, last_error = '', next_attempt_at = '', updated_at = ? WHERE job_id = ? AND state IN (?, ?)`, actionJobRunning, now, id, actionJobPending, actionJobRetry)
	if err != nil {
		return actionJob{}, false, err
	}
	changed, err := result.RowsAffected()
	if err != nil || changed != 1 {
		return actionJob{}, false, err
	}
	job, err := d.loadActionJob(id)
	return job, true, err
}

func (d *daemon) finishActionJob(job actionJob, state, message, orderID, reportID string) error {
	now := time.Now().UTC().Format(time.RFC3339Nano)
	next := ""
	if state == actionJobRetry {
		next = time.Now().UTC().Add(time.Duration(job.AttemptCount) * time.Minute).Format(time.RFC3339Nano)
	}
	_, err := d.db.Exec(`UPDATE action_jobs SET state = ?, apollo_order_id = CASE WHEN ? <> '' THEN ? ELSE apollo_order_id END, apollo_report_id = CASE WHEN ? <> '' THEN ? ELSE apollo_report_id END, last_error = ?, next_attempt_at = ?, updated_at = ? WHERE job_id = ?`, state, orderID, orderID, reportID, reportID, nullableString(message), nullableString(next), now, job.JobID)
	return err
}

func (d *daemon) runActionJob(ctx context.Context, job actionJob) error {
	if paused, err := d.paused(); err != nil {
		return err
	} else if paused {
		return errors.New("Prometheus is paused; Omi action reasoning is waiting")
	}
	request := "Understand this Omi transcript for explicit personal requests. Extract only clearly requested calendar events, email drafts, reminders, or other bounded work. Do not infer an action from ordinary conversation. For a calendar proposal, provide title, start, and end when the source supports them. For an email draft, provide to, subject, and body only when the person clearly requested one. Treat every proposed action as a draft for human review, never as authority. Name uncertainties and missing fields plainly, and cite the transcript."
	present, err := d.processOccurrence(ctx, job.OccurrenceID, request)
	if err != nil {
		return fmt.Errorf("Present: %w", err)
	}
	if present.State != processingVerified {
		if present.Error != "" {
			return errors.New("Present: " + present.Error)
		}
		return errors.New("Present did not verify the Omi transcript")
	}
	payload, _ := json.Marshal(apolloOrderPayload{OccurrenceID: job.OccurrenceID})
	order, err := d.createOrder(orderRequest{RequestText: request, Recipient: "apollo", Capability: apolloCapability, Scope: "omi-action:" + job.JobID, Payload: payload, Purpose: "Kernel 32 Omi action understanding"}, true)
	if err != nil {
		return fmt.Errorf("Apollo order: %w", err)
	}
	if order.Status != orderStatusSealed {
		return fmt.Errorf("Apollo order ended in %s: %s", order.Status, order.LastError)
	}
	command, err := d.issueGo(order)
	if err != nil {
		return fmt.Errorf("Apollo Go: %w", err)
	}
	if err := d.dispatchOrder(ctx, order, command); err != nil {
		return fmt.Errorf("Apollo dispatch: %w", err)
	}
	var reportID string
	if err := d.db.QueryRow(`SELECT report_id FROM apollo_reports WHERE order_id = ? ORDER BY created_at DESC LIMIT 1`, order.OrderID).Scan(&reportID); err != nil {
		return fmt.Errorf("Apollo report: %w", err)
	}
	return d.finishActionJob(job, actionJobDone, "", order.OrderID, reportID)
}

func (d *daemon) processActionJobs(ctx context.Context, limit int) (int, error) {
	processed := 0
	for processed < limit {
		job, found, err := d.claimActionJob()
		if err != nil || !found {
			return processed, err
		}
		if err := d.runActionJob(ctx, job); err != nil {
			state := actionJobRetry
			if job.AttemptCount >= 3 {
				state = actionJobFailed
			}
			if finishErr := d.finishActionJob(job, state, err.Error(), "", ""); finishErr != nil {
				return processed, finishErr
			}
			if state == actionJobFailed {
				processed++
				continue
			}
			return processed, err
		}
		processed++
	}
	return processed, nil
}

func (d *daemon) startActionWatcher() func() {
	ctx, cancel := context.WithCancel(context.Background())
	interval := time.Duration(d.cfg.ZodiacWakeSeconds) * time.Second
	if interval < 5*time.Second {
		interval = 30 * time.Second
	}
	go func() {
		cycle := func() {
			if _, err := d.processActionJobs(ctx, 1); err != nil && !strings.Contains(err.Error(), "Prometheus is paused") {
				d.audit("actions.watcher", "", "apollo.action", "", "failed", "", err.Error(), 0)
			}
		}
		cycle()
		ticker := time.NewTicker(interval)
		defer ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				cycle()
			}
		}
	}()
	return cancel
}
