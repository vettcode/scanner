package output

import (
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/vettcode/scanner/pkg/models"
)

// TerminalFormatter renders a ScanResult to the terminal matching
// the format specified in Section 7.5 of the design doc.
type TerminalFormatter struct {
	Color      *ColorConfig
	OutputPath string
	Duration   time.Duration
}

// Format writes the formatted scan result to w.
func (f *TerminalFormatter) Format(w io.Writer, result *models.ScanResult) {
	c := f.Color

	// Header
	fmt.Fprintf(w, "\n%s\n", c.bold("VettCode Scan Complete -- "+result.Timestamp))
	fmt.Fprintln(w, "=====================================")
	fmt.Fprintf(w, "Repositories:  %d scanned\n", result.RepoCount)

	// Languages
	if len(result.Repositories) > 0 {
		langs := aggregateLanguages(result.Repositories)
		if len(langs) > 0 {
			fmt.Fprintf(w, "Languages:     %s\n", langs)
		}
	}

	fmt.Fprintf(w, "Total LOC:     %s\n", formatNumber(result.TotalLOC))

	// Tech stack
	if len(result.TechStack.Frameworks) > 0 || len(result.TechStack.Databases) > 0 || len(result.TechStack.ExternalServices) > 0 {
		var items []string
		items = append(items, result.TechStack.Frameworks...)
		items = append(items, result.TechStack.Databases...)
		items = append(items, result.TechStack.ExternalServices...)
		fmt.Fprintf(w, "Tech Stack:    %s\n", strings.Join(items, ", "))
	}
	if len(result.TechStack.Runtimes) > 0 {
		fmt.Fprintf(w, "Runtimes:      %s\n", strings.Join(result.TechStack.Runtimes, ", "))
	}

	if f.Duration > 0 {
		fmt.Fprintf(w, "Scan Duration: %s\n", formatDuration(f.Duration))
	}

	// Category sections — ordered by importance to buyers
	fmt.Fprintln(w)
	f.formatHandoff(w, result.Metrics.HandoffReadiness)
	fmt.Fprintln(w)
	f.formatSecurity(w, result.Metrics.Security)
	fmt.Fprintln(w)
	f.formatMaintainability(w, result.Metrics.Maintainability)
	fmt.Fprintln(w)
	f.formatDependencyHealth(w, result.Metrics.DependencyHealth)
	fmt.Fprintln(w)
	f.formatInfrastructure(w, result.Detection.Infrastructure, result.TechStack.ExternalServices)
	fmt.Fprintln(w)
	f.formatAIDetection(w, result.Detection.AI)
	fmt.Fprintln(w)
	f.formatActivity(w, result.Activity)

	// Overall grade
	fmt.Fprintln(w)
	overallGrade := "N/A"
	if result.Summary.OverallGrade != nil {
		overallGrade = string(*result.Summary.OverallGrade)
	}
	fmt.Fprintf(w, "%-25s%s\n", c.bold("OVERALL GRADE"), c.gradeColor(overallGrade))

	// Footer
	fmt.Fprintln(w)
	fmt.Fprintln(w, "=====================================")
	if f.OutputPath != "" {
		fmt.Fprintf(w, "%s  %s\n", c.bold("Full results:"), f.OutputPath)
		fmt.Fprintf(w, "%s  %s\n", c.bold("Upload report:"), "https://platform.vettcode.com/upload")
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, c.yellow("Ready to close your deal faster?"))
	fmt.Fprintln(w, c.yellow("Upload this scan to platform.vettcode.com to get a"))
	fmt.Fprintln(w, c.yellow("signed, buyer-ready report — builds trust and speeds"))
	fmt.Fprintln(w, c.yellow("up due diligence."))
	fmt.Fprintln(w, "=====================================")
	fmt.Fprintln(w)
}

func (f *TerminalFormatter) formatMaintainability(w io.Writer, m *models.Maintainability) {
	c := f.Color
	if m == nil {
		fmt.Fprintf(w, "%-25s%s\n", c.bold("MAINTAINABILITY"), c.gray("N/A"))
		return
	}
	grade := gradeStr(m.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("MAINTAINABILITY"), c.gradeColor(grade))
	fmt.Fprintf(w, "  Avg Complexity:        %.1f\n", m.CyclomaticComplexity.Avg)
	fmt.Fprintf(w, "  Code Duplication:      %.1f%%\n", m.DuplicationPct)
	fmt.Fprintf(w, "  Hotspot Files:         %d\n", m.HotspotCount)
	for i, h := range m.HotspotFiles {
		if i >= 5 {
			break
		}
		displayPath := h.Path
		if displayPath == "" {
			displayPath = h.FileHash
		}
		fmt.Fprintf(w, "    %d. %s/%s    complexity: %d  LOC: %d\n",
			i+1, h.Repo, displayPath, h.Complexity, h.LOC)
	}
}

func (f *TerminalFormatter) formatSecurity(w io.Writer, s *models.Security) {
	c := f.Color
	if s == nil {
		fmt.Fprintf(w, "%-25s%s\n", c.bold("SECURITY"), c.gray("N/A"))
		return
	}
	grade := gradeStr(s.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("SECURITY"), c.gradeColor(grade))

	secretsStr := fmt.Sprintf("%d", s.SecretsFound)
	if s.SecretsFound > 0 {
		secretsStr = c.red(secretsStr)
	}
	fmt.Fprintf(w, "  Secrets Found:         %s\n", secretsStr)
	if s.SecretsFound > 0 {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Rotate exposed keys and remove hardcoded credentials."))
	}
	for i, sf := range s.SecretFindings {
		if i >= 5 {
			remaining := len(s.SecretFindings) - 5
			fmt.Fprintf(w, "    ... and %d more\n", remaining)
			break
		}
		fmt.Fprintf(w, "    %d. %s:%d  %s\n", i+1, sf.Path, sf.Line, c.red(sf.Name))
	}

	totalCVEs := s.CVESummary.Critical + s.CVESummary.High + s.CVESummary.Medium + s.CVESummary.Low
	if totalCVEs > 0 {
		breakdown := fmt.Sprintf("%d critical, %d high, %d medium, %d low",
			s.CVESummary.Critical, s.CVESummary.High, s.CVESummary.Medium, s.CVESummary.Low)
		fmt.Fprintf(w, "  Known CVEs:            %d (%s)\n", totalCVEs, breakdown)
		for i, cve := range s.CVEs {
			if i >= 5 {
				remaining := len(s.CVEs) - 5
				fmt.Fprintf(w, "    ... and %d more\n", remaining)
				break
			}
			fix := cve.FixedIn
			if fix == "" {
				fix = "no fix available"
			} else {
				fix = "fix: " + fix
			}
			fmt.Fprintf(w, "    %d. %s  %s@%s  (%s)\n",
				i+1, c.red(string(cve.Severity)), cve.Package, cve.CurrentVersion, fix)
		}
	} else {
		fmt.Fprintf(w, "  Known CVEs:            %d\n", totalCVEs)
	}
	if s.CVESummary.Critical > 0 {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Update dependencies with known critical vulnerabilities."))
	}

	fmt.Fprintf(w, "  Outdated Deps:         %d/%d\n", s.OutdatedDeps.Outdated, s.OutdatedDeps.Total)
	fmt.Fprintf(w, "  License Issues:        %d\n", s.LicenseIssueCount)
}

func (f *TerminalFormatter) formatDependencyHealth(w io.Writer, d *models.DependencyHealth) {
	c := f.Color
	if d == nil {
		fmt.Fprintf(w, "%-25s%s\n", c.bold("DEPENDENCY HEALTH"), c.gray("N/A"))
		return
	}
	grade := gradeStr(d.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("DEPENDENCY HEALTH"), c.gradeColor(grade))
	fmt.Fprintf(w, "  Median Dep Age:        %d months\n", d.MedianAgeMonths)
	fmt.Fprintf(w, "  Unmaintained (2yr+):   %.0f%% (%d)\n", d.UnmaintainedPct, d.UnmaintainedCount)
	if d.UnmaintainedPct >= 50 {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Updating outdated dependencies improves Dependency Health."))
	}
	if d.Oldest != nil {
		fmt.Fprintf(w, "  Oldest:                %s (%.1f years)\n", d.Oldest.Package, d.Oldest.AgeYears)
	}
}

func (f *TerminalFormatter) formatActivity(w io.Writer, a *models.Activity) {
	c := f.Color
	if a == nil {
		fmt.Fprintf(w, "%-25s%s\n", c.bold("DEVELOPMENT ACTIVITY"), c.gray("N/A"))
		return
	}
	grade := gradeStr(a.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("DEVELOPMENT ACTIVITY"), c.gradeColor(grade))
	fmt.Fprintf(w, "  Last Commit:           %s (%d days ago)\n", a.LastCommitDate, a.DaysSinceLastCommit)
	if a.DaysSinceLastCommit > 180 {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Recent commit activity improves your Activity score."))
	}
	fmt.Fprintf(w, "  Commit Velocity:       %.0f/mo avg (last 12 months)\n", a.CommitVelocity.AvgPerMonth)
	fmt.Fprintf(w, "  Trend:                 %s\n", titleCase(string(a.CommitVelocity.Trend)))
	fmt.Fprintf(w, "  Active Months:         %d of 12\n", a.ActiveMonths)
}

func (f *TerminalFormatter) formatAIDetection(w io.Writer, ai models.AIDetection) {
	c := f.Color
	fmt.Fprintln(w, c.bold("AI DETECTION"))
	fmt.Fprintf(w, "  LLM API:               %s\n", c.yesNo(ai.LLMAPI, ai.LLMProvider))
	fmt.Fprintf(w, "  Vector DB:             %s\n", c.yesNo(ai.VectorDatabase, ai.VectorDBName))
	fmt.Fprintf(w, "  RAG Pipeline:          %s\n", c.yesNo(ai.RAGPipeline, ""))
	fmt.Fprintf(w, "  MCP Servers:           %s\n", c.yesNo(ai.MCPServers, ""))
	fmt.Fprintf(w, "  Proprietary Data:      %s\n", c.yesNo(ai.ProprietaryDataset, ""))
}

func (f *TerminalFormatter) formatInfrastructure(w io.Writer, infra models.InfrastructureDetection, externalServices []string) {
	c := f.Color
	grade := gradeStr(infra.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("INFRASTRUCTURE"), c.gradeColor(grade))
	fmt.Fprintf(w, "  IaC:                   %s\n", c.yesNo(infra.IaCDetected, strings.Join(infra.IaCTypes, ", ")))
	if !infra.IaCDetected {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 IaC in a separate repo? Add it to the scan scope."))
	}
	fmt.Fprintf(w, "  CI/CD:                 %s\n", c.yesNo(infra.CICDDetected, infra.CICDProvider))
	if !infra.CICDDetected {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 CI/CD in a separate repo? Add it to the scan scope."))
	}
	fmt.Fprintf(w, "  Monitoring:            %s\n", c.yesNo(infra.MonitoringDetected, strings.Join(infra.MonitoringTools, ", ")))
	if len(externalServices) > 0 {
		fmt.Fprintf(w, "  External Services:     %s\n", strings.Join(externalServices, ", "))
	}
}

func (f *TerminalFormatter) formatHandoff(w io.Writer, h *models.HandoffReadiness) {
	c := f.Color
	if h == nil {
		fmt.Fprintf(w, "%-25s%s\n", c.bold("HANDOFF READINESS"), c.gray("N/A"))
		return
	}
	grade := gradeStr(h.Grade)
	fmt.Fprintf(w, "%-25s%s\n", c.bold("HANDOFF READINESS"), c.gradeColor(grade))
	fmt.Fprintf(w, "  Est. Test Coverage:    %.0f%%\n", h.EstTestCoveragePct)
	if h.EstTestCoveragePct < 1 {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Even minimal test coverage significantly improves Handoff Readiness."))
	}
	fmt.Fprintf(w, "  Doc Density:           %s\n", titleCase(string(h.DocDensity)))
	fmt.Fprintf(w, "  Env Vars:              %d\n", h.EnvVarCount)
	if !h.HasReadme {
		fmt.Fprintf(w, "  %s\n", c.yellow("💡 Adding a README helps buyers understand your project."))
	}
}

// Helpers

func gradeStr(g *models.Grade) string {
	if g == nil {
		return "N/A"
	}
	return string(*g)
}

func aggregateLanguages(repos []models.Repository) string {
	totalLOC := 0
	langLOC := make(map[string]int)
	for _, r := range repos {
		totalLOC += r.LOC
		for lang, pct := range r.Languages {
			langLOC[lang] += int(pct * float64(r.LOC) / 100.0)
		}
	}
	if totalLOC == 0 {
		return ""
	}

	type langPct struct {
		lang string
		pct  float64
	}
	var sorted []langPct
	for lang, loc := range langLOC {
		sorted = append(sorted, langPct{lang, float64(loc) / float64(totalLOC) * 100})
	}
	// Sort descending by percentage
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].pct > sorted[i].pct {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	parts := make([]string, 0, len(sorted))
	for _, lp := range sorted {
		if lp.pct < 1 {
			continue
		}
		parts = append(parts, fmt.Sprintf("%s (%.0f%%)", lp.lang, lp.pct))
	}
	return strings.Join(parts, ", ")
}

func formatNumber(n int) string {
	if n < 1000 {
		return fmt.Sprintf("%d", n)
	}
	s := fmt.Sprintf("%d", n)
	l := len(s)
	result := make([]byte, 0, l+(l-1)/3)
	for i, c := range s {
		if i > 0 && (l-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result)
}

func titleCase(s string) string {
	if s == "" {
		return s
	}
	r := []rune(s)
	if r[0] >= 'a' && r[0] <= 'z' {
		r[0] -= 32
	}
	return string(r)
}

func formatDuration(d time.Duration) string {
	if d < time.Second {
		return fmt.Sprintf("%dms", d.Milliseconds())
	}
	if d < time.Minute {
		return fmt.Sprintf("%.1fs", d.Seconds())
	}
	m := int(d.Minutes())
	s := int(d.Seconds()) % 60
	return fmt.Sprintf("%dm%ds", m, s)
}
