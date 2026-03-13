package scorer

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/pkg/models"
)

func TestDeterminePricingTier(t *testing.T) {
	tests := []struct {
		loc  int
		want models.PricingTierName
	}{
		{500, models.PricingTierStarter},
		{30000, models.PricingTierStarter},
		{30001, models.PricingTierStandard},
		{100000, models.PricingTierStandard},
		{100001, models.PricingTierProfessional},
		{300000, models.PricingTierProfessional},
		{300001, models.PricingTierEnterprise},
		{1000000, models.PricingTierEnterprise},
	}
	for _, tt := range tests {
		pt := DeterminePricingTier(tt.loc)
		assert.Equal(t, tt.want, pt.Tier, "LOC=%d", tt.loc)
		assert.NotEmpty(t, pt.Reason)
	}
}

func TestFormatLOC(t *testing.T) {
	assert.Equal(t, "500 LOC", formatLOC(500))
	assert.Equal(t, "1,000 LOC", formatLOC(1000))
	assert.Equal(t, "42,600 LOC", formatLOC(42600))
	assert.Equal(t, "1,000,000 LOC", formatLOC(1000000))
}

func TestDeterminePricingTier_Zero(t *testing.T) {
	pt := DeterminePricingTier(0)
	assert.Equal(t, models.PricingTierStarter, pt.Tier)
}

func TestFormatLOC_EdgeCases(t *testing.T) {
	assert.Equal(t, "0 LOC", formatLOC(0))
	assert.Equal(t, "1 LOC", formatLOC(1))
	assert.Equal(t, "999 LOC", formatLOC(999))
	assert.Equal(t, "0 LOC", formatLOC(-5)) // negative clamped to 0
}
