package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/pkg/models"
)

func TestEvaluateRedFlags_None(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 0, rf.Count)
	assert.Empty(t, rf.Flags)
}

func TestEvaluateRedFlags_SecretsDetected(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		SecretsCount:       3,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagSecretsDetected, rf.Flags[0].Flag)
	assert.Equal(t, models.SeverityCritical, rf.Flags[0].Severity)
	assert.Contains(t, rf.Flags[0].Detail, "3")
}

func TestEvaluateRedFlags_CriticalCVE(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		CVECritical:        2,
		CVEHigh:            3,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagCriticalCVE, rf.Flags[0].Flag)
	assert.Contains(t, rf.Flags[0].Detail, "5") // 2+3
}

func TestEvaluateRedFlags_HighCVEOnly(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		CVEHigh:            1,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagCriticalCVE, rf.Flags[0].Flag)
}

func TestEvaluateRedFlags_NoTests(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 0,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagNoTests, rf.Flags[0].Flag)
	assert.Equal(t, models.SeverityHigh, rf.Flags[0].Severity)
}

func TestEvaluateRedFlags_TinyCoverage_NotFlagged(t *testing.T) {
	// Anything >= 0.01 should NOT trigger no_tests
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 1.0,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 0, rf.Count)
}

func TestEvaluateRedFlags_StaleRepo(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		DaysSinceLastCommit: 365,
		EstTestCoveragePct:  50,
		CICDDetected:        true,
		HasReadme:           true,
		HasGitHistory:       true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagStaleRepo, rf.Flags[0].Flag)
	assert.Contains(t, rf.Flags[0].Detail, "12") // ~12 months
}

func TestEvaluateRedFlags_180DaysExactly_NotStale(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		DaysSinceLastCommit: 180,
		EstTestCoveragePct:  50,
		CICDDetected:        true,
		HasReadme:           true,
		HasGitHistory:       true,
	})
	assert.Equal(t, 0, rf.Count) // exactly 180 is NOT > 180
}

func TestEvaluateRedFlags_UnmaintainedDeps(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		UnmaintainedPct:    55,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagUnmaintainedDeps, rf.Flags[0].Flag)
}

func TestEvaluateRedFlags_NoCICD(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 50,
		CICDDetected:       false,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagNoCICD, rf.Flags[0].Flag)
	assert.Equal(t, models.SeverityMedium, rf.Flags[0].Severity)
}

func TestEvaluateRedFlags_NoReadme(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          false,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagNoReadme, rf.Flags[0].Flag)
}

func TestEvaluateRedFlags_NoGitHistory(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      false,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagNoGitHistory, rf.Flags[0].Flag)
	assert.Equal(t, models.SeverityHigh, rf.Flags[0].Severity)
}

func TestEvaluateRedFlags_Multiple(t *testing.T) {
	rf := EvaluateRedFlags(RedFlagInput{
		SecretsCount:        2,
		CVECritical:         1,
		EstTestCoveragePct:  0,
		DaysSinceLastCommit: 200,
		UnmaintainedPct:     60,
		CICDDetected:        false,
		HasReadme:           false,
		HasGitHistory:       false,
	})
	assert.Equal(t, 8, rf.Count) // all flags triggered
}

func TestEvaluateRedFlags_SecurityCombo(t *testing.T) {
	// Secrets + CVEs should both trigger independently
	rf := EvaluateRedFlags(RedFlagInput{
		SecretsCount:       5,
		CVECritical:        3,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 2, rf.Count)
	// Both should be critical
	for _, f := range rf.Flags {
		assert.Equal(t, models.SeverityCritical, f.Severity)
	}
}

func TestEvaluateRedFlags_ProcessCombo(t *testing.T) {
	// No tests + no CI/CD + no readme = process issues
	rf := EvaluateRedFlags(RedFlagInput{
		EstTestCoveragePct: 0,
		CICDDetected:       false,
		HasReadme:          false,
		HasGitHistory:      true,
	})
	assert.Equal(t, 3, rf.Count)
}

func TestEvaluateRedFlags_ExactThresholds(t *testing.T) {
	// DaysSinceLastCommit exactly 180 should NOT trigger stale
	rf := EvaluateRedFlags(RedFlagInput{
		DaysSinceLastCommit: 180,
		EstTestCoveragePct:  50,
		CICDDetected:        true,
		HasReadme:           true,
		HasGitHistory:       true,
	})
	assert.Equal(t, 0, rf.Count)

	// DaysSinceLastCommit 181 SHOULD trigger stale
	rf = EvaluateRedFlags(RedFlagInput{
		DaysSinceLastCommit: 181,
		EstTestCoveragePct:  50,
		CICDDetected:        true,
		HasReadme:           true,
		HasGitHistory:       true,
	})
	assert.Equal(t, 1, rf.Count)
	assert.Equal(t, models.RedFlagStaleRepo, rf.Flags[0].Flag)
}

func TestEvaluateRedFlags_UnmaintainedExactThreshold(t *testing.T) {
	// 49% should NOT trigger (threshold is >= 50)
	rf := EvaluateRedFlags(RedFlagInput{
		UnmaintainedPct:    49,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 0, rf.Count)

	// 50% exactly SHOULD trigger (>= 50)
	rf = EvaluateRedFlags(RedFlagInput{
		UnmaintainedPct:    50,
		EstTestCoveragePct: 50,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	})
	assert.Equal(t, 1, rf.Count)
}
