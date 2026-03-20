package scorer

import "github.com/vettcode/scanner/pkg/models"

// ScoreToGrade converts a numeric score (0-100) to a letter grade.
func ScoreToGrade(score float64) models.Grade {
	switch {
	case score >= 93:
		return models.GradeA
	case score >= 90:
		return models.GradeAM
	case score >= 87:
		return models.GradeBP
	case score >= 83:
		return models.GradeB
	case score >= 80:
		return models.GradeBM
	case score >= 77:
		return models.GradeCP
	case score >= 73:
		return models.GradeC
	case score >= 70:
		return models.GradeCM
	case score >= 67:
		return models.GradeDP
	case score >= 63:
		return models.GradeD
	case score >= 60:
		return models.GradeDM
	default:
		return models.GradeF
	}
}

// gradeRank maps each grade to a numeric rank for comparison (higher = better).
var gradeRank = map[models.Grade]int{
	models.GradeA:  12,
	models.GradeAM: 11,
	models.GradeBP: 10,
	models.GradeB:  9,
	models.GradeBM: 8,
	models.GradeCP: 7,
	models.GradeC:  6,
	models.GradeCM: 5,
	models.GradeDP: 4,
	models.GradeD:  3,
	models.GradeDM: 2,
	models.GradeF:  1,
}

// GradeMeetsThreshold returns true if actual grade is >= the threshold grade.
func GradeMeetsThreshold(actual, threshold models.Grade) bool {
	return gradeRank[actual] >= gradeRank[threshold]
}

// CategoryWeight defines the weight of each scoring category.
type CategoryWeight struct {
	Name   string
	Weight float64
}

// DefaultWeights are the standard category weights per the scoring methodology.
// SRE & Infrastructure is data-only (not scored) — 5 scored categories.
var DefaultWeights = []CategoryWeight{
	{"security", 0.30},
	{"maintainability", 0.22},
	{"handoff_readiness", 0.22},
	{"development_activity", 0.15},
	{"dependency_health", 0.11},
}

// CategoryScore holds a scored category's name and numeric score.
type CategoryScore struct {
	Name  string
	Score float64
}

// OverallScore computes the weighted overall score from category scores.
// Categories with nil/N/A scores are excluded and weights are renormalized.
func OverallScore(scores []CategoryScore) float64 {
	weightMap := make(map[string]float64)
	for _, w := range DefaultWeights {
		weightMap[w.Name] = w.Weight
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for _, cs := range scores {
		w, ok := weightMap[cs.Name]
		if !ok {
			continue
		}
		totalWeight += w
		weightedSum += cs.Score * w
	}

	if totalWeight == 0 {
		return 0
	}

	return clamp(weightedSum / totalWeight)
}
