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
	} {
		if !strings.Contains(body, section) {
			t.Fatalf("rendered Apollo report missing %q:\n%s", section, body)
		}
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
}
