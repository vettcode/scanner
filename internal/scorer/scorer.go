// Package scorer implements the VettCode scoring methodology.
// It converts raw analyzer metrics into category scores (0-100)
// and letter grades per the scoring specification.
package scorer

import (
	"math"

	"github.com/vettcode/scanner/pkg/models"
)

// clamp restricts a value to [0, 100].
func clamp(v float64) float64 {
	return math.Max(0, math.Min(100, v))
}

// --- 3.1 Code Maintainability (20% of overall) ---

// MaintainabilityInput holds the raw metrics needed for maintainability scoring.
type MaintainabilityInput struct {
	AvgComplexity      float64
	DuplicationPct     float64
	AvgNesting         float64
	PctFilesOver500LOC float64
}

// ScoreMaintainability computes the maintainability category score (0-100).
func ScoreMaintainability(in MaintainabilityInput) float64 {
	complexity := clamp(100 - (in.AvgComplexity-5)*4)  // 40%
	duplication := clamp(100 - in.DuplicationPct*3)     // 30%
	nesting := clamp(100 - (in.AvgNesting-1.5)*20)     // 15%
	fileSize := clamp(100 - in.PctFilesOver500LOC*2)    // 15%

	return complexity*0.40 + duplication*0.30 + nesting*0.15 + fileSize*0.15
}

// --- 3.2 Security Posture (25% of overall) ---

// SecurityInput holds the raw metrics needed for security scoring.
type SecurityInput struct {
	SecretsCount     int
	CVECritical      int
	CVEHigh          int
	CVEMedium        int
	CVELow           int
	LicenseIssueCount int
}

// ScoreSecurity computes the security category score (0-100).
func ScoreSecurity(in SecurityInput) float64 {
	var secrets float64
	if in.SecretsCount == 0 {
		secrets = 100
	} // else 0

	cves := clamp(100 - float64(in.CVECritical)*30 - float64(in.CVEHigh)*15 -
		float64(in.CVEMedium)*5 - float64(in.CVELow)*1)

	licenses := clamp(100 - float64(in.LicenseIssueCount)*25)

	return secrets*0.35 + cves*0.45 + licenses*0.20
}

// --- 3.3 Handoff Readiness (20% of overall) ---

// HandoffInput holds the raw metrics needed for handoff scoring.
type HandoffInput struct {
	EstTestCoveragePct float64
	DocDensity         models.DocDensity
	EnvVarCount        int
}

// ScoreHandoff computes the handoff readiness category score (0-100).
func ScoreHandoff(in HandoffInput) float64 {
	coverage := clamp(in.EstTestCoveragePct * 1.25) // 50%

	var docScore float64 // 25%
	switch in.DocDensity {
	case models.DocDensityHigh:
		docScore = 90
	case models.DocDensityMedium:
		docScore = 60
	default:
		docScore = 30
	}

	envScore := clamp(100 - math.Max(0, float64(in.EnvVarCount-5))*3) // 25%

	return coverage*0.50 + docScore*0.25 + envScore*0.25
}

// --- 3.4 Dependency Health (10% of overall) ---

// DependencyHealthInput holds the raw metrics needed for dependency health scoring.
type DependencyHealthInput struct {
	MedianAgeMonths int
	UnmaintainedPct float64
}

// ScoreDependencyHealth computes the dependency health category score (0-100).
func ScoreDependencyHealth(in DependencyHealthInput) float64 {
	age := clamp(100 - math.Max(0, float64(in.MedianAgeMonths-6))*2.5)  // 50%
	unmaintained := clamp(100 - in.UnmaintainedPct*4)                    // 50%

	return age*0.50 + unmaintained*0.50
}

// --- 3.5 Development Activity (15% of overall) ---

// ActivityInput holds the raw metrics needed for activity scoring.
type ActivityInput struct {
	DaysSinceLastCommit int
	AvgCommitsPerMonth  float64
	ActiveMonths        int // months with commits in the observation window
	RepoAgeMonths       int // months since first commit (capped at 12)
}

// ScoreActivity computes the development activity category score (0-100).
// Velocity uses a diminishing-returns curve (22 * sqrt) so that mature projects
// with 5-10 commits/month score well, while still rewarding higher velocity.
// Consistency is scaled by repo age so new projects aren't penalized for not
// existing for 12 months yet.
func ScoreActivity(in ActivityInput) float64 {
	recency := clamp(100 - float64(in.DaysSinceLastCommit)*0.55)             // 40%
	velocity := clamp(22 * math.Sqrt(float64(in.AvgCommitsPerMonth)))        // 30%

	// Scale consistency by repo age: a 1-month-old repo with 1 active month = 100%
	window := 12
	if in.RepoAgeMonths > 0 && in.RepoAgeMonths < 12 {
		window = in.RepoAgeMonths
	}
	consistency := clamp(float64(in.ActiveMonths) / float64(window) * 100)   // 30%

	return recency*0.40 + velocity*0.30 + consistency*0.30
}

// --- 3.6 SRE & Infrastructure (10% of overall) ---

// InfraInput holds the raw metrics needed for infrastructure scoring.
type InfraInput struct {
	IaCDetected        bool
	CICDDetected       bool
	MonitoringDetected bool
}

// ScoreInfra computes the SRE & infrastructure category score (0-100).
func ScoreInfra(in InfraInput) float64 {
	var iac, cicd, monitoring float64
	if in.IaCDetected {
		iac = 100
	}
	if in.CICDDetected {
		cicd = 100
	}
	if in.MonitoringDetected {
		monitoring = 100
	}

	return iac*0.30 + cicd*0.60 + monitoring*0.10
}
