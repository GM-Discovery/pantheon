package pantheon

import (
	"strings"
	"testing"
)

func TestFormatApolloStructuredReportRendersRequestedSections(t *testing.T) {
	body := formatApolloStructuredReport(futureReadyResult{
		Summary:           "The meeting produced one follow-up.",
		ActionItems:       []map[string]any{{"action": "Send the draft", "owner": "Trinh"}},
		Commitments:       []map[string]any{{"commitment": "Review it tomorrow"}},
		Decisions:         []map[string]any{{"decision": "Use the local path"}},
		UnresolvedThreads: []map[string]any{{"question": "Which date should be used?"}},
		Facts:             []map[string]any{{"fact": "The draft is stored locally", "source": "[[80 Pantheon Raw/example.md]]"}},
		Recommendations:   []map[string]any{{"recommendation": "Review the draft"}},
		Uncertainties:     []string{"Speaker identity was not verified."},
	}, "fallback answer")
	for _, section := range []string{
		"## Concise summary",
		"## Concrete requests and action items",
		"## Commitments and deadlines",
		"## Decisions",
		"## Unresolved questions",
		"## Uncertainties and gaps",
		"Send the draft",
		"Speaker identity was not verified.",
		"The draft is stored locally",
		"**Recommendation:** Review the draft",
	} {
		if !strings.Contains(body, section) {
			t.Fatalf("rendered Apollo report missing %q:\n%s", section, body)
		}
	}
	if strings.Contains(body, "{\"") || strings.Contains(body, "```json") {
		t.Fatalf("human Apollo report contains machine-oriented JSON:")
	}
}

func TestHumanApolloReportBodyRemovesFrontmatterAndMachineReceipt(t *testing.T) {
	body := []byte("---\nreport_id: report-1\n---\n\n# Apollo Report report-1\n\n## Request\n\nSummarize this.\n\n## Status\n\ncomplete\n\n## Human-readable result\n\n## Concise summary\n\nA readable finding.\n\n## Evidence\n\n- Source: [[80 Pantheon Raw/example.md]]\n\n## Future-ready result\n\n```json\n{\"summary\":\"A readable finding.\"}\n```\n")
	rendered := humanApolloReportBody(body)
	if !strings.Contains(rendered, "A readable finding.") {
		t.Fatalf("human report lost the readable result: %s", rendered)
	}
	if strings.Contains(rendered, "report_id:") || strings.Contains(rendered, "```json") || strings.Contains(rendered, "Future-ready") {
		t.Fatalf("human report retained machine receipt content: %s", rendered)
	}
}

func TestLibraDefaultSourcePolicyExcludesGeneratedReports(t *testing.T) {
	recipe := libraReportRecipe{SourceFamilies: []string{"all"}}
	for _, path := range []string{
		"70 Pantheon Reports/Apollo/report-old.md",
		"70 Pantheon Reports/Weekly/2026-08-24 Weekly Report.md",
		"10 Operations/Situation Briefs/2026-08-25 Situation Brief.md",
	} {
		if libraRecipeAllowsPath(recipe, path) {
			t.Fatalf("default Libra source policy admitted generated report %q", path)
		}
	}
	if !libraRecipeAllowsPath(recipe, "80 Pantheon Raw/Text/2026/08/transcript.md") {
		t.Fatal("default Libra source policy excluded a raw transcript")
	}
	if !libraRecipeAllowsPath(recipe, "70 Pantheon Reports/Integrated Events/2026-08-25-event.md") {
		t.Fatal("default Libra source policy excluded an integrated event")
	}
	if !libraRecipeAllowsPath(libraReportRecipe{SourceFamilies: []string{"all", "apollo_reports"}}, "70 Pantheon Reports/Apollo/report-old.md") {
		t.Fatal("explicit Apollo report source family was not honored")
	}
	if !libraRecipeAllowsMail(libraReportRecipe{SourceFamilies: []string{"all"}}) {
		t.Fatal("default Libra source policy excluded understood mail")
	}
	if libraRecipeAllowsMail(libraReportRecipe{SourceFamilies: []string{"transcripts"}}) {
		t.Fatal("transcript-only Libra source policy admitted mail")
	}
}
