package pantheon

import (
	"strings"
	"testing"
)

func TestLibraReportIncludesCosmicForecastWithoutRenamingReport(t *testing.T) {
	run := libraReportRun{RunID: "libra-run-test", WindowStart: "2026-08-25", WindowEnd: "2026-08-25"}
	report := libraReportDefinition{Name: "Daily Brief", Recipe: libraReportRecipe{Outputs: []string{libraOutputWeekly}}}
	assembly := reportAssemblyReceipt{AssemblyID: "assembly-test", ManifestPath: "/tmp/manifest.json", SourceCount: 1}
	body, _ := renderLibraReport(run, report, assembly, []byte("A cited local report."))
	text := string(body)
	if !strings.Contains(text, "# Daily Brief") {
		t.Fatalf("report name changed: %s", text)
	}
	if !strings.Contains(text, "> Cosmic forecast: ") {
		t.Fatalf("cosmic forecast missing: %s", text)
	}
}
