package scorer

import (
	"fmt"

	"github.com/vettcode/scanner/pkg/models"
)

// RedFlagInput holds all metrics needed for red flag evaluation.
type RedFlagInput struct {
	SecretsCount        int
	CVECritical         int
	CVEHigh             int
	EstTestCoveragePct  float64
	DaysSinceLastCommit int
	UnmaintainedPct     float64
	CICDDetected        bool
	HasReadme           bool
	HasGitHistory       bool
}

// EvaluateRedFlags checks all threshold conditions and returns triggered red flags.
func EvaluateRedFlags(in RedFlagInput) models.RedFlags {
	var flags []models.RedFlag

	if in.SecretsCount > 0 {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagSecretsDetected,
			Detail:   fmt.Sprintf("%d hardcoded secrets found", in.SecretsCount),
			Severity: models.SeverityCritical,
		})
	}

	if in.CVECritical > 0 || in.CVEHigh > 0 {
		total := in.CVECritical + in.CVEHigh
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagCriticalCVE,
			Detail:   fmt.Sprintf("%d critical/high CVEs in dependencies", total),
			Severity: models.SeverityCritical,
		})
	}

	if in.EstTestCoveragePct < 0.01 {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagNoTests,
			Detail:   "0% est. test coverage — no test files found",
			Severity: models.SeverityHigh,
		})
	}

	if in.DaysSinceLastCommit > 180 {
		months := (in.DaysSinceLastCommit + 15) / 30
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagStaleRepo,
			Detail:   fmt.Sprintf("Last commit %d months ago", months),
			Severity: models.SeverityHigh,
		})
	}

	if in.UnmaintainedPct >= 50 {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagUnmaintainedDeps,
			Detail:   fmt.Sprintf("%.0f%% of dependencies unmaintained (2yr+)", in.UnmaintainedPct),
			Severity: models.SeverityHigh,
		})
	}

	if !in.CICDDetected {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagNoCICD,
			Detail:   "No CI/CD pipeline detected",
			Severity: models.SeverityMedium,
		})
	}

	if !in.HasReadme {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagNoReadme,
			Detail:   "No README found in any repository",
			Severity: models.SeverityMedium,
		})
	}

	if !in.HasGitHistory {
		flags = append(flags, models.RedFlag{
			Flag:     models.RedFlagNoGitHistory,
			Detail:   "No git history detected (Development Activity: N/A)",
			Severity: models.SeverityHigh,
		})
	}

	return models.RedFlags{
		Count: len(flags),
		Flags: flags,
	}
}
