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
		DuplicationPct:     5.0,  // → 75
		AvgNesting:         2.5,  // → 80
		PctFilesOver500LOC: 25.0, // → 50
	})
	// 80*0.40 + 75*0.30 + 80*0.15 + 50*0.15 = 32 + 22.5 + 12 + 7.5 = 74
	assert.InDelta(t, 74.0, score, 0.01)
}

func TestScoreMaintainability_Poor(t *testing.T) {
	score := ScoreMaintainability(MaintainabilityInput{
		AvgComplexity:      30.0,  // → 0 (clamped)
		DuplicationPct:     25.0,  // → 0 (clamped)
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
	// secrets=0, cves=100, licenses=100 → 0*0.35 + 100*0.45 + 100*0.20 = 65
	assert.InDelta(t, 65.0, score, 0.01)
}

func TestScoreSecurity_OneCriticalCVE(t *testing.T) {
	score := ScoreSecurity(SecurityInput{CVECritical: 1})
	// secrets=100, cves=70, licenses=100 → 100*0.35 + 70*0.45 + 100*0.20 = 35+31.5+20 = 86.5
	assert.InDelta(t, 86.5, score, 0.01)
}

func TestScoreSecurity_TwoCriticalCVEs(t *testing.T) {
	score := ScoreSecurity(SecurityInput{CVECritical: 2})
	// cves = max(0, 100 - 60) = 40
	// 100*0.35 + 40*0.45 + 100*0.20 = 35+18+20 = 73
	assert.InDelta(t, 73.0, score, 0.01)
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
	})
	assert.InDelta(t, 100.0, score, 0.01)
}

func TestScoreActivity_Stale(t *testing.T) {
	score := ScoreActivity(ActivityInput{
		DaysSinceLastCommit: 180,
		AvgCommitsPerMonth:  0,
		ActiveMonths:        3,
	})
	// recency = max(0, 100 - 180*0.55) = max(0, 1.0) = 1.0
	// velocity = 0
	// consistency = 3/12*100 = 25
	// 1.0*0.40 + 0*0.30 + 25*0.30 = 0.4+0+7.5 = 7.9
	assert.InDelta(t, 7.9, score, 0.01)
}

// --- Infrastructure scorer tests ---

func TestScoreInfra_AllDetected(t *testing.T) {
	score := ScoreInfra(InfraInput{
		IaCDetected:        true,
		CICDDetected:       true,
		MonitoringDetected: true,
	})
	assert.InDelta(t, 100.0, score, 0.01)
}

func TestScoreInfra_OnlyCICD(t *testing.T) {
	score := ScoreInfra(InfraInput{CICDDetected: true})
	assert.InDelta(t, 40.0, score, 0.01)
}

func TestScoreInfra_NoneDetected(t *testing.T) {
	score := ScoreInfra(InfraInput{})
	assert.InDelta(t, 0.0, score, 0.01)
}

// --- Overall score tests (SC-041) ---

func TestOverallScore_AllCategories(t *testing.T) {
	scores := []CategoryScore{
		{"security", 100},
		{"maintainability", 100},
		{"handoff_readiness", 100},
		{"development_activity", 100},
		{"dependency_health", 100},
		{"sre_infrastructure", 100},
	}
	assert.InDelta(t, 100.0, OverallScore(scores), 0.01)
}

func TestOverallScore_Weighted(t *testing.T) {
	scores := []CategoryScore{
		{"security", 80},          // 0.25
		{"maintainability", 70},   // 0.20
		{"handoff_readiness", 60}, // 0.20
		{"development_activity", 90}, // 0.15
		{"dependency_health", 50}, // 0.10
		{"sre_infrastructure", 40}, // 0.10
	}
	// 80*0.25 + 70*0.20 + 60*0.20 + 90*0.15 + 50*0.10 + 40*0.10
	// = 20 + 14 + 12 + 13.5 + 5 + 4 = 68.5
	assert.InDelta(t, 68.5, OverallScore(scores), 0.01)
}

func TestOverallScore_MissingCategory_Renormalized(t *testing.T) {
	// Activity (15%) missing → weights renormalized across remaining 85%
	scores := []CategoryScore{
		{"security", 100},
		{"maintainability", 100},
		{"handoff_readiness", 100},
		{"dependency_health", 100},
		{"sre_infrastructure", 100},
	}
	// All 100, renormalized → 100
	assert.InDelta(t, 100.0, OverallScore(scores), 0.01)
}

func TestOverallScore_MissingCategory_NonUniform(t *testing.T) {
	// Activity missing, different scores → renormalized weights
	scores := []CategoryScore{
		{"security", 80},          // 0.25/0.85
		{"maintainability", 60},   // 0.20/0.85
		{"handoff_readiness", 70}, // 0.20/0.85
		{"dependency_health", 50}, // 0.10/0.85
		{"sre_infrastructure", 40}, // 0.10/0.85
	}
	// weighted sum = 80*0.25 + 60*0.20 + 70*0.20 + 50*0.10 + 40*0.10
	//             = 20 + 12 + 14 + 5 + 4 = 55
	// total weight = 0.85
	// renormalized = 55 / 0.85 = 64.706...
	assert.InDelta(t, 64.71, OverallScore(scores), 0.01)
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
		{"sre_infrastructure", 100},
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
