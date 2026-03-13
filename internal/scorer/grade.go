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

// CategoryWeight defines the weight of each scoring category.
type CategoryWeight struct {
	Name   string
	Weight float64
}

// DefaultWeights are the standard category weights per the scoring methodology.
var DefaultWeights = []CategoryWeight{
	{"security", 0.25},
	{"maintainability", 0.20},
	{"handoff_readiness", 0.20},
	{"development_activity", 0.15},
	{"dependency_health", 0.10},
	{"sre_infrastructure", 0.10},
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
