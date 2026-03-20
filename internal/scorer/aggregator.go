package scorer

import (
	"math"

	"github.com/vettcode/scanner/pkg/models"
)

// docDensityRank returns a numeric rank for worst-case comparison.
func docDensityRank(d models.DocDensity) int {
	switch d {
	case models.DocDensityHigh:
		return 2
	case models.DocDensityMedium:
		return 1
	default:
		return 0
	}
}

// RepoMetrics holds the raw per-repo metrics needed for aggregation.
type RepoMetrics struct {
	LOC                int
	AvgComplexity      float64
	MaxComplexity      int
	AvgNesting         float64
	MaxNesting         int
	DuplicationPct     float64
	PctFilesOver500LOC float64
	SecretsCount       int
	CVECritical        int
	CVEHigh            int
	CVEMedium          int
	CVELow             int
	CVECriticalTrans   int // transitive (indirect) dependency CVEs
	CVEHighTrans       int
	CVEMediumTrans     int
	CVELowTrans        int
	LicenseIssueCount  int
	EstTestCoveragePct float64
	EnvVarCount        int
	DocDensity         models.DocDensity
	MedianAgeMonths    int
	UnmaintainedPct    float64
	DaysSinceLastCommit int
	AvgCommitsPerMonth float64
	ActiveMonths       int // bit set of months (0-11)
	IaCDetected        bool
	CICDDetected       bool
	MonitoringDetected bool
	HasReadme          bool
	HasGitHistory      bool
}

// AggregatedMetrics is the result of aggregating multiple repos.
type AggregatedMetrics struct {
	TotalLOC           int
	AvgComplexity      float64
	MaxComplexity      int
	AvgNesting         float64
	MaxNesting         int
	DuplicationPct     float64
	PctFilesOver500LOC float64
	SecretsCount       int
	CVECritical        int
	CVEHigh            int
	CVEMedium          int
	CVELow             int
	CVECriticalTrans   int // transitive (indirect) dependency CVEs
	CVEHighTrans       int
	CVEMediumTrans     int
	CVELowTrans        int
	LicenseIssueCount  int
	EstTestCoveragePct float64
	EnvVarCount        int
	DocDensity         models.DocDensity
	MedianAgeMonths    int
	UnmaintainedPct    float64
	DaysSinceLastCommit int
	AvgCommitsPerMonth float64
	ActiveMonths       int
	IaCDetected        bool
	CICDDetected       bool
	MonitoringDetected bool
	HasReadme          bool
	HasGitHistory      bool
}

// Aggregate combines metrics from multiple repositories following the aggregation rules:
// - Counts: sum
// - Percentages/averages: LOC-weighted average
// - Max values: global worst
// - Last commit: most recent (smallest days_since)
// - Commit velocity: sum
// - Active months: union (capped at 12)
// - Binary flags: OR logic
// - Doc density: worst-case (minimum)
func Aggregate(repos []RepoMetrics) AggregatedMetrics {
	if len(repos) == 0 {
		return AggregatedMetrics{}
	}

	var a AggregatedMetrics
	totalLOC := 0
	activeMonthBits := 0
	minDaysSinceCommit := -1
	worstDocRank := 3 // higher than any valid rank

	for _, r := range repos {
		totalLOC += r.LOC

		// Counts: sum
		a.SecretsCount += r.SecretsCount
		a.CVECritical += r.CVECritical
		a.CVEHigh += r.CVEHigh
		a.CVEMedium += r.CVEMedium
		a.CVELow += r.CVELow
		a.CVECriticalTrans += r.CVECriticalTrans
		a.CVEHighTrans += r.CVEHighTrans
		a.CVEMediumTrans += r.CVEMediumTrans
		a.CVELowTrans += r.CVELowTrans
		a.LicenseIssueCount += r.LicenseIssueCount
		a.EnvVarCount += r.EnvVarCount

		// Max: global worst
		if r.MaxComplexity > a.MaxComplexity {
			a.MaxComplexity = r.MaxComplexity
		}
		if r.MaxNesting > a.MaxNesting {
			a.MaxNesting = r.MaxNesting
		}

		// Last commit: most recent (clamp negative to 0)
		days := r.DaysSinceLastCommit
		if days < 0 {
			days = 0
		}
		if minDaysSinceCommit < 0 || days < minDaysSinceCommit {
			minDaysSinceCommit = days
		}

		// Commit velocity: sum
		a.AvgCommitsPerMonth += r.AvgCommitsPerMonth

		// Active months: union (mask to 12 bits)
		activeMonthBits |= r.ActiveMonths & 0xFFF

		// Doc density: worst-case (minimum rank)
		rank := docDensityRank(r.DocDensity)
		if rank < worstDocRank {
			worstDocRank = rank
			a.DocDensity = r.DocDensity
		}

		// Binary flags: OR
		a.IaCDetected = a.IaCDetected || r.IaCDetected
		a.CICDDetected = a.CICDDetected || r.CICDDetected
		a.MonitoringDetected = a.MonitoringDetected || r.MonitoringDetected
		a.HasReadme = a.HasReadme || r.HasReadme
		a.HasGitHistory = a.HasGitHistory || r.HasGitHistory
	}

	a.TotalLOC = totalLOC
	if minDaysSinceCommit >= 0 {
		a.DaysSinceLastCommit = minDaysSinceCommit
	}

	// Count set bits for active months
	bits := activeMonthBits
	for bits > 0 {
		a.ActiveMonths += bits & 1
		bits >>= 1
	}

	// LOC-weighted averages
	if totalLOC > 0 {
		var weightedAge float64
		for _, r := range repos {
			w := float64(r.LOC) / float64(totalLOC)
			a.AvgComplexity += r.AvgComplexity * w
			a.AvgNesting += r.AvgNesting * w
			a.DuplicationPct += r.DuplicationPct * w
			a.PctFilesOver500LOC += r.PctFilesOver500LOC * w
			a.EstTestCoveragePct += r.EstTestCoveragePct * w
			a.UnmaintainedPct += r.UnmaintainedPct * w
			weightedAge += float64(r.MedianAgeMonths) * w
		}
		a.MedianAgeMonths = int(math.Round(weightedAge))
	}

	return a
}
