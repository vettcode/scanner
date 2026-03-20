package integration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"testing"
	"time"

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

// contractFixturesDir returns the absolute path to testdata/contract-fixtures/.
func contractFixturesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("runtime.Caller failed")
	}
	// This file is at internal/integration/contract_test.go.
	// Project root is two levels up.
	projectRoot := filepath.Join(filepath.Dir(filename), "..", "..")
	return filepath.Join(projectRoot, "testdata", "contract-fixtures")
}

// scoringFixture captures raw metrics alongside computed grades for a fixture.
type scoringFixture struct {
	Name string `json:"name"`

	// Raw metrics
	AvgComplexity      float64 `json:"avg_complexity"`
	MaxComplexity      int     `json:"max_complexity"`
	P90Complexity      int     `json:"p90_complexity"`
	AvgNesting         float64 `json:"avg_nesting"`
	MaxNesting         int     `json:"max_nesting"`
	DuplicationPct     float64 `json:"duplication_pct"`
	PctFilesOver500LOC float64 `json:"pct_files_over_500loc"`
	SecretsCount       int     `json:"secrets_count"`
	CVECritical        int     `json:"cve_critical"`
	CVEHigh            int     `json:"cve_high"`
	CVEMedium          int     `json:"cve_medium"`
	CVELow             int     `json:"cve_low"`
	LicenseIssueCount  int     `json:"license_issue_count"`
	EstTestCoveragePct float64 `json:"est_test_coverage_pct"`
	DocDensity         string  `json:"doc_density"`
	EnvVarCount        int     `json:"env_var_count"`
	HasReadme          bool    `json:"has_readme"`
	HasCICD            bool    `json:"has_cicd"`
	HasIaC             bool    `json:"has_iac"`
	HasMonitoring      bool    `json:"has_monitoring"`
	TotalLOC           int     `json:"total_loc"`

	// Computed scores
	MaintainabilityScore float64      `json:"maintainability_score"`
	MaintainabilityGrade models.Grade `json:"maintainability_grade"`
	SecurityScore        float64      `json:"security_score"`
	SecurityGrade        models.Grade `json:"security_grade"`
	HandoffScore         float64      `json:"handoff_score"`
	HandoffGrade         models.Grade `json:"handoff_grade"`
	InfraInvestment      string       `json:"infra_investment"`
	OverallScore         float64      `json:"overall_score"`
	OverallGrade         models.Grade `json:"overall_grade"`

	// Pricing
	PricingTier models.PricingTierName `json:"pricing_tier"`
}

// fixtureSpec holds per-fixture configuration for analysis.
type fixtureSpec struct {
	name       string
	depRoots   []string // sub-directories containing dependency manifests
	hasSecrets bool     // whether secrets must be scanned from a temp dir
}

// TestGenerateContractFixtures walks each of the 3 main fixtures, runs all
// analyzers, builds full ScanResults with metrics/scoring/signing, and writes
// them as signed fixture JSON files. It also verifies the signature round-trips
// and saves raw metrics alongside computed grades as scoring fixtures.
func TestGenerateContractFixtures(t *testing.T) {
	outDir := contractFixturesDir()
	require.NoError(t, os.MkdirAll(outDir, 0755))

	specs := []fixtureSpec{
		{
			name:     testdata.HealthySaas,
			depRoots: []string{"frontend", "backend"},
		},
		{
			name:     testdata.NeglectedProject,
			depRoots: []string{""}, // root-level composer.json
		},
		{
			name:       testdata.SecurityNightmare,
			depRoots:   []string{""}, // root-level Gemfile
			hasSecrets: true,
		},
	}

	var allScoringFixtures []scoringFixture

	for _, spec := range specs {
		t.Run(spec.name, func(t *testing.T) {
			root := testdata.FixturePath(spec.name)

			// ----- Walk -----
			walkResult, err := walker.Walk(root)
			require.NoError(t, err)

			langResult := language.AggregateResults(walkResult.LanguageLOC)

			// ----- Complexity -----
			var complexityResults []*complexity.FileResult
			var allFuncComplexities []int
			for _, f := range walkResult.Files {
				if f.Tier == language.Tier1 && !f.IsTest {
					fr, cErr := complexity.AnalyzeFile(f.Path, f.Language)
					if cErr != nil {
						t.Logf("complexity analysis error for %s: %v", f.RelPath, cErr)
					} else if fr != nil {
						complexityResults = append(complexityResults, fr)
						for _, fn := range fr.Functions {
							allFuncComplexities = append(allFuncComplexities, fn.Complexity)
						}
					}
				}
			}
			summary := complexity.Summarize(complexityResults)
			p90 := computeP90(allFuncComplexities)

			// ----- Duplication -----
			dupResult := duplication.Analyze(walkResult.Files)
			require.NotNil(t, dupResult)

			// ----- File size -----
			fsResult := filesize.Analyze(walkResult.Files)
			require.NotNil(t, fsResult)

			// ----- Secrets -----
			var secretsResult *secrets.Result
			if spec.hasSecrets {
				// Copy all files to a temp dir so they are not under testdata/
				secretsResult = scanSecretsFromTempCopy(t, root, walkResult)
			} else {
				secretsResult = secrets.Scan(walkResult.Files)
			}
			require.NotNil(t, secretsResult)

			// ----- Dependencies -----
			var allDeps []deps.Dependency
			var allEcosystems []string
			ecoSet := make(map[string]bool)
			for _, dr := range spec.depRoots {
				depRoot := root
				if dr != "" {
					depRoot = filepath.Join(root, dr)
				}
				depResult := deps.ParseDependencies(depRoot)
				allDeps = append(allDeps, depResult.Dependencies...)
				for _, eco := range depResult.Ecosystems {
					if !ecoSet[eco] {
						ecoSet[eco] = true
						allEcosystems = append(allEcosystems, eco)
					}
				}
			}

			// ----- Licenses -----
			licenseResult := security.DetectLicenses(root)
			require.NotNil(t, licenseResult)

			// ----- CVEs (offline mode) -----
			cveResult := security.LookupCVEs(allDeps, true)
			require.NotNil(t, cveResult)

			// ----- Tech stack -----
			var depNames []string
			for _, d := range allDeps {
				depNames = append(depNames, d.Name)
			}
			techResult := techstack.Detect(root, depNames)
			require.NotNil(t, techResult)

			// ----- AI detection -----
			aiResult := aidetect.Detect(depNames, walkResult.Files)
			require.NotNil(t, aiResult)

			// ----- Infrastructure -----
			infraResult := infra.Analyze(root, walkResult.Files, depNames)
			require.NotNil(t, infraResult)

			// ----- Handoff -----
			handoffResult := handoff.Analyze(root, walkResult)
			require.NotNil(t, handoffResult)

			// ----- Scoring -----
			maintScore := scorer.ScoreMaintainability(scorer.MaintainabilityInput{
				AvgComplexity:      summary.AvgComplexity,
				DuplicationPct:     dupResult.DuplicationPct,
				AvgNesting:         summary.AvgNesting,
				PctFilesOver500LOC: fsResult.PctOver500LOC,
			})
			maintGrade := scorer.ScoreToGrade(maintScore)

			secScore := scorer.ScoreSecurity(scorer.SecurityInput{
				SecretsCount:      secretsResult.SecretsCount,
				CVECritical:       cveResult.Summary.Critical,
				CVEHigh:           cveResult.Summary.High,
				CVEMedium:         cveResult.Summary.Medium,
				CVELow:            cveResult.Summary.Low,
				LicenseIssueCount: licenseResult.IssueCount,
			})
			secGrade := scorer.ScoreToGrade(secScore)

			handoffScore := scorer.ScoreHandoff(scorer.HandoffInput{
				EstTestCoveragePct: handoffResult.EstTestCoveragePct,
				DocDensity:         models.DocDensity(handoffResult.DocDensity),
				EnvVarCount:        handoffResult.EnvVarCount,
			})
			handoffGrade := scorer.ScoreToGrade(handoffScore)

			infraAssessment := scorer.AssessInfra(scorer.InfraInput{
				IaCDetected:        infraResult.HasIaC,
				CICDDetected:       infraResult.HasCICD,
				MonitoringDetected: infraResult.HasMonitoring,
			})

			// Build category scores for overall calculation (5 scored categories; SRE is data-only)
			categoryScores := []scorer.CategoryScore{
				{Name: "maintainability", Score: maintScore},
				{Name: "security", Score: secScore},
				{Name: "handoff_readiness", Score: handoffScore},
			}
			overallScore := scorer.OverallScore(categoryScores)
			overallGrade := scorer.ScoreToGrade(overallScore)

			// ----- Pricing tier -----
			pricingTier := scorer.DeterminePricingTier(walkResult.TotalLOC)

			// ----- Hotspot files -----
			hotspots := extractHotspotFiles(complexityResults, spec.name)

			// ----- Build language percentages for Repository -----
			langPcts := make(map[string]float64)
			var detectedLangs []string
			for lang, pct := range langResult.Percentages {
				langPcts[lang] = pct
			}
			for lang := range langResult.Languages {
				detectedLangs = append(detectedLangs, lang)
			}
			sort.Strings(detectedLangs)

			// ----- Build CVE list for JSON -----
			var cveList []models.CVE
			for _, v := range cveResult.Vulnerabilities {
				cveList = append(cveList, models.CVE{
					ID:             v.ID,
					Severity:       models.Severity(v.Severity),
					Package:        v.Package,
					CurrentVersion: v.Version,
					FixedIn:        v.FixedVersion,
					Repo:           spec.name,
				})
			}
			if cveList == nil {
				cveList = []models.CVE{}
			}

			// ----- Build license issues for JSON -----
			var licenseIssues []models.LicenseIssue
			for _, li := range licenseResult.Issues {
				licenseIssues = append(licenseIssues, models.LicenseIssue{
					Package: li.Package,
					License: li.License,
					Issue:   li.Reason,
					Repo:    spec.name,
				})
			}
			if licenseIssues == nil {
				licenseIssues = []models.LicenseIssue{}
			}

			// ----- Build tech stack -----
			techStackOut := models.TechStack{
				Frameworks: techResult.Frameworks,
			}
			var runtimes []string
			for _, rt := range techResult.Runtimes {
				runtimes = append(runtimes, rt.Name)
			}
			techStackOut.Runtimes = runtimes
			techStackOut.Databases = techResult.Databases
			techStackOut.ExternalServices = techResult.Services

			// Ensure no nil slices
			if techStackOut.Frameworks == nil {
				techStackOut.Frameworks = []string{}
			}
			if techStackOut.Runtimes == nil {
				techStackOut.Runtimes = []string{}
			}
			if techStackOut.Databases == nil {
				techStackOut.Databases = []string{}
			}
			if techStackOut.ExternalServices == nil {
				techStackOut.ExternalServices = []string{}
			}

			// ----- Build AI detection -----
			llmProvider := ""
			if len(aiResult.LLMProviders) > 0 {
				llmProvider = aiResult.LLMProviders[0]
			}
			vectorDBName := ""
			if len(aiResult.VectorDBProviders) > 0 {
				vectorDBName = aiResult.VectorDBProviders[0]
			}

			// ----- Build infra detection -----
			cicdProvider := ""
			if len(infraResult.CICDProviders) > 0 {
				cicdProvider = infraResult.CICDProviders[0]
			}

			// ----- Build scored categories -----
			scoredCategories := []string{"maintainability", "security", "handoff_readiness"}

			// ----- Build summary -----
			summaryOut := models.Summary{
				ScoredCategories: scoredCategories,
				OverallGrade:     &overallGrade,
				TopRisks:         []models.Risk{},
				TopStrengths:     []models.Strength{},
			}

			// ----- Build repo path hash -----
			pathHash := fmt.Sprintf("%x", sha256.Sum256([]byte(root)))[:16]

			// ----- Assemble ScanResult -----
			now := time.Now().UTC()
			result := &models.ScanResult{
				Version:        "1.0",
				ScanID:         fmt.Sprintf("contract-test-%s", spec.name),
				Timestamp:      now.Format(time.RFC3339),
				ScannerVersion: "0.1.0-test",
				Repositories: []models.Repository{
					{
						Name:              spec.name,
						PathHash:          pathHash,
						Languages:         langPcts,
						FileCount:         walkResult.TotalFiles,
						LOC:               walkResult.TotalLOC,
						Status:            models.RepoStatusAnalyzed,
						DetectedLanguages: detectedLangs,
					},
				},
				TotalLOC:       walkResult.TotalLOC,
				TotalFileCount: walkResult.TotalFiles,
				RepoCount:      1,
				TechStack:      techStackOut,
				Metrics: models.Metrics{
					Maintainability: &models.Maintainability{
						Grade: &maintGrade,
						CyclomaticComplexity: models.ComplexityStats{
							Avg: summary.AvgComplexity,
							P90: p90,
							Max: summary.MaxComplexity,
						},
						NestingDepth: models.NestingStats{
							Avg: summary.AvgNesting,
							Max: summary.MaxNesting,
						},
						DuplicationPct:     dupResult.DuplicationPct,
						HotspotCount:       len(hotspots),
						HotspotFiles:       hotspots,
						PctFilesOver500LOC: fsResult.PctOver500LOC,
					},
					Security: &models.Security{
						Grade:        &secGrade,
						SecretsFound: secretsResult.SecretsCount,
						CVEs:         cveList,
						CVESummary: models.CVESummary{
							Critical: cveResult.Summary.Critical,
							High:     cveResult.Summary.High,
							Medium:   cveResult.Summary.Medium,
							Low:      cveResult.Summary.Low,
						},
						LicenseIssues:        licenseIssues,
						LicenseIssueCount:    licenseResult.IssueCount,
						CVEEcosystemsSkipped: cveResult.EcosystemsSkipped,
					},
					HandoffReadiness: &models.HandoffReadiness{
						Grade:              &handoffGrade,
						EstTestCoveragePct: handoffResult.EstTestCoveragePct,
						DocDensity:         models.DocDensity(handoffResult.DocDensity),
						EnvVarCount:        handoffResult.EnvVarCount,
						HasReadme:          handoffResult.HasReadme,
						HasContributingGuide: handoffResult.HasContributing,
						HasEnvTemplate:     handoffResult.HasEnvTemplate,
						HasSetupScript:     handoffResult.HasSetupScript,
					},
				},
				Detection: models.Detection{
					AI: models.AIDetection{
						LLMAPI:             aiResult.HasLLMAPI,
						LLMProvider:        llmProvider,
						VectorDatabase:     aiResult.HasVectorDB,
						VectorDBName:       vectorDBName,
						RAGPipeline:        aiResult.HasRAGPipeline,
						MCPServers:         aiResult.HasMCP,
						FineTunedModels:    aiResult.HasFineTuning,
						TrainingPipeline:   aiResult.HasTrainingPipeline,
						ProprietaryDataset: aiResult.HasProprietaryData,
					},
					Infrastructure: models.InfrastructureDetection{
						IaCDetected:               infraResult.HasIaC,
						IaCTypes:                  infraResult.IaCTools,
						CICDDetected:              infraResult.HasCICD,
						CICDProvider:              cicdProvider,
						MonitoringDetected:        infraResult.HasMonitoring,
						MonitoringTools:           infraResult.MonitorTools,
						PostAcquisitionInvestment: string(infraAssessment.InvestmentLevel),
					},
				},
				Summary:     summaryOut,
				PricingTier: pricingTier,
				Warnings:    []models.Warning{},
			}

			// Ensure nil slices become empty arrays in JSON
			if result.Metrics.Security.CVEEcosystemsSkipped == nil {
				result.Metrics.Security.CVEEcosystemsSkipped = []string{}
			}
			if result.Metrics.Maintainability.HotspotFiles == nil {
				result.Metrics.Maintainability.HotspotFiles = []models.HotspotFile{}
			}
			if result.Detection.Infrastructure.IaCTypes == nil {
				result.Detection.Infrastructure.IaCTypes = []string{}
			}
			if result.Detection.Infrastructure.MonitoringTools == nil {
				result.Detection.Infrastructure.MonitoringTools = []string{}
			}

			// ----- Sign -----
			err = output.SignScanResult(result)
			require.NoError(t, err, "SignScanResult should succeed for %s", spec.name)

			// ----- Write signed fixture JSON -----
			signedPath := filepath.Join(outDir, fmt.Sprintf("signed-9a-%s.json", spec.name))
			err = output.WriteScanResult(result, signedPath)
			require.NoError(t, err, "WriteScanResult should succeed for %s", spec.name)

			t.Logf("Wrote signed fixture: %s", signedPath)

			// ----- Verify signature round-trip -----
			// Read back the file, unmarshal, and verify signature
			data, err := os.ReadFile(signedPath)
			require.NoError(t, err)

			var roundTrip models.ScanResult
			err = json.Unmarshal(data, &roundTrip)
			require.NoError(t, err, "JSON round-trip unmarshal should succeed for %s", spec.name)

			err = output.VerifyScannerSignature(&roundTrip)
			require.NoError(t, err, "Signature verification should pass after round-trip for %s", spec.name)

			// Validate key fields survived round-trip
			assert.Equal(t, "1.0", roundTrip.Version)
			assert.Equal(t, result.ScanID, roundTrip.ScanID)
			assert.Equal(t, result.TotalLOC, roundTrip.TotalLOC)
			assert.Equal(t, 1, roundTrip.RepoCount)
			assert.NotEmpty(t, roundTrip.Integrity.ScanChecksum)
			assert.NotEmpty(t, roundTrip.Integrity.ScannerSignature)
			assert.Equal(t, output.ScannerKeyID, roundTrip.Integrity.ScannerPublicKeyID)

			// ----- Build scoring fixture -----
			sf := scoringFixture{
				Name:               spec.name,
				AvgComplexity:      summary.AvgComplexity,
				MaxComplexity:      summary.MaxComplexity,
				P90Complexity:      p90,
				AvgNesting:         summary.AvgNesting,
				MaxNesting:         summary.MaxNesting,
				DuplicationPct:     dupResult.DuplicationPct,
				PctFilesOver500LOC: fsResult.PctOver500LOC,
				SecretsCount:       secretsResult.SecretsCount,
				CVECritical:        cveResult.Summary.Critical,
				CVEHigh:            cveResult.Summary.High,
				CVEMedium:          cveResult.Summary.Medium,
				CVELow:             cveResult.Summary.Low,
				LicenseIssueCount:  licenseResult.IssueCount,
				EstTestCoveragePct: handoffResult.EstTestCoveragePct,
				DocDensity:         handoffResult.DocDensity,
				EnvVarCount:        handoffResult.EnvVarCount,
				HasReadme:          handoffResult.HasReadme,
				HasCICD:            infraResult.HasCICD,
				HasIaC:             infraResult.HasIaC,
				HasMonitoring:      infraResult.HasMonitoring,
				TotalLOC:           walkResult.TotalLOC,

				MaintainabilityScore: maintScore,
				MaintainabilityGrade: maintGrade,
				SecurityScore:        secScore,
				SecurityGrade:        secGrade,
				HandoffScore:         handoffScore,
				HandoffGrade:         handoffGrade,
				InfraInvestment:      string(infraAssessment.InvestmentLevel),
				OverallScore:         overallScore,
				OverallGrade:         overallGrade,

				PricingTier: pricingTier.Tier,
			}
			allScoringFixtures = append(allScoringFixtures, sf)
		})
	}

	// ----- Write scoring fixtures -----
	scoringPath := filepath.Join(outDir, "scoring-fixtures.json")
	scoringJSON, err := json.MarshalIndent(allScoringFixtures, "", "  ")
	require.NoError(t, err)
	require.NoError(t, os.WriteFile(scoringPath, scoringJSON, 0644))
	t.Logf("Wrote scoring fixtures: %s", scoringPath)

	// Verify all 3 signed fixtures were created
	for _, spec := range specs {
		signedPath := filepath.Join(outDir, fmt.Sprintf("signed-9a-%s.json", spec.name))
		info, err := os.Stat(signedPath)
		require.NoError(t, err, "signed fixture should exist: %s", signedPath)
		assert.Greater(t, info.Size(), int64(100), "signed fixture should not be empty: %s", signedPath)
	}

	// Verify scoring fixtures were created with all 3 entries
	assert.Len(t, allScoringFixtures, 3, "should have scoring data for all 3 fixtures")
}

// scanSecretsFromTempCopy copies all files from the fixture to a temp directory
// (so they are not under testdata/) and runs secrets scanning there.
// This is needed because the secrets scanner skips files under testdata/ paths.
func scanSecretsFromTempCopy(t *testing.T, root string, walkResult *walker.WalkResult) *secrets.Result {
	t.Helper()
	dir := t.TempDir()

	// Copy the full fixture tree to the temp directory
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relPath, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dir, relPath)
		if d.IsDir() {
			return os.MkdirAll(target, 0755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0644)
	})
	require.NoError(t, err, "copying fixture to temp dir")

	// Build FileInfo list pointing to temp dir copies
	var tmpFiles []walker.FileInfo
	for _, f := range walkResult.Files {
		relPath, err := filepath.Rel(root, f.Path)
		require.NoError(t, err)
		tmpFiles = append(tmpFiles, walker.FileInfo{
			Path:     filepath.Join(dir, relPath),
			RelPath:  f.RelPath,
			Language: f.Language,
			Tier:     f.Tier,
			IsTest:   f.IsTest,
			LOC:      f.LOC,
		})
	}

	return secrets.Scan(tmpFiles)
}

// computeP90 computes the 90th percentile from a slice of int values.
func computeP90(values []int) int {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]int, len(values))
	copy(sorted, values)
	sort.Ints(sorted)
	idx := int(float64(len(sorted)) * 0.90)
	if idx >= len(sorted) {
		idx = len(sorted) - 1
	}
	return sorted[idx]
}

// extractHotspotFiles builds a list of hotspot files (functions with complexity > 10)
// from complexity results, capped at 10, sorted by complexity descending.
func extractHotspotFiles(results []*complexity.FileResult, repoName string) []models.HotspotFile {
	var hotspots []models.HotspotFile
	for _, fr := range results {
		if fr == nil {
			continue
		}
		maxComplexity := 0
		for _, fn := range fr.Functions {
			if fn.Complexity > maxComplexity {
				maxComplexity = fn.Complexity
			}
		}
		if maxComplexity > 10 {
			fileHash := fmt.Sprintf("%x", sha256.Sum256([]byte(fr.Path)))[:16]
			hotspots = append(hotspots, models.HotspotFile{
				FileHash:   fileHash,
				Complexity: maxComplexity,
				LOC:        countFileFunctions(fr),
				Repo:       repoName,
				Path:       fr.Path,
			})
		}
	}
	sort.Slice(hotspots, func(i, j int) bool {
		return hotspots[i].Complexity > hotspots[j].Complexity
	})
	if len(hotspots) > 10 {
		hotspots = hotspots[:10]
	}
	return hotspots
}

// countFileFunctions returns the number of functions in a FileResult (used as LOC proxy for hotspot).
func countFileFunctions(fr *complexity.FileResult) int {
	if fr == nil || len(fr.Functions) == 0 {
		return 0
	}
	// Use the span of the file (last function end line) as a LOC estimate
	maxLine := 0
	for _, fn := range fr.Functions {
		if fn.EndLine > maxLine {
			maxLine = fn.EndLine
		}
	}
	return maxLine
}
