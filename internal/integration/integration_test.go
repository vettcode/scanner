package integration

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/vettcode/scanner/internal/analyzer/aidetect"
	"github.com/vettcode/scanner/internal/analyzer/complexity"
	"github.com/vettcode/scanner/internal/analyzer/deps"
	"github.com/vettcode/scanner/internal/analyzer/duplication"
	"github.com/vettcode/scanner/internal/analyzer/filesize"
	"github.com/vettcode/scanner/internal/analyzer/handoff"
	"github.com/vettcode/scanner/internal/analyzer/infra"
	"github.com/vettcode/scanner/internal/analyzer/secrets"
	"github.com/vettcode/scanner/internal/analyzer/security"
	"github.com/vettcode/scanner/internal/analyzer/techstack"
	"github.com/vettcode/scanner/internal/language"
	"github.com/vettcode/scanner/internal/output"
	"github.com/vettcode/scanner/internal/scorer"
	"github.com/vettcode/scanner/internal/walker"
	"github.com/vettcode/scanner/pkg/models"
	"github.com/vettcode/scanner/testdata"
)

// TestMultiLanguageScan_HealthySaas walks the healthy-saas fixture and runs
// all analyzers, verifying language detection and per-language results.
func TestMultiLanguageScan_HealthySaas(t *testing.T) {
	root := testdata.FixturePath(testdata.HealthySaas)
	walkResult, err := walker.Walk(root)
	require.NoError(t, err)

	// Language detection: should detect JS/TS and Python with percentage breakdown
	langResult := language.AggregateResults(walkResult.LanguageLOC)
	assert.True(t, langResult.HasTier1, "should detect Tier 1 languages")
	assert.Contains(t, langResult.DetectedLanguages, "JavaScript")
	assert.Contains(t, langResult.DetectedLanguages, "Python")
	assert.Greater(t, langResult.TotalLOC, 0)
	// Verify percentage breakdown exists and sums to ~100%
	totalPct := 0.0
	for _, pct := range langResult.Percentages {
		totalPct += pct
	}
	assert.InDelta(t, 100.0, totalPct, 0.1, "percentages should sum to ~100")

	// Tier 2 files (YAML, Dockerfile) should appear in LOC but not in complexity
	hasTier2Files := false
	for _, f := range walkResult.Files {
		if f.Tier == language.Tier2 {
			hasTier2Files = true
			break
		}
	}
	assert.True(t, hasTier2Files, "healthy-saas should have some Tier 2 files (YAML, Docker, etc.)")

	// Complexity: run on all Tier 1 files
	var complexityResults []*complexity.FileResult
	for _, f := range walkResult.Files {
		if f.Tier == language.Tier1 && !f.IsTest {
			fr, cErr := complexity.AnalyzeFile(f.Path, f.Language)
			if cErr != nil {
				t.Logf("complexity analysis error for %s: %v", f.RelPath, cErr)
			} else if fr != nil {
				complexityResults = append(complexityResults, fr)
			}
		}
	}
	summary := complexity.Summarize(complexityResults)
	assert.Greater(t, summary.TotalFunctions, 0, "should find functions")
	assert.Greater(t, summary.AvgComplexity, 0.0, "should compute complexity")

	// Duplication
	dupResult := duplication.Analyze(walkResult.Files)
	assert.NotNil(t, dupResult)

	// File size distribution
	fsResult := filesize.Analyze(walkResult.Files)
	assert.NotNil(t, fsResult)

	// Secrets
	secretsResult := secrets.Scan(walkResult.Files)
	assert.NotNil(t, secretsResult)
	// healthy-saas should have no secrets
	assert.Equal(t, 0, secretsResult.SecretsCount, "healthy-saas should have no secrets")

	// Dependencies: ParseDependencies looks at root-level manifests, so parse sub-projects
	npmResult := deps.ParseDependencies(filepath.Join(root, "frontend"))
	assert.Greater(t, len(npmResult.Dependencies), 0, "should find npm deps")
	assert.Contains(t, npmResult.Ecosystems, "npm")

	pyResult := deps.ParseDependencies(filepath.Join(root, "backend"))
	assert.Greater(t, len(pyResult.Dependencies), 0, "should find pypi deps")
	assert.Contains(t, pyResult.Ecosystems, "pypi")

	// Merge dependencies for downstream use
	depResult := &deps.ParseResult{
		Dependencies: append(npmResult.Dependencies, pyResult.Dependencies...),
		Ecosystems:   append(npmResult.Ecosystems, pyResult.Ecosystems...),
	}

	// Licenses
	licenseResult := security.DetectLicenses(root)
	assert.NotNil(t, licenseResult)

	// CVEs (offline mode)
	cveResult := security.LookupCVEs(depResult.Dependencies, true)
	assert.NotNil(t, cveResult)

	// Tech stack
	var depNames []string
	for _, d := range depResult.Dependencies {
		depNames = append(depNames, d.Name)
	}
	techResult := techstack.Detect(root, depNames)
	assert.NotNil(t, techResult)

	// AI detection
	aiResult := aidetect.Detect(depNames, walkResult.Files)
	assert.NotNil(t, aiResult)

	// Infrastructure
	infraResult := infra.Analyze(root, walkResult.Files, depNames)
	assert.NotNil(t, infraResult)
	assert.True(t, infraResult.HasCICD, "healthy-saas should have CI/CD")

	// Handoff
	handoffResult := handoff.Analyze(root, walkResult)
	assert.NotNil(t, handoffResult)
	assert.True(t, handoffResult.HasReadme, "healthy-saas should have README")
	assert.True(t, handoffResult.HasEnvTemplate, "healthy-saas should have .env.example")
	assert.Greater(t, handoffResult.EstTestCoveragePct, 0.0, "should have some test coverage")

	// Scoring: wire up maintainability
	maintScore := scorer.ScoreMaintainability(scorer.MaintainabilityInput{
		AvgComplexity:      summary.AvgComplexity,
		DuplicationPct:     dupResult.DuplicationPct,
		AvgNesting:         summary.AvgNesting,
		PctFilesOver500LOC: fsResult.PctOver500LOC,
	})
	assert.Greater(t, maintScore, 0.0, "maintainability score should be positive")

	// Scoring: security
	secScore := scorer.ScoreSecurity(scorer.SecurityInput{
		SecretsCount:    secretsResult.SecretsCount,
		CVECritical:     cveResult.Summary.Critical,
		CVEHigh:         cveResult.Summary.High,
		CVEMedium:       cveResult.Summary.Medium,
		CVELow:          cveResult.Summary.Low,
		LicenseIssueCount: licenseResult.IssueCount,
	})
	assert.Greater(t, secScore, 50.0, "healthy-saas security should be decent")

	// Infrastructure: data-only assessment (no numeric score)
	infraAssessment := scorer.AssessInfra(scorer.InfraInput{
		IaCDetected:        infraResult.HasIaC,
		CICDDetected:       infraResult.HasCICD,
		MonitoringDetected: infraResult.HasMonitoring,
	})
	assert.NotEmpty(t, infraAssessment.InvestmentLevel, "should have investment level")
}

// TestMultiLanguageScan_JavaEnterprise verifies multi-language (Java + Go) analysis.
func TestMultiLanguageScan_JavaEnterprise(t *testing.T) {
	root := testdata.FixturePath(testdata.JavaEnterprise)
	walkResult, err := walker.Walk(root)
	require.NoError(t, err)

	langResult := language.AggregateResults(walkResult.LanguageLOC)
	assert.True(t, langResult.HasTier1)
	assert.Contains(t, langResult.DetectedLanguages, "Java")
	// Go files are present but go.mod is renamed to .fixture, so Go detection
	// depends on .go file extension detection
	foundGo := false
	for _, f := range walkResult.Files {
		if f.Language == "Go" {
			foundGo = true
			break
		}
	}
	assert.True(t, foundGo, "should detect Go files in worker/")

	// Dependencies: ParseDependencies looks at root-level manifests; pom.xml is in api/
	depResult := deps.ParseDependencies(filepath.Join(root, "api"))
	assert.NotNil(t, depResult)
	assert.Contains(t, depResult.Ecosystems, "maven")

	// Complexity: should find functions in Java files
	var javaResults []*complexity.FileResult
	for _, f := range walkResult.Files {
		if f.Language == "Java" && !f.IsTest {
			fr, cErr := complexity.AnalyzeFile(f.Path, "Java")
			if cErr != nil {
				t.Logf("Java complexity error for %s: %v", f.RelPath, cErr)
			} else if fr != nil {
				javaResults = append(javaResults, fr)
			}
		}
	}
	javaSummary := complexity.Summarize(javaResults)
	assert.Greater(t, javaSummary.TotalFunctions, 0, "should find Java functions")

	// Infrastructure: should have CI/CD and Docker
	infraResult := infra.Analyze(root, walkResult.Files, nil)
	assert.True(t, infraResult.HasCICD, "java-enterprise should have CI/CD")
	assert.True(t, infraResult.HasIaC, "java-enterprise should have Docker")
}

// TestTier2Only_NoComplexityScoring verifies Tier 2 repos get LOC/tech
// stack but no complexity metrics.
func TestTier2Only_NoComplexityScoring(t *testing.T) {
	root := testdata.FixturePath(testdata.Tier2Only)
	walkResult, err := walker.Walk(root)
	require.NoError(t, err)

	langResult := language.AggregateResults(walkResult.LanguageLOC)
	assert.False(t, langResult.HasTier1, "tier2-only should have no Tier 1 languages")
	assert.Greater(t, langResult.TotalLOC, 0, "should still count LOC")

	// Complexity: no Tier 1 files → no complexity results
	var complexityResults []*complexity.FileResult
	for _, f := range walkResult.Files {
		if f.Tier == language.Tier1 && !f.IsTest {
			fr, cErr := complexity.AnalyzeFile(f.Path, f.Language)
			if cErr != nil {
				t.Logf("complexity analysis error for %s: %v", f.RelPath, cErr)
			} else if fr != nil {
				complexityResults = append(complexityResults, fr)
			}
		}
	}
	assert.Empty(t, complexityResults, "tier2-only should have no complexity results")

	// Tech stack should still detect IaC
	techResult := techstack.Detect(root, nil)
	assert.NotNil(t, techResult)

	// Infrastructure should detect Terraform, K8s, Docker
	infraResult := infra.Analyze(root, walkResult.Files, nil)
	assert.True(t, infraResult.HasIaC, "tier2-only should detect IaC")
}

// TestNeglectedProject_Metrics verifies the neglected-project fixture
// has expected metric deficiencies.
func TestNeglectedProject_Metrics(t *testing.T) {
	root := testdata.FixturePath(testdata.NeglectedProject)
	walkResult, err := walker.Walk(root)
	require.NoError(t, err)

	// Should detect PHP
	langResult := language.AggregateResults(walkResult.LanguageLOC)
	assert.True(t, langResult.HasTier1)
	assert.Contains(t, langResult.DetectedLanguages, "PHP")

	// Handoff: no tests, no readme
	handoffResult := handoff.Analyze(root, walkResult)
	assert.False(t, handoffResult.HasReadme, "neglected-project has no README")
	assert.Equal(t, 0.0, handoffResult.EstTestCoveragePct, "neglected-project has no tests")

	// Infrastructure: no CI/CD
	infraResult := infra.Analyze(root, walkResult.Files, nil)
	assert.False(t, infraResult.HasCICD, "neglected-project has no CI/CD")
}

// TestSecurityNightmare_SecretsDetected verifies the security-nightmare
// fixture detects planted secrets.
// Note: The secrets scanner skips files under testdata/fixtures/ paths
// (isTestOrFixture filter), so we copy the secret-containing file to a
// temp directory to exercise the detection path end-to-end.
func TestSecurityNightmare_SecretsDetected(t *testing.T) {
	root := testdata.FixturePath(testdata.SecurityNightmare)

	// Verify Ruby detection via walker
	walkResult, err := walker.Walk(root)
	require.NoError(t, err)
	langResult := language.AggregateResults(walkResult.LanguageLOC)
	assert.Contains(t, langResult.DetectedLanguages, "Ruby")

	// Copy the secret-containing file to a temp dir so it's not in testdata/
	dir := t.TempDir()
	srcData, err := os.ReadFile(filepath.Join(root, "app", "controllers", "api_controller.rb"))
	require.NoError(t, err)
	tmpPath := filepath.Join(dir, "api_controller.rb")
	require.NoError(t, os.WriteFile(tmpPath, srcData, 0644))

	tmpFiles := []walker.FileInfo{{Path: tmpPath, RelPath: "api_controller.rb", Language: "Ruby"}}
	secretsResult := secrets.Scan(tmpFiles)
	assert.Greater(t, secretsResult.SecretsCount, 0, "should detect planted secrets")

}

// TestMultiRepoAggregation simulates scanning multiple fixture repos and
// aggregating results. Uses 3 fixtures per spec.
func TestMultiRepoAggregation(t *testing.T) {
	fixtures := []string{testdata.HealthySaas, testdata.JavaEnterprise, testdata.NeglectedProject}
	var repoMetrics []scorer.RepoMetrics

	for _, name := range fixtures {
		root := testdata.FixturePath(name)
		walkResult, err := walker.Walk(root)
		require.NoError(t, err)

		// Complexity
		var complexityResults []*complexity.FileResult
		for _, f := range walkResult.Files {
			if f.Tier == language.Tier1 && !f.IsTest {
				fr, err := complexity.AnalyzeFile(f.Path, f.Language)
				if err == nil && fr != nil {
					complexityResults = append(complexityResults, fr)
				}
			}
		}
		summary := complexity.Summarize(complexityResults)

		// Duplication
		dupResult := duplication.Analyze(walkResult.Files)

		// File size
		fsResult := filesize.Analyze(walkResult.Files)

		// Handoff
		handoffResult := handoff.Analyze(root, walkResult)

		// Infrastructure
		infraResult := infra.Analyze(root, walkResult.Files, nil)

		repoMetrics = append(repoMetrics, scorer.RepoMetrics{
			LOC:                walkResult.TotalLOC,
			AvgComplexity:      summary.AvgComplexity,
			MaxComplexity:      summary.MaxComplexity,
			AvgNesting:         summary.AvgNesting,
			MaxNesting:         summary.MaxNesting,
			DuplicationPct:     dupResult.DuplicationPct,
			PctFilesOver500LOC: fsResult.PctOver500LOC,
			DocDensity:         models.DocDensity(handoffResult.DocDensity),
			HasReadme:          handoffResult.HasReadme,
			HasGitHistory:      false,
			IaCDetected:        infraResult.HasIaC,
			CICDDetected:       infraResult.HasCICD,
			MonitoringDetected: infraResult.HasMonitoring,
		})
	}

	agg := scorer.Aggregate(repoMetrics)
	// Aggregated LOC should be sum of per-repo LOC
	totalExpectedLOC := 0
	maxComplexity := 0
	for _, rm := range repoMetrics {
		totalExpectedLOC += rm.LOC
		if rm.MaxComplexity > maxComplexity {
			maxComplexity = rm.MaxComplexity
		}
	}
	assert.Equal(t, totalExpectedLOC, agg.TotalLOC, "aggregated LOC should be sum of repos")
	assert.Greater(t, agg.AvgComplexity, 0.0, "aggregated complexity should be positive")
	assert.Equal(t, maxComplexity, agg.MaxComplexity, "aggregated MaxComplexity should be global max")
	// healthy-saas and java-enterprise have CI/CD (OR logic)
	assert.True(t, agg.CICDDetected, "aggregated CI/CD should be true (OR logic)")
}

// TestAllTier1Languages walks all 4 Tier 1 fixtures and verifies that all
// 6 Tier 1 languages are detected across them:
//   - healthy-saas: JavaScript, Python (+ TypeScript detected as JavaScript)
//   - java-enterprise: Java, Go
//   - neglected-project: PHP
//   - security-nightmare: Ruby
func TestAllTier1Languages(t *testing.T) {
	tier1Fixtures := []string{
		testdata.HealthySaas,
		testdata.JavaEnterprise,
		testdata.NeglectedProject,
		testdata.SecurityNightmare,
	}

	allLanguages := make(map[string]bool)
	for _, name := range tier1Fixtures {
		root := testdata.FixturePath(name)
		walkResult, err := walker.Walk(root)
		require.NoError(t, err, "walking fixture %s", name)

		langResult := language.AggregateResults(walkResult.LanguageLOC)
		for _, lang := range langResult.DetectedLanguages {
			allLanguages[lang] = true
		}
	}

	// All 6 Tier 1 languages should be present across the 4 fixtures
	expectedTier1 := []string{"JavaScript", "Python", "Java", "Go", "PHP", "Ruby"}
	for _, lang := range expectedTier1 {
		assert.True(t, allLanguages[lang], "Tier 1 language %q should be detected across all fixtures, got: %v", lang, allLanguages)
	}
}

// TestJSONOutputValidation builds a ScanResult, writes it, reads it back,
// and validates required fields are present and no raw paths leak.
func TestJSONOutputValidation(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "test-output.json")

	gradeA := models.GradeA
	result := &models.ScanResult{
		Version:        "1.0",
		ScanID:         "test-scan-001",
		Timestamp:      "2026-03-13T00:00:00Z",
		ScannerVersion: "0.1.0-test",
		Repositories: []models.Repository{
			{
				Name:              "test-repo",
				PathHash:          "abc123",
				HeadCommitSHA:     "deadbeef1234567890abcdef1234567890abcdef",
				Languages:         map[string]float64{"Go": 80.0, "JavaScript": 20.0},
				FileCount:         50,
				LOC:               5000,
				Status:            models.RepoStatusAnalyzed,
				DetectedLanguages: []string{"Go", "JavaScript"},
			},
		},
		TotalLOC:       5000,
		TotalFileCount: 50,
		RepoCount:      1,
		TechStack: models.TechStack{
			Frameworks:       []string{"gin"},
			Runtimes:         []string{"go1.21"},
			Databases:        []string{"postgres"},
			ExternalServices: []string{"stripe"},
		},
		Metrics: models.Metrics{
			Maintainability: &models.Maintainability{
				Grade:              &gradeA,
				CyclomaticComplexity: models.ComplexityStats{Avg: 5.0, P90: 10, Max: 15},
				NestingDepth:       models.NestingStats{Avg: 2.0, Max: 5},
				DuplicationPct:     3.5,
				HotspotCount:       2,
				HotspotFiles: []models.HotspotFile{
					{FileHash: "hash1", Complexity: 15, LOC: 200, Repo: "test-repo", Path: "/real/path/file.go"},
				},
				PctFilesOver500LOC: 5.0,
			},
		},
		Activity: &models.Activity{
			Grade:               &gradeA,
			LastCommitDate:      "2026-03-10",
			DaysSinceLastCommit: 3,
			CommitVelocity: models.CommitVelocity{
				AvgPerMonth:  12.5,
				Trend:        models.TrendStable,
				Last12Months: []int{10, 12, 11, 13, 14, 12, 11, 13, 12, 14, 13, 15},
			},
			ActiveMonths:     12,
			TotalMonths:      12,
			ContributorCount: 5,
		},
		Detection: models.Detection{
			AI: models.AIDetection{
				LLMAPI:      true,
				LLMProvider: "openai",
			},
			Infrastructure: models.InfrastructureDetection{
				IaCDetected:               true,
				IaCTypes:                  []string{"terraform"},
				CICDDetected:              true,
				CICDProvider:              "github-actions",
				PostAcquisitionInvestment: "low",
			},
		},
		Summary: models.Summary{
			ScoredCategories: []string{"maintainability", "security"},
			OverallGrade:     &gradeA,
		},
		PricingTier: models.PricingTier{
			Tier:   models.PricingTierStandard,
			Reason: "5000 LOC",
		},
		Warnings: []models.Warning{
			{Code: "partial_analysis", Message: "CVE lookup skipped for npm"},
		},
	}

	// Write JSON
	err := output.WriteScanResult(result, outPath)
	require.NoError(t, err)

	// Read back
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	// Parse as generic JSON to validate structure
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	// Required top-level fields
	assert.Contains(t, parsed, "version")
	assert.Contains(t, parsed, "scan_id")
	assert.Contains(t, parsed, "timestamp")
	assert.Contains(t, parsed, "scanner_version")
	assert.Contains(t, parsed, "repositories")
	assert.Contains(t, parsed, "total_loc")
	assert.Contains(t, parsed, "total_file_count")
	assert.Contains(t, parsed, "repo_count")
	assert.Contains(t, parsed, "tech_stack")
	assert.Contains(t, parsed, "metrics")
	assert.Contains(t, parsed, "activity")
	assert.Contains(t, parsed, "detection")
	assert.Contains(t, parsed, "summary")
	assert.Contains(t, parsed, "pricing_tier")
	assert.Contains(t, parsed, "warnings")
	assert.Contains(t, parsed, "integrity")

	// Validate total_file_count and repo_count values
	assert.Equal(t, float64(50), parsed["total_file_count"], "total_file_count should be 50")
	assert.Equal(t, float64(1), parsed["repo_count"], "repo_count should be 1")

	// Validate tech_stack structure
	techStackRaw, ok := parsed["tech_stack"].(map[string]interface{})
	require.True(t, ok, "tech_stack should be an object")
	assert.Contains(t, techStackRaw, "frameworks")
	assert.Contains(t, techStackRaw, "runtimes")
	assert.Contains(t, techStackRaw, "databases")
	assert.Contains(t, techStackRaw, "external_services")

	// Validate activity structure
	activityRaw, ok := parsed["activity"].(map[string]interface{})
	require.True(t, ok, "activity should be an object")
	assert.Contains(t, activityRaw, "grade")
	assert.Contains(t, activityRaw, "last_commit_date")
	assert.Contains(t, activityRaw, "contributor_count")
	assert.Equal(t, float64(5), activityRaw["contributor_count"])

	// Validate detection structure
	detectionRaw, ok := parsed["detection"].(map[string]interface{})
	require.True(t, ok, "detection should be an object")
	assert.Contains(t, detectionRaw, "ai")
	assert.Contains(t, detectionRaw, "infrastructure")

	// Validate warnings is present as array
	warningsRaw, ok := parsed["warnings"].([]interface{})
	require.True(t, ok, "warnings should be an array")
	assert.Len(t, warningsRaw, 1)

	// Validate head_commit_sha per repo
	reposRaw, ok := parsed["repositories"].([]interface{})
	require.True(t, ok, "repositories should be an array")
	require.Len(t, reposRaw, 1)
	repoRaw, ok := reposRaw[0].(map[string]interface{})
	require.True(t, ok, "repository should be an object")
	assert.Equal(t, "deadbeef1234567890abcdef1234567890abcdef", repoRaw["head_commit_sha"],
		"head_commit_sha should be present in repository JSON")

	// Verify no raw file paths leak in JSON (hotspot Path has json:"-")
	jsonStr := string(data)
	assert.NotContains(t, jsonStr, "/real/path/file.go",
		"real file paths should not appear in JSON output")

	// Verify hotspot has file_hash but not path (safe type assertions)
	metricsRaw, ok := parsed["metrics"].(map[string]interface{})
	require.True(t, ok, "metrics should be an object")
	maintRaw, ok := metricsRaw["maintainability"].(map[string]interface{})
	require.True(t, ok, "maintainability should be an object")
	hotspotsRaw, ok := maintRaw["hotspot_files"].([]interface{})
	require.True(t, ok, "hotspot_files should be an array")
	require.Len(t, hotspotsRaw, 1)
	hotspot, ok := hotspotsRaw[0].(map[string]interface{})
	require.True(t, ok, "hotspot should be an object")
	assert.Contains(t, hotspot, "file_hash")
	_, hasPath := hotspot["path"]
	assert.False(t, hasPath, "path should not be in JSON (tagged json:\"-\")")

	// Terminal output SHOULD show real file paths for hotspots
	var termBuf bytes.Buffer
	formatter := &output.TerminalFormatter{Color: &output.ColorConfig{Enabled: false}}
	formatter.Format(&termBuf, result)
	termOutput := termBuf.String()
	assert.Contains(t, termOutput, "/real/path/file.go",
		"terminal output should show real file paths")

	// JSON round-trip: deserialize back into ScanResult and verify key fields
	var roundTrip models.ScanResult
	err = json.Unmarshal(data, &roundTrip)
	require.NoError(t, err, "JSON should deserialize back to ScanResult")
	assert.Equal(t, "1.0", roundTrip.Version)
	assert.Equal(t, "test-scan-001", roundTrip.ScanID)
	assert.Equal(t, 5000, roundTrip.TotalLOC)
	assert.Equal(t, 1, roundTrip.RepoCount)
	require.NotNil(t, roundTrip.Metrics.Maintainability)
	assert.Equal(t, 3.5, roundTrip.Metrics.Maintainability.DuplicationPct)
}

// TestWarningsArrayValidation creates a ScanResult with various warning codes,
// writes it to JSON, reads it back, and verifies the warnings array round-trips
// correctly. This validates that warning codes defined in the data contract
// (partial_analysis, cve_lookup_skipped, large_file_skipped, analyzer_timeout)
// survive JSON serialization/deserialization.
func TestWarningsArrayValidation(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "warnings-test.json")

	gradeB := models.GradeB
	warnings := []models.Warning{
		{Code: "partial_analysis", Message: "Some analyzers could not complete"},
		{Code: "cve_lookup_skipped", Message: "CVE lookup skipped for npm ecosystem", Repo: "frontend", Ecosystem: "npm"},
		{Code: "large_file_skipped", Message: "File exceeds 1MB limit", Repo: "backend"},
		{Code: "analyzer_timeout", Message: "Complexity analyzer timed out for repo", Repo: "monolith"},
	}

	result := &models.ScanResult{
		Version:        "1.0",
		ScanID:         "test-warnings-001",
		Timestamp:      "2026-03-16T00:00:00Z",
		ScannerVersion: "0.1.0-test",
		Repositories: []models.Repository{
			{
				Name:              "test-repo",
				PathHash:          "abc123",
				Languages:         map[string]float64{"Go": 100.0},
				FileCount:         10,
				LOC:               1000,
				Status:            models.RepoStatusAnalyzed,
				DetectedLanguages: []string{"Go"},
			},
		},
		TotalLOC:       1000,
		TotalFileCount: 10,
		RepoCount:      1,
		TechStack: models.TechStack{
			Frameworks:       []string{},
			Runtimes:         []string{"go1.21"},
			Databases:        []string{},
			ExternalServices: []string{},
		},
		Metrics: models.Metrics{
			Maintainability: &models.Maintainability{
				Grade:              &gradeB,
				CyclomaticComplexity: models.ComplexityStats{Avg: 3.0, P90: 8, Max: 12},
				NestingDepth:       models.NestingStats{Avg: 1.5, Max: 4},
				HotspotFiles:       []models.HotspotFile{},
			},
			Security: &models.Security{
				Grade:                &gradeB,
				CVEs:                 []models.CVE{},
				LicenseIssues:        []models.LicenseIssue{},
				CVEEcosystemsSkipped: []string{},
			},
			DependencyHealth: &models.DependencyHealth{NAReason: "No dependencies detected"},
			HandoffReadiness: &models.HandoffReadiness{Grade: &gradeB},
		},
		Summary: models.Summary{
			ScoredCategories: []string{"maintainability", "security"},
			OverallGrade:     &gradeB,
			TopRisks:         []models.Risk{},
			TopStrengths:     []models.Strength{},
		},
		PricingTier: models.PricingTier{Tier: models.PricingTierStandard, Reason: "1000 LOC"},
		Warnings:    warnings,
	}

	// Write JSON
	err := output.WriteScanResult(result, outPath)
	require.NoError(t, err)

	// Read back raw JSON
	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	// Parse as generic JSON to validate warnings array structure
	var parsed map[string]interface{}
	err = json.Unmarshal(data, &parsed)
	require.NoError(t, err)

	warningsRaw, ok := parsed["warnings"].([]interface{})
	require.True(t, ok, "warnings should be a JSON array")
	require.Len(t, warningsRaw, 4, "should have exactly 4 warnings")

	// Verify each warning has the expected code
	expectedCodes := []string{"partial_analysis", "cve_lookup_skipped", "large_file_skipped", "analyzer_timeout"}
	for i, w := range warningsRaw {
		wObj, ok := w.(map[string]interface{})
		require.True(t, ok, "each warning should be a JSON object")
		assert.Equal(t, expectedCodes[i], wObj["code"], "warning %d should have code %q", i, expectedCodes[i])
		assert.NotEmpty(t, wObj["message"], "warning %d should have a message", i)
	}

	// Verify optional fields are present where set
	w1 := warningsRaw[1].(map[string]interface{})
	assert.Equal(t, "frontend", w1["repo"], "cve_lookup_skipped warning should have repo field")
	assert.Equal(t, "npm", w1["ecosystem"], "cve_lookup_skipped warning should have ecosystem field")

	// JSON round-trip: deserialize back into ScanResult and verify warnings
	var roundTrip models.ScanResult
	err = json.Unmarshal(data, &roundTrip)
	require.NoError(t, err, "JSON should deserialize back to ScanResult")
	require.Len(t, roundTrip.Warnings, 4, "round-trip should preserve all 4 warnings")

	for i, w := range roundTrip.Warnings {
		assert.Equal(t, expectedCodes[i], w.Code, "round-trip warning %d code mismatch", i)
		assert.NotEmpty(t, w.Message, "round-trip warning %d message should not be empty", i)
	}

	// Verify specific fields survived the round-trip
	assert.Equal(t, "frontend", roundTrip.Warnings[1].Repo)
	assert.Equal(t, "npm", roundTrip.Warnings[1].Ecosystem)
	assert.Equal(t, "backend", roundTrip.Warnings[2].Repo)
	assert.Equal(t, "monolith", roundTrip.Warnings[3].Repo)
}
