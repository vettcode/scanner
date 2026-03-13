package scorer

import (
	"fmt"

	"github.com/vettcode/scanner/pkg/models"
)

// DeterminePricingTier auto-determines the pricing tier based on total LOC.
func DeterminePricingTier(totalLOC int) models.PricingTier {
	locStr := formatLOC(totalLOC)

	switch {
	case totalLOC <= 30000:
		return models.PricingTier{Tier: models.PricingTierStarter, Reason: locStr}
	case totalLOC <= 100000:
		return models.PricingTier{Tier: models.PricingTierStandard, Reason: locStr}
	case totalLOC <= 300000:
		return models.PricingTier{Tier: models.PricingTierProfessional, Reason: locStr}
	default:
		return models.PricingTier{Tier: models.PricingTierEnterprise, Reason: locStr}
	}
}

// formatLOC formats a LOC count with comma separators and "LOC" suffix.
func formatLOC(loc int) string {
	if loc < 0 {
		loc = 0
	}
	if loc < 1000 {
		return fmt.Sprintf("%d LOC", loc)
	}

	s := fmt.Sprintf("%d", loc)
	n := len(s)
	result := make([]byte, 0, n+(n-1)/3)
	for i, c := range s {
		if i > 0 && (n-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, byte(c))
	}
	return string(result) + " LOC"
}
