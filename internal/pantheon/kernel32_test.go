package pantheon

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func kernel32EmailPackage() seshatPackageSpec {
	return seshatPackageSpec{
		SubjectCode: "EMAIL:DRAFT", Version: "1.0.0", Description: "Kernel 32 test email draft", Capability: "email.draft", Adapter: "gmail",
		RequiredFields: []string{"to", "subject", "body"}, OptionalFields: []string{"cc", "bcc"},
		PayloadRules: map[string]string{"to": "TO", "subject": "SUBJECT", "body": "body", "cc": "CC", "bcc": "BCC"},
		Returns:      []string{"draft_id", "status"}, PermissionClass: "nonpublic_composed_output", PermissionShapes: []string{"once"},
	}
}

func TestKernel32ApolloActionRoutingIsDurableAndTyped(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL32", "expire"))
	defer d.Close()
	result := futureReadyResult{
		Status: "complete", SourceCitations: []string{"[[80 Pantheon Raw/Text/whisper.md]]"},
		MeetingProposal:     map[string]any{"title": "Trinh planning", "start": "2026-08-25T10:00:00-07:00", "end": "2026-08-25T11:00:00-07:00"},
		DraftRecommendation: map[string]any{"to": "trinh@example.test", "subject": "Follow-up", "body": "Thanks for the conversation."},
	}
	drafts, err := d.routeApolloActions("report-kernel32", "/vault/70 Pantheon Reports/Apollo/report-kernel32.md", "occurrence-kernel32", result)
	if err != nil || len(drafts) != 2 {
		t.Fatalf("action drafts=%#v err=%v", drafts, err)
	}
	for _, draft := range drafts {
		if draft.State != actionStateProposed || draft.SourceReportID != "report-kernel32" || len(draft.SourceCitations) != 1 {
			t.Fatalf("unexpected action draft=%#v", draft)
		}
	}
	replayed, err := d.routeApolloActions("report-kernel32", "/vault/70 Pantheon Reports/Apollo/report-kernel32.md", "occurrence-kernel32", result)
	if err != nil || len(replayed) != 2 {
		t.Fatalf("replayed action drafts=%#v err=%v", replayed, err)
	}
	var count int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM action_drafts WHERE source_report_id = ?`, "report-kernel32").Scan(&count); err != nil || count != 2 {
		t.Fatalf("durable action count=%d err=%v", count, err)
	}
	inspection, err := d.actionDraftInspection()
	if err != nil || inspection["state_counts"] == nil {
		t.Fatalf("action inspection=%#v err=%v", inspection, err)
	}
}

func TestKernel32IncompleteActionBecomesClarification(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL32CLARIFY", "expire"))
	defer d.Close()
	drafts, err := d.routeApolloActions("report-clarify", "/vault/report.md", "occurrence-clarify", futureReadyResult{
		Status: "complete", MeetingProposal: map[string]any{"title": "Unscheduled conversation"},
	})
	if err != nil || len(drafts) != 1 || drafts[0].State != actionStateClarification || !strings.Contains(drafts[0].LastError, "start") {
		t.Fatalf("clarification draft=%#v err=%v", drafts, err)
	}
}

func TestKernel32AcceptEmailDraftStopsAtFutureAuthority(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL32ACCEPT", "expire"))
	defer d.Close()
	if err := d.seshat.addManifest(seshatManifest{SchemaVersion: 1, Plugin: seshatPlugin{Name: "kernel32-email", Version: "1.0.0"}, Packages: []seshatPackageSpec{kernel32EmailPackage()}}, "memory://kernel32-email"); err != nil {
		t.Fatal(err)
	}
	payload, _ := json.Marshal(map[string]any{"to": "trinh@example.test", "subject": "Follow-up", "body": "Thanks."})
	draft, err := d.createActionDraft(actionDraft{Kind: actionKindEmail, SourceReportID: "report-email", Summary: "Follow-up", Payload: payload})
	if err != nil || draft.State != actionStateProposed {
		t.Fatalf("created draft=%#v err=%v", draft, err)
	}
	accepted, err := d.acceptActionDraft(context.Background(), draft.ActionID)
	if err != nil {
		t.Fatalf("accept action draft: %v", err)
	}
	if accepted.FutureProposalID == "" || accepted.State != actionStateAwaitingAuth {
		t.Fatalf("accepted draft crossed too far or lost proposal: %#v", accepted)
	}
	var orders int
	if err := d.db.QueryRow(`SELECT COUNT(*) FROM orders WHERE recipient = 'gmail'`).Scan(&orders); err != nil {
		t.Fatal(err)
	}
	if orders != 0 {
		t.Fatalf("accepting a draft created %d Gmail orders without authority", orders)
	}
	replayed, err := d.createActionDraft(actionDraft{ActionID: draft.ActionID, Kind: actionKindEmail, State: actionStateProposed, SourceReportID: draft.SourceReportID, Summary: draft.Summary, Payload: draft.Payload})
	if err != nil || replayed.State != actionStateAwaitingAuth || replayed.FutureProposalID == "" {
		t.Fatalf("Apollo replay reset accepted action state: %#v err=%v", replayed, err)
	}
	if !d.mercuryCapabilityAvailable("gmail", "email.draft") {
		t.Fatal("generic email.draft is not recognized by Mercury")
	}
}

func TestKernel32ActionRoutesRespectControlBoundary(t *testing.T) {
	d := kernel15Daemon(t, janusTestPackage("TEST:KERNEL32HTTP", "expire"))
	defer d.Close()
	unauthorized := httptest.NewRecorder()
	d.Handler().ServeHTTP(unauthorized, httptest.NewRequest(http.MethodPost, "/v1/actions/not-real/accept", nil))
	if unauthorized.Code != http.StatusForbidden {
		t.Fatalf("unauthorized action accept status=%d body=%s", unauthorized.Code, unauthorized.Body.String())
	}
	page := httptest.NewRecorder()
	d.Handler().ServeHTTP(page, httptest.NewRequest(http.MethodGet, "/v1/actions", nil))
	if page.Code != http.StatusOK || !strings.Contains(page.Body.String(), "state_counts") {
		t.Fatalf("action inspection status=%d body=%s", page.Code, page.Body.String())
	}
}
