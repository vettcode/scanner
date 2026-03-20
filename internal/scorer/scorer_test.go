package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/pkg/models"
)

// --- Grade conversion tests (SC-041) ---

func TestScoreToGrade(t *testing.T) {
	tests := []struct {
		score float64
		want  models.Grade
	}{
		{100, models.GradeA},
		{93, models.GradeA},
		{92, models.GradeAM},
		{90, models.GradeAM},
		{89, models.GradeBP},
		{87, models.GradeBP},
		{86, models.GradeB},
		{83, models.GradeB},
		{82, models.GradeBM},
		{80, models.GradeBM},
		{79, models.GradeCP},
		{77, models.GradeCP},
		{76, models.GradeC},
		{73, models.GradeC},
		{72, models.GradeCM},
		{70, models.GradeCM},
		{69, models.GradeDP},
		{67, models.GradeDP},
		{66, models.GradeD},
		{63, models.GradeD},
		{62, models.GradeDM},
		{60, models.GradeDM},
		{59, models.GradeF},
		{0, models.GradeF},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, ScoreToGrade(tt.score), "score %.0f", tt.score)
	}
}

// --- Maintainability scorer tests (SC-040) ---

func TestScoreMaintainability_Perfect(t *testing.T) {
	score := ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      5.0,  // → 100
		DuplicationPct:     0.0,  // → 100
		AvgNesting:         1.5,  // → 100
		PctFilesOver500LOC: 0.0,  // → 100
	})
	assert.InDelta(t, 100.0, score, 0.01)
}

func TestScoreMaintainability_Moderate(t *testing.T) {
	score := ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      10.0, // → 80
		DuplicationPct:     5.0,  // → 85
		AvgNesting:         2.5,  // → 80
		PctFilesOver500LOC: 25.0, // → 50
	})
	// 80*0.40 + 85*0.30 + 80*0.15 + 50*0.15 = 32 + 25.5 + 12 + 7.5 = 77
	assert.InDelta(t, 77.0, score, 0.01)
}

func TestScoreMaintainability_Poor(t *testing.T) {
	score := ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      30.0,  // → 0 (clamped)
		DuplicationPct:     40.0,  // → 0 (clamped, 100-40*3=-20→0)
		AvgNesting:         6.5,   // → 0 (clamped)
		PctFilesOver500LOC: 100.0, // → 0 (clamped)
	})
	assert.InDelta(t, 0.0, score, 0.01)
}

// --- Security scorer tests ---

func TestScoreSecurity_Perfect(t *testing.T) {
	score := ScoreSecurity(SecurityInput{})
	assert.InDelta(t, 100.0, score, 0.01)
}

func TestScoreSecurity_SecretsFound(t *testing.T) {
	score := ScoreSecurity(SecurityInput{SecretsCount: 1})
	// secrets=60, cves=100, licenses=100 → 60*0.35 + 100*0.45 + 100*0.20 = 21+45+20 = 86
	assert.InDelta(t, 86.0, score, 0.01)
}

func TestScoreSecurity_TwoSecrets(t *testing.T) {
	score := ScoreSecurity(SecurityInput{SecretsCount: 2})
	// secrets=20, cves=100, licenses=100 → 20*0.35 + 100*0.45 + 100*0.20 = 7+45+20 = 72
	assert.InDelta(t, 72.0, score, 0.01)
}

func TestScoreSecurity_ThreeSecretsFloorsToZero(t *testing.T) {
	score := ScoreSecurity(SecurityInput{SecretsCount: 3})
	// secrets=0, cves=100, licenses=100 → 0*0.35 + 100*0.45 + 100*0.20 = 65
	assert.InDelta(t, 65.0, score, 0.01)
}

func TestScoreSecurity_OneCriticalCVE(t *testing.T) {
	score := ScoreSecurity(SecurityInput{CVECritical: 1})
	// secrets=100, cves=50, licenses=100 → 100*0.35 + 50*0.45 + 100*0.20 = 35+22.5+20 = 77.5
	assert.InDelta(t, 77.5, score, 0.01)
}

func TestScoreSecurity_TwoCriticalCVEs(t *testing.T) {
	score := ScoreSecurity(SecurityInput{CVECritical: 2})
	// cves = max(0, 100 - 100) = 0
	// 100*0.35 + 0*0.45 + 100*0.20 = 35+0+20 = 55
	assert.InDelta(t, 55.0, score, 0.01)
}

func TestScoreSecurity_OneHighCVE(t *testing.T) {
	score := ScoreSecurity(SecurityInput{CVEHigh: 1})
	// secrets=100, cves=75, licenses=100 → 100*0.35 + 75*0.45 + 100*0.20 = 35+33.75+20 = 88.75
	assert.InDelta(t, 88.75, score, 0.01)
}

func TestScoreSecurity_LicenseIssues(t *testing.T) {
	score := ScoreSecurity(SecurityInput{LicenseIssueCount: 2})
	// licenses = max(0, 100 - 50) = 50
	// 100*0.35 + 100*0.45 + 50*0.20 = 35+45+10 = 90
	assert.InDelta(t, 90.0, score, 0.01)
}

// --- Handoff scorer tests ---

func TestScoreHandoff_Perfect(t *testing.T) {
	score := ScoreHandoff(HandoffInput{
		EstTestCoveragePct: 80.0,
		DocDensity:         models.DocDensityHigh,
		EnvVarCount:        3,
	})
	// coverage = min(100, 80*1.25) = 100
	// doc = 90
	// env = max(0, 100 - 0) = 100
	// 100*0.50 + 90*0.25 + 100*0.25 = 50+22.5+25 = 97.5
	assert.InDelta(t, 97.5, score, 0.01)
}

func TestScoreHandoff_NoTests(t *testing.T) {
	score := ScoreHandoff(HandoffInput{
		EstTestCoveragePct: 0,
		DocDensity:         models.DocDensityLow,
		EnvVarCount:        20,
	})
	// coverage = 0
	// doc = 30
	// env = max(0, 100 - 15*3) = 55
	// 0*0.50 + 30*0.25 + 55*0.25 = 0+7.5+13.75 = 21.25
	assert.InDelta(t, 21.25, score, 0.01)
}

// --- Dependency health scorer tests ---

func TestScoreDependencyHealth_Healthy(t *testing.T) {
	score := ScoreDependencyHealth(DependencyHealthInput{
		MedianAgeMonths: 3,
		UnmaintainedPct: 0,
	})
	assert.InDelta(t, 100.0, score, 0.01)
}

func TestScoreDependencyHealth_Moderate(t *testing.T) {
	score := ScoreDependencyHealth(DependencyHealthInput{
		MedianAgeMonths: 18,
		UnmaintainedPct: 10,
	})
	// age = max(0, 100 - 12*2.5) = 70
	// unmaintained = max(0, 100 - 40) = 60
	// 70*0.50 + 60*0.50 = 65
	assert.InDelta(t, 65.0, score, 0.01)
}

// --- Activity scorer tests ---

func TestScoreActivity_Active(t *testing.T) {
	score := ScoreActivity(ActivityInput{
		DaysSinceLastCommit: 0,
		AvgCommitsPerMonth:  20,
		ActiveMonths:        12,
		RepoAgeMonths:       12,
	})
	// recency = 100, velocity = min(100, 22*sqrt(20)) ≈ 98.39, consistency = 12/12 = 100
	// 100*0.40 + 98.39*0.30 + 100*0.30 ≈ 99.52
	assert.InDelta(t, 99.52, score, 0.1)
}

func TestScoreActivity_Stale(t *testing.T) {
	score := ScoreActivity(ActivityInput{
		DaysSinceLastCommit: 180,
		AvgCommitsPerMonth:  0,
		ActiveMonths:        3,
		RepoAgeMonths:       12,
	})
	// recency = max(0, 100 - 180*0.55) = max(0, 1.0) = 1.0
	// velocity = 0
	// consistency = 3/12*100 = 25
	// 1.0*0.40 + 0*0.30 + 25*0.30 = 0.4+0+7.5 = 7.9
	assert.InDelta(t, 7.9, score, 0.01)
}

func TestScoreActivity_NewProject(t *testing.T) {
	// A 1-month-old project with 1 active month should get 100% consistency
	score := ScoreActivity(ActivityInput{
		DaysSinceLastCommit: 0,
		AvgCommitsPerMonth:  6,
		ActiveMonths:        1,
		RepoAgeMonths:       1,
	})
	// recency = 100, velocity = 22*sqrt(6) ≈ 53.9, consistency = 1/1 = 100
	// 100*0.40 + 53.9*0.30 + 100*0.30 = 40 + 16.17 + 30 = 86.17
	assert.InDelta(t, 86.17, score, 0.5)
}

// --- Infrastructure assessment tests (data-only, no score) ---

func TestAssessInfra_AllDetected(t *testing.T) {
	a := AssessInfra(InfraInput{
		IaCDetected:        true,
		CICDDetected:       true,
		MonitoringDetected: true,
	})
	assert.Equal(t, InvestmentLow, a.InvestmentLevel)
}

func TestAssessInfra_OnlyCICD(t *testing.T) {
	a := AssessInfra(InfraInput{CICDDetected: true})
	assert.Equal(t, InvestmentMedium, a.InvestmentLevel)
}

func TestAssessInfra_CICDAndMonitoring(t *testing.T) {
	a := AssessInfra(InfraInput{CICDDetected: true, MonitoringDetected: true})
	assert.Equal(t, InvestmentLow, a.InvestmentLevel)
}

func TestAssessInfra_CICDAndIaC(t *testing.T) {
	a := AssessInfra(InfraInput{CICDDetected: true, IaCDetected: true})
	assert.Equal(t, InvestmentLow, a.InvestmentLevel)
}

func TestAssessInfra_NoneDetected(t *testing.T) {
	a := AssessInfra(InfraInput{})
	assert.Equal(t, InvestmentHigh, a.InvestmentLevel)
}

func TestAssessInfra_IaCOnly(t *testing.T) {
	a := AssessInfra(InfraInput{IaCDetected: true})
	assert.Equal(t, InvestmentHigh, a.InvestmentLevel)
}

func TestAssessInfra_MonitoringOnly(t *testing.T) {
	a := AssessInfra(InfraInput{MonitoringDetected: true})
	assert.Equal(t, InvestmentHigh, a.InvestmentLevel)
}

// --- Overall score tests (SC-041) ---

func TestOverallScore_AllCategories(t *testing.T) {
	scores := []CategoryScore{
		{"security", 100},
		{"maintainability", 100},
		{"handoff_readiness", 100},
		{"development_activity", 100},
		{"dependency_health", 100},
	}
	assert.InDelta(t, 100.0, OverallScore(scores), 0.01)
}

func TestOverallScore_Weighted(t *testing.T) {
	scores := []CategoryScore{
		{"security", 80},             // 0.30
		{"maintainability", 70},      // 0.22
		{"handoff_readiness", 60},    // 0.22
		{"development_activity", 90}, // 0.15
		{"dependency_health", 50},    // 0.11
	}
	// 80*0.30 + 70*0.22 + 60*0.22 + 90*0.15 + 50*0.11
	// = 24 + 15.4 + 13.2 + 13.5 + 5.5 = 71.6
	assert.InDelta(t, 71.6, OverallScore(scores), 0.01)
}

func TestOverallScore_MissingCategory_Renormalized(t *testing.T) {
	// Activity (15%) missing → weights renormalized across remaining 85%
	scores := []CategoryScore{
		{"security", 100},
		{"maintainability", 100},
		{"handoff_readiness", 100},
		{"dependency_health", 100},
	}
	// All 100, renormalized → 100
	assert.InDelta(t, 100.0, OverallScore(scores), 0.01)
}

func TestOverallScore_MissingCategory_NonUniform(t *testing.T) {
	// Activity missing, different scores → renormalized weights
	scores := []CategoryScore{
		{"security", 80},          // 0.30/0.85
		{"maintainability", 60},   // 0.22/0.85
		{"handoff_readiness", 70}, // 0.22/0.85
		{"dependency_health", 50}, // 0.11/0.85
	}
	// weighted sum = 80*0.30 + 60*0.22 + 70*0.22 + 50*0.11
	//             = 24 + 13.2 + 15.4 + 5.5 = 58.1
	// total weight = 0.85
	// renormalized = 58.1 / 0.85 = 68.35...
	assert.InDelta(t, 68.35, OverallScore(scores), 0.01)
}

func TestOverallScore_Empty(t *testing.T) {
	assert.InDelta(t, 0.0, OverallScore(nil), 0.01)
}

func TestScoreToGrade_OutOfRange(t *testing.T) {
	assert.Equal(t, models.GradeA, ScoreToGrade(200))
	assert.Equal(t, models.GradeF, ScoreToGrade(-5))
}

func TestOverallScore_Clamped(t *testing.T) {
	// Ensure OverallScore never exceeds 100
	scores := []CategoryScore{
		{"security", 100},
		{"maintainability", 100},
		{"handoff_readiness", 100},
		{"development_activity", 100},
		{"dependency_health", 100},
	}
	result := OverallScore(scores)
	assert.LessOrEqual(t, result, 100.0)
	assert.GreaterOrEqual(t, result, 0.0)
}

func TestScoreMaintainability_ExtremeValues(t *testing.T) {
	score := ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      100.0,
		DuplicationPct:     100.0,
		AvgNesting:         20.0,
		PctFilesOver500LOC: 100.0,
	})
	assert.GreaterOrEqual(t, score, 0.0)

	score = ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      0.0,
		DuplicationPct:     0.0,
		AvgNesting:         0.0,
		PctFilesOver500LOC: 0.0,
	})
	assert.LessOrEqual(t, score, 100.0)
}

func TestScoreActivity_ExtremeValues(t *testing.T) {
	score := ScoreActivity(ActivityInput{
		DaysSinceLastCommit: 10000,
		AvgCommitsPerMonth:  0,
		ActiveMonths:        0,
	})
	assert.InDelta(t, 0.0, score, 0.01)
}

func TestScoreSecurity_AllBad(t *testing.T) {
	score := ScoreSecurity(SecurityInput{
		SecretsCount:      10,
		CVECritical:       5,
		CVEHigh:           5,
		CVEMedium:         10,
		CVELow:            20,
		LicenseIssueCount: 10,
	})
	assert.InDelta(t, 0.0, score, 0.01)
}

func TestScoreSecurity_TransitiveCVE_HalfWeight(t *testing.T) {
	// One critical direct CVE: penalty = 50
	directScore := ScoreSecurity(SecurityInput{CVECritical: 1})
	// One critical transitive CVE: penalty = 25 (half weight)
	transScore := ScoreSecurity(SecurityInput{CVECriticalTrans: 1})

	// Direct: cves = 100-50 = 50, overall = 100*0.35 + 50*0.45 + 100*0.20 = 77.5
	assert.InDelta(t, 77.5, directScore, 0.01)
	// Transitive: cves = 100-25 = 75, overall = 100*0.35 + 75*0.45 + 100*0.20 = 88.75
	assert.InDelta(t, 88.75, transScore, 0.01)
}

func TestScoreSecurity_MixedDirectAndTransitive(t *testing.T) {
	// 1 direct critical (penalty 50) + 1 transitive critical (penalty 25)
	score := ScoreSecurity(SecurityInput{CVECritical: 1, CVECriticalTrans: 1})
	// cves = clamp(100 - 50 - 25) = 25
	// 100*0.35 + 25*0.45 + 100*0.20 = 35 + 11.25 + 20 = 66.25
	assert.InDelta(t, 66.25, score, 0.01)
}

// --- Additional boundary value tests ---

func TestScoreToGrade_ExactBoundaries(t *testing.T) {
	// Test exact grade boundary values
	boundaries := []struct {
		score float64
		want  models.Grade
	}{
		{93, models.GradeA},   // A starts at 93
		{92.9, models.GradeAM}, // Just below A
		{90, models.GradeAM},  // A- starts at 90
		{89.9, models.GradeBP}, // Just below A-
		{87, models.GradeBP},  // B+ starts at 87
		{86.9, models.GradeB}, // Just below B+
		{83, models.GradeB},   // B starts at 83
		{82.9, models.GradeBM}, // Just below B
		{80, models.GradeBM},  // B- starts at 80
		{79.9, models.GradeCP}, // Just below B-
		{77, models.GradeCP},  // C+ starts at 77
		{76.9, models.GradeC}, // Just below C+
		{73, models.GradeC},   // C starts at 73
		{72.9, models.GradeCM}, // Just below C
		{70, models.GradeCM},  // C- starts at 70
		{69.9, models.GradeDP}, // Just below C-
		{67, models.GradeDP},  // D+ starts at 67
		{66.9, models.GradeD}, // Just below D+
		{63, models.GradeD},   // D starts at 63
		{62.9, models.GradeDM}, // Just below D
		{60, models.GradeDM},  // D- starts at 60
		{59.9, models.GradeF}, // Just below D-
		{0, models.GradeF},    // Bottom
	}
	for _, tt := range boundaries {
		got := ScoreToGrade(tt.score)
		assert.Equal(t, tt.want, got, "score %.1f should be %s, got %s", tt.score, tt.want, got)
	}
}

func TestGradeMeetsThreshold(t *testing.T) {
	tests := []struct {
		actual    models.Grade
		threshold models.Grade
		want      bool
	}{
		{models.GradeA, models.GradeA, true},   // exact match
		{models.GradeA, models.GradeF, true},   // A >= F
		{models.GradeF, models.GradeA, false},  // F < A
		{models.GradeC, models.GradeC, true},   // exact match
		{models.GradeBM, models.GradeC, true},  // B- >= C
		{models.GradeCM, models.GradeC, false}, // C- < C
		{models.GradeCP, models.GradeC, true},  // C+ >= C
		{models.GradeDP, models.GradeC, false}, // D+ < C
		{models.GradeF, models.GradeF, true},   // F >= F
		{models.GradeAM, models.GradeA, false}, // A- < A
	}
	for _, tt := range tests {
		got := GradeMeetsThreshold(tt.actual, tt.threshold)
		assert.Equal(t, tt.want, got, "GradeMeetsThreshold(%s, %s) should be %v", tt.actual, tt.threshold, tt.want)
	}
}

func TestScoreDependencyHealth_ExtremeOld(t *testing.T) {
	score := ScoreDependencyHealth(DependencyHealthInput{
		MedianAgeMonths: 120, // 10 years
		UnmaintainedPct: 100, // all unmaintained
	})
	assert.InDelta(t, 0.0, score, 0.01)
}

func TestScoreHandoff_MediumDocDensity(t *testing.T) {
	score := ScoreHandoff(HandoffInput{
		EstTestCoveragePct: 30,
		DocDensity:         models.DocDensityMedium,
		EnvVarCount:        5,
	})
	// coverage = min(100, 30*1.25) = 37.5
	// doc = 60 (medium)
	// env = clamp(100 - max(0, (5-5))*3) = 100
	// 37.5*0.50 + 60*0.25 + 100*0.25 = 18.75 + 15 + 25 = 58.75
	assert.InDelta(t, 58.75, score, 0.01)
}

func TestOverallScore_SingleCategory(t *testing.T) {
	scores := []CategoryScore{
		{"security", 75},
	}
	// Only security (weight 0.25) present, renormalized to 100%
	assert.InDelta(t, 75.0, OverallScore(scores), 0.01)
}

func TestAssessInfra_IaCAndMonitoring(t *testing.T) {
	a := AssessInfra(InfraInput{IaCDetected: true, MonitoringDetected: true})
	// No CI/CD → still high investment
	assert.Equal(t, InvestmentHigh, a.InvestmentLevel)
}
