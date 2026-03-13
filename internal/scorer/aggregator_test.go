package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/pkg/models"
)

func TestAggregate_Empty(t *testing.T) {
	a := Aggregate(nil)
	assert.Equal(t, 0, a.TotalLOC)
}

func TestAggregate_SingleRepo(t *testing.T) {
	r := RepoMetrics{
		LOC:                1000,
		AvgComplexity:      8.0,
		MaxComplexity:      25,
		AvgNesting:         2.5,
		MaxNesting:         6,
		DuplicationPct:     5.0,
		PctFilesOver500LOC: 10.0,
		SecretsCount:       2,
		CVECritical:        1,
		CVEHigh:            3,
		EstTestCoveragePct: 60.0,
		EnvVarCount:        8,
		IaCDetected:        true,
		CICDDetected:       true,
		HasReadme:          true,
		HasGitHistory:      true,
	}
	a := Aggregate([]RepoMetrics{r})
	assert.Equal(t, 1000, a.TotalLOC)
	assert.Equal(t, 8.0, a.AvgComplexity)
	assert.Equal(t, 25, a.MaxComplexity)
	assert.Equal(t, 2, a.SecretsCount)
	assert.True(t, a.IaCDetected)
}

func TestAggregate_TwoRepos_Counts(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, SecretsCount: 2, CVECritical: 1, CVEHigh: 0, CVEMedium: 3, LicenseIssueCount: 1, EnvVarCount: 5},
		{LOC: 2000, SecretsCount: 0, CVECritical: 0, CVEHigh: 2, CVEMedium: 1, LicenseIssueCount: 2, EnvVarCount: 10},
	}
	a := Aggregate(repos)
	assert.Equal(t, 3000, a.TotalLOC)
	assert.Equal(t, 2, a.SecretsCount)      // sum
	assert.Equal(t, 1, a.CVECritical)       // sum
	assert.Equal(t, 2, a.CVEHigh)           // sum
	assert.Equal(t, 4, a.CVEMedium)         // sum
	assert.Equal(t, 3, a.LicenseIssueCount) // sum
	assert.Equal(t, 15, a.EnvVarCount)      // sum
}

func TestAggregate_TwoRepos_LOCWeightedAvg(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, AvgComplexity: 10.0, DuplicationPct: 5.0, AvgNesting: 2.0, EstTestCoveragePct: 80.0},
		{LOC: 3000, AvgComplexity: 6.0, DuplicationPct: 15.0, AvgNesting: 3.0, EstTestCoveragePct: 40.0},
	}
	a := Aggregate(repos)
	// LOC-weighted: 10*0.25 + 6*0.75 = 2.5 + 4.5 = 7.0
	assert.InDelta(t, 7.0, a.AvgComplexity, 0.01)
	// LOC-weighted dup: 5*0.25 + 15*0.75 = 1.25 + 11.25 = 12.5
	assert.InDelta(t, 12.5, a.DuplicationPct, 0.01)
	// LOC-weighted nesting: 2*0.25 + 3*0.75 = 0.5 + 2.25 = 2.75
	assert.InDelta(t, 2.75, a.AvgNesting, 0.01)
	// LOC-weighted coverage: 80*0.25 + 40*0.75 = 20 + 30 = 50
	assert.InDelta(t, 50.0, a.EstTestCoveragePct, 0.01)
}

func TestAggregate_TwoRepos_MaxValues(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, MaxComplexity: 20, MaxNesting: 4},
		{LOC: 1000, MaxComplexity: 35, MaxNesting: 8},
	}
	a := Aggregate(repos)
	assert.Equal(t, 35, a.MaxComplexity) // global worst
	assert.Equal(t, 8, a.MaxNesting)     // global worst
}

func TestAggregate_TwoRepos_LastCommit(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, DaysSinceLastCommit: 30},
		{LOC: 1000, DaysSinceLastCommit: 90},
	}
	a := Aggregate(repos)
	assert.Equal(t, 30, a.DaysSinceLastCommit) // most recent
}

func TestAggregate_TwoRepos_CommitVelocity(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, AvgCommitsPerMonth: 10},
		{LOC: 1000, AvgCommitsPerMonth: 15},
	}
	a := Aggregate(repos)
	assert.InDelta(t, 25.0, a.AvgCommitsPerMonth, 0.01) // sum
}

func TestAggregate_TwoRepos_ActiveMonths(t *testing.T) {
	// Repo 1: months 0,1,2 active (bits 0b111 = 7)
	// Repo 2: months 2,3,4 active (bits 0b11100 = 28)
	// Union: months 0,1,2,3,4 active = 5 months
	repos := []RepoMetrics{
		{LOC: 1000, ActiveMonths: 7},
		{LOC: 1000, ActiveMonths: 28},
	}
	a := Aggregate(repos)
	assert.Equal(t, 5, a.ActiveMonths)
}

func TestAggregate_TwoRepos_BinaryFlags(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, IaCDetected: true, CICDDetected: false, MonitoringDetected: false, HasReadme: false, HasGitHistory: true},
		{LOC: 1000, IaCDetected: false, CICDDetected: true, MonitoringDetected: false, HasReadme: true, HasGitHistory: false},
	}
	a := Aggregate(repos)
	assert.True(t, a.IaCDetected)        // OR
	assert.True(t, a.CICDDetected)       // OR
	assert.False(t, a.MonitoringDetected) // neither
	assert.True(t, a.HasReadme)          // OR
	assert.True(t, a.HasGitHistory)      // OR
}

func TestAggregate_TwoRepos_DocDensity_WorstCase(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, DocDensity: models.DocDensityHigh},
		{LOC: 2000, DocDensity: models.DocDensityLow},
	}
	a := Aggregate(repos)
	assert.Equal(t, models.DocDensityLow, a.DocDensity) // worst-case
}

func TestAggregate_TwoRepos_DocDensity_BothHigh(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, DocDensity: models.DocDensityHigh},
		{LOC: 2000, DocDensity: models.DocDensityHigh},
	}
	a := Aggregate(repos)
	assert.Equal(t, models.DocDensityHigh, a.DocDensity)
}

func TestAggregate_NegativeDaysSinceLastCommit(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 1000, DaysSinceLastCommit: -5},
		{LOC: 1000, DaysSinceLastCommit: 10},
	}
	a := Aggregate(repos)
	assert.Equal(t, 0, a.DaysSinceLastCommit) // clamped -5 → 0, min(0,10) = 0
}

func TestAggregate_ZeroLOC(t *testing.T) {
	repos := []RepoMetrics{
		{LOC: 0, AvgComplexity: 10.0, SecretsCount: 5},
		{LOC: 0, AvgComplexity: 20.0, SecretsCount: 3},
	}
	a := Aggregate(repos)
	assert.Equal(t, 0, a.TotalLOC)
	assert.Equal(t, 8, a.SecretsCount) // sum still works
	assert.InDelta(t, 0.0, a.AvgComplexity, 0.01) // LOC-weighted with zero LOC → 0
}

func TestAggregate_MedianAgeMonths_Rounding(t *testing.T) {
	// Tests that MedianAgeMonths uses math.Round instead of truncation
	// Repo1 weight: 1000/3000 = 0.333, Repo2 weight: 2000/3000 = 0.667
	// Weighted age: 11*0.333 + 5*0.667 = 3.667 + 3.333 = 7.0
	repos := []RepoMetrics{
		{LOC: 1000, MedianAgeMonths: 11},
		{LOC: 2000, MedianAgeMonths: 5},
	}
	a := Aggregate(repos)
	assert.Equal(t, 7, a.MedianAgeMonths)
}
