package models

// ScanResult is the top-level JSON output conforming to data contract 9a.
type ScanResult struct {
	Version        string          `json:"version"`
	ScanID         string          `json:"scan_id"`
	Timestamp      string          `json:"timestamp"`
	ScannerVersion string          `json:"scanner_version"`
	Repositories   []Repository    `json:"repositories"`
	TotalLOC       int             `json:"total_loc"`
	TotalFileCount int             `json:"total_file_count"`
	RepoCount      int             `json:"repo_count"`
	TechStack      TechStack       `json:"tech_stack"`
	Metrics        Metrics         `json:"metrics"`
	Activity       *Activity       `json:"activity"`
	Detection      Detection       `json:"detection"`
	RedFlags       RedFlags        `json:"red_flags"`
	Summary        Summary         `json:"summary"`
	PricingTier    PricingTier     `json:"pricing_tier"`
	Warnings       []Warning       `json:"warnings"`
	Integrity      Integrity       `json:"integrity"`
}

// Repository represents a scanned repository.
type Repository struct {
	Name              string             `json:"name"`
	PathHash          string             `json:"path_hash"`
	HeadCommitSHA     string             `json:"head_commit_sha,omitempty"`
	Languages         map[string]float64 `json:"languages"`
	FileCount         int                `json:"file_count"`
	LOC               int                `json:"loc"`
	Status            RepoStatus         `json:"status"`
	DetectedLanguages []string           `json:"detected_languages"`
}

// TechStack represents detected technology stack.
type TechStack struct {
	Frameworks       []string `json:"frameworks"`
	Runtimes         []string `json:"runtimes"`
	Databases        []string `json:"databases"`
	ExternalServices []string `json:"external_services"`
}

// Metrics contains all scored metric categories.
type Metrics struct {
	Maintainability  *Maintainability  `json:"maintainability"`
	Security         *Security         `json:"security"`
	DependencyHealth *DependencyHealth `json:"dependency_health"`
	HandoffReadiness *HandoffReadiness `json:"handoff_readiness"`
}

// Maintainability metrics.
type Maintainability struct {
	Grade              *Grade        `json:"grade"`
	NAReason           string        `json:"na_reason,omitempty"`
	CyclomaticComplexity ComplexityStats `json:"cyclomatic_complexity"`
	NestingDepth       NestingStats  `json:"nesting_depth"`
	DuplicationPct     float64       `json:"duplication_pct"`
	HotspotCount       int           `json:"hotspot_count"`
	HotspotFiles       []HotspotFile `json:"hotspot_files"`
	PctFilesOver500LOC float64       `json:"pct_files_over_500loc"`
}

// ComplexityStats holds complexity statistics.
type ComplexityStats struct {
	Avg float64 `json:"avg"`
	P90 int     `json:"p90"`
	Max int     `json:"max"`
}

// NestingStats holds nesting depth statistics.
type NestingStats struct {
	Avg float64 `json:"avg"`
	Max int     `json:"max"`
}

// HotspotFile represents a high-complexity file.
type HotspotFile struct {
	FileHash   string `json:"file_hash"`
	Complexity int    `json:"complexity"`
	LOC        int    `json:"loc"`
	Repo       string `json:"repo"`
	// Path is the real file path, used only for terminal display (never serialized to JSON).
	Path string `json:"-"`
}

// Security metrics.
type Security struct {
	Grade                *Grade         `json:"grade"`
	SecretsFound         int            `json:"secrets_found"`
	CVEs                 []CVE          `json:"cves"`
	CVESummary           CVESummary     `json:"cve_summary"`
	OutdatedDeps         OutdatedDeps   `json:"outdated_deps"`
	LicenseIssues        []LicenseIssue `json:"license_issues"`
	LicenseIssueCount    int            `json:"license_issue_count"`
	CVEEcosystemsSkipped []string       `json:"cve_ecosystems_skipped"`
}

// CVE represents a known vulnerability.
type CVE struct {
	ID             string   `json:"id"`
	Severity       Severity `json:"severity"`
	Package        string   `json:"package"`
	CurrentVersion string   `json:"current_version"`
	FixedIn        string   `json:"fixed_in"`
	Repo           string   `json:"repo"`
}

// CVESummary counts CVEs by severity.
type CVESummary struct {
	Critical int `json:"critical"`
	High     int `json:"high"`
	Medium   int `json:"medium"`
	Low      int `json:"low"`
}

// OutdatedDeps summarizes outdated dependencies.
type OutdatedDeps struct {
	Total       int     `json:"total"`
	Outdated    int     `json:"outdated"`
	Critical    int     `json:"critical"`
	OutdatedPct float64 `json:"outdated_pct"`
}

// LicenseIssue represents a license compatibility problem.
type LicenseIssue struct {
	Package string `json:"package"`
	License string `json:"license"`
	Issue   string `json:"issue"`
	Repo    string `json:"repo"`
}

// DependencyHealth metrics.
type DependencyHealth struct {
	Grade            *Grade      `json:"grade"`
	NAReason         string      `json:"na_reason,omitempty"`
	MedianAgeMonths  int         `json:"median_age_months"`
	UnmaintainedPct  float64     `json:"unmaintained_pct"`
	UnmaintainedCount int        `json:"unmaintained_count"`
	Oldest           *OldestDep  `json:"oldest"`
}

// OldestDep represents the oldest dependency.
type OldestDep struct {
	Package  string  `json:"package"`
	AgeYears float64 `json:"age_years"`
	Repo     string  `json:"repo"`
}

// HandoffReadiness metrics.
type HandoffReadiness struct {
	Grade               *Grade     `json:"grade"`
	EstTestCoveragePct  float64    `json:"est_test_coverage_pct"`
	DocDensity          DocDensity `json:"doc_density"`
	EnvVarCount         int        `json:"env_var_count"`
	HasReadme           bool       `json:"has_readme"`
	HasContributingGuide bool      `json:"has_contributing_guide"`
	HasEnvTemplate      bool       `json:"has_env_template"`
	HasSetupScript      bool       `json:"has_setup_script"`
}

// Activity holds development activity metrics.
type Activity struct {
	Grade              *Grade          `json:"grade"`
	NAReason           string          `json:"na_reason,omitempty"`
	LastCommitDate     string          `json:"last_commit_date"`
	DaysSinceLastCommit int            `json:"days_since_last_commit"`
	CommitVelocity     CommitVelocity  `json:"commit_velocity"`
	ActiveMonths       int             `json:"active_months"`
	TotalMonths        int             `json:"total_months"`
	ContributorCount   int             `json:"contributor_count"`
}

// CommitVelocity represents commit frequency data.
type CommitVelocity struct {
	AvgPerMonth  float64 `json:"avg_per_month"`
	Trend        Trend   `json:"trend"`
	Last12Months []int   `json:"last_12_months"`
}

// Detection contains binary detection flags.
type Detection struct {
	AI             AIDetection             `json:"ai"`
	Infrastructure InfrastructureDetection `json:"infrastructure"`
}

// AIDetection flags for AI/ML capabilities.
type AIDetection struct {
	LLMAPI            bool   `json:"llm_api"`
	LLMProvider       string `json:"llm_provider,omitempty"`
	VectorDatabase    bool   `json:"vector_database"`
	VectorDBName      string `json:"vector_db_name,omitempty"`
	RAGPipeline       bool   `json:"rag_pipeline"`
	MCPServers        bool   `json:"mcp_servers"`
	FineTunedModels   bool   `json:"fine_tuned_models"`
	TrainingPipeline  bool   `json:"training_pipeline"`
	ProprietaryDataset bool  `json:"proprietary_dataset"`
}

// InfrastructureDetection holds infrastructure detection results.
type InfrastructureDetection struct {
	Grade            *Grade   `json:"grade"`
	IaCDetected      bool     `json:"iac_detected"`
	IaCTypes         []string `json:"iac_types"`
	CICDDetected     bool     `json:"ci_cd_detected"`
	CICDProvider     string   `json:"ci_cd_provider,omitempty"`
	MonitoringDetected bool   `json:"monitoring_detected"`
	MonitoringTools  []string `json:"monitoring_tools"`
}

// RedFlags contains triggered red flags.
type RedFlags struct {
	Count int       `json:"count"`
	Flags []RedFlag `json:"flags"`
}

// RedFlag represents a single triggered red flag.
type RedFlag struct {
	Flag     RedFlagCode `json:"flag"`
	Detail   string      `json:"detail"`
	Severity Severity    `json:"severity"`
}

// Summary contains the overall assessment.
type Summary struct {
	ScoredCategories []string    `json:"scored_categories"`
	OverallGrade     *Grade      `json:"overall_grade"`
	TopRisks         []Risk      `json:"top_risks"`
	TopStrengths     []Strength  `json:"top_strengths"`
}

// Risk represents a top risk finding.
type Risk struct {
	Category string   `json:"category"`
	Issue    string   `json:"issue"`
	Severity Severity `json:"severity"`
}

// Strength represents a top strength finding.
type Strength struct {
	Category string `json:"category"`
	Detail   string `json:"detail"`
}

// PricingTier represents the auto-determined pricing tier.
type PricingTier struct {
	Tier   PricingTierName `json:"tier"`
	Reason string          `json:"reason"`
}

// Warning represents a partial analysis warning.
type Warning struct {
	Code      string `json:"code"`
	Message   string `json:"message"`
	Repo      string `json:"repo,omitempty"`
	Ecosystem string `json:"ecosystem,omitempty"`
}

// Integrity holds the scan signature and co-signing data.
type Integrity struct {
	ScanChecksum        string            `json:"scan_checksum"`
	ScannerPublicKeyID  string            `json:"scanner_public_key_id"`
	ScannerSignature    string            `json:"scanner_signature"`
	CosignNonce         *string           `json:"cosign_nonce"`
	PlatformCosignature *string           `json:"platform_cosignature"`
	PlatformPublicKeyID *string           `json:"platform_public_key_id"`
	Cosigned            bool              `json:"cosigned"`
	VerificationLevel   VerificationLevel `json:"verification_level"`
}
