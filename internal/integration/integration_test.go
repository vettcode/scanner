package integration

import (
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

	// Scoring: infrastructure
	infraScore := scorer.ScoreInfra(scorer.InfraInput{
		IaCDetected:        infraResult.HasIaC,
		CICDDetected:       infraResult.HasCICD,
		MonitoringDetected: infraResult.HasMonitoring,
	})
	assert.Greater(t, infraScore, 0.0, "should have some infra score")
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

// TestNeglectedProject_RedFlags verifies the neglected-project fixture
// triggers expected red flags.
func TestNeglectedProject_RedFlags(t *testing.T) {
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

	// Red flags evaluation
	redFlags := scorer.EvaluateRedFlags(scorer.RedFlagInput{
		SecretsCount:        0,
		CVECritical:         0,
		EstTestCoveragePct:  0,
		CICDDetected:        false,
		HasReadme:           false,
		HasGitHistory:       false,
		UnmaintainedPct:     0,
	})
	assert.GreaterOrEqual(t, redFlags.Count, 3, "should flag no tests, no CI/CD, no readme")

	// Verify specific flags present
	flagCodes := make(map[models.RedFlagCode]bool)
	for _, f := range redFlags.Flags {
		flagCodes[f.Flag] = true
	}
	assert.True(t, flagCodes[models.RedFlagNoTests], "should flag no tests")
	assert.True(t, flagCodes[models.RedFlagNoCICD], "should flag no CI/CD")
	assert.True(t, flagCodes[models.RedFlagNoReadme], "should flag no readme")
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

	// Red flags should include secrets_detected
	redFlags := scorer.EvaluateRedFlags(scorer.RedFlagInput{
		SecretsCount:       secretsResult.SecretsCount,
		EstTestCoveragePct: 10.0, // has some tests
		CICDDetected:       false,
		HasReadme:          true,
		HasGitHistory:      false,
	})
	flagCodes := make(map[models.RedFlagCode]bool)
	for _, f := range redFlags.Flags {
		flagCodes[f.Flag] = true
	}
	assert.True(t, flagCodes[models.RedFlagSecretsDetected], "should flag secrets detected")
}

// TestMultiRepoAggregation simulates scanning multiple fixture repos and
// aggregating results.
func TestMultiRepoAggregation(t *testing.T) {
	fixtures := []string{testdata.HealthySaas, testdata.JavaEnterprise}
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
	for _, rm := range repoMetrics {
		totalExpectedLOC += rm.LOC
	}
	assert.Equal(t, totalExpectedLOC, agg.TotalLOC, "aggregated LOC should be sum of repos")
	assert.Greater(t, agg.AvgComplexity, 0.0, "aggregated complexity should be positive")
	// Both fixtures have CI/CD
	assert.True(t, agg.CICDDetected, "aggregated CI/CD should be true (OR logic)")
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
		Summary: models.Summary{
			ScoredCategories: []string{"maintainability", "security"},
			OverallGrade:     &gradeA,
		},
		PricingTier: models.PricingTier{
			Tier:   models.PricingTierStandard,
			Reason: "5000 LOC",
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
	assert.Contains(t, parsed, "metrics")
	assert.Contains(t, parsed, "summary")
	assert.Contains(t, parsed, "pricing_tier")
	assert.Contains(t, parsed, "integrity")

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
