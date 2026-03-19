package models

// Grade represents a letter grade from A to F.
type Grade string

const (
	GradeA  Grade = "A"
	GradeAM Grade = "A-"
	GradeBP Grade = "B+"
	GradeB  Grade = "B"
	GradeBM Grade = "B-"
	GradeCP Grade = "C+"
	GradeC  Grade = "C"
	GradeCM Grade = "C-"
	GradeDP Grade = "D+"
	GradeD  Grade = "D"
	GradeDM Grade = "D-"
	GradeF  Grade = "F"
)

// Severity represents a CVE or risk severity level.
type Severity string

const (
	SeverityCritical Severity = "critical"
	SeverityHigh     Severity = "high"
	SeverityMedium   Severity = "medium"
	SeverityLow      Severity = "low"
)

// RepoStatus represents the analysis status of a repository.
type RepoStatus string

const (
	RepoStatusAnalyzed    RepoStatus = "analyzed"
	RepoStatusUnsupported RepoStatus = "unsupported"
	RepoStatusError       RepoStatus = "error"
)

// Trend represents a development activity trend.
type Trend string

const (
	TrendIncreasing Trend = "increasing"
	TrendStable     Trend = "stable"
	TrendDeclining  Trend = "declining"
)

// DocDensity represents documentation density level.
type DocDensity string

const (
	DocDensityHigh   DocDensity = "high"
	DocDensityMedium DocDensity = "medium"
	DocDensityLow    DocDensity = "low"
)

// PricingTierName represents the pricing tier.
type PricingTierName string

const (
	PricingTierStarter      PricingTierName = "starter"
	PricingTierStandard     PricingTierName = "standard"
	PricingTierProfessional PricingTierName = "professional"
	PricingTierEnterprise   PricingTierName = "enterprise"
)

// VerificationLevel represents the scan verification level.
type VerificationLevel string

const (
	VerificationSelfReported    VerificationLevel = "self_reported"
	VerificationPlatformCosigned VerificationLevel = "platform_cosigned"
	VerificationProviderVerified VerificationLevel = "provider_verified"
)
