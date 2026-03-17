package cli

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"log/slog"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/spf13/cobra"

	"github.com/vettcode/scanner/internal/analyzer/activity"
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
	"github.com/vettcode/scanner/internal/config"
	"github.com/vettcode/scanner/internal/exclusion"
	"github.com/vettcode/scanner/internal/language"
	"github.com/vettcode/scanner/internal/output"
	"github.com/vettcode/scanner/internal/scorer"
	"github.com/vettcode/scanner/internal/updater"
	"github.com/vettcode/scanner/internal/walker"
	"github.com/vettcode/scanner/pkg/models"
)

// runScan is the main scan orchestrator. It wires together:
// walker → language detection → analyzers → scoring → output.
func runScan(cmd *cobra.Command, args []string) error {
	// Load configuration
	cfg, err := config.Load(cmd)
	if err != nil {
		return err
	}

	// Parse labels flag (only use if explicitly set by the user)
	var labels []string
	if cmd.Flags().Changed("label") {
		labels, _ = cmd.Flags().GetStringSlice("label")
	}

	// Parse and validate paths
	repos, err := config.ParsePaths(args, labels)
	if err != nil {
		return err
	}

	// Validate output path before doing any work
	if cfg.Format != "terminal" {
		if err := config.ValidateOutputPath(cfg.Output); err != nil {
			return err
		}
	}

	// Set up timeout context
	ctx, cancel := context.WithTimeout(cmd.Context(), cfg.Timeout)
	defer cancel()

	// Set up progress spinner
	showProgress := !cfg.Quiet && cfg.Format != "json"
	progress := output.NewProgress(os.Stderr, showProgress)
	progress.Start()
	defer progress.Stop()

	// Phase 1: Walk and detect languages for each repo
	progress.SetPhase(output.PhaseLanguageDetection)

	var repoResults []repoData
	for _, repo := range repos {
		progress.SetPhaseDetail(output.PhaseLanguageDetection, repo.Name)
		wr, err := walker.Walk(repo.Path)
		if err != nil {
			return fmt.Errorf("walk %s: %w", repo.Name, err)
		}

		langResult := language.AggregateResults(wr.LanguageLOC)
		repoResults = append(repoResults, repoData{
			input:   repo,
			walk:    wr,
			langDet: langResult,
		})
	}

	// Phase 2: Analyze each repository
	progress.SetPhase(output.PhaseAnalyzing)

	var (
		allRepoMetrics      []scorer.RepoMetrics
		allRepositories     []models.Repository
		allCVEs             []models.CVE
		allLicenseIssues    []models.LicenseIssue
		allWarnings         []models.Warning
		allHotspots         []models.HotspotFile
		allSecretFindings   []models.SecretFinding
		allFuncComplexities []int // for global P90 computation
		globalTechStack     *techstack.Result
		globalAIDetect      *aidetect.Result
		globalInfra         *infra.Result
		globalActivity      *activity.Result
		totalLOC            int
		totalFiles          int
		cveSummary          models.CVESummary
		ecosystemsSkipped   []string
	)

	for i := range repoResults {
		rd := &repoResults[i]
		repo := rd.input
		wr := rd.walk

		select {
		case <-ctx.Done():
			return fmt.Errorf("scan timed out")
		default:
		}

		progress.SetPhaseDetail(output.PhaseAnalyzing, repo.Name)

		// Complexity analysis (Tier 1 languages via tree-sitter, Go via goast)
		var complexityResults []*complexity.FileResult
		tokenStreams := make(map[string][]duplication.Token)

		for _, f := range wr.Files {
			if f.IsTest || f.Tier != language.Tier1 || exclusion.IsAuxiliaryPath(f.RelPath) {
				continue
			}
			cr, err := complexity.AnalyzeFile(f.Path, f.Language)
			if err != nil {
				slog.Debug("complexity analysis failed", "path", f.RelPath, "error", err)
				continue
			}
			if cr == nil {
				continue // unsupported language for complexity
			}
			complexityResults = append(complexityResults, cr)

			// Convert complexity tokens to duplication tokens
			if len(cr.Tokens) > 0 {
				dupTokens := make([]duplication.Token, len(cr.Tokens))
				for j, t := range cr.Tokens {
					dupTokens[j] = duplication.Token{Value: t.Value, Line: t.Line}
				}
				tokenStreams[f.Path] = dupTokens
			}
		}

		complexitySummary := complexity.Summarize(complexityResults)

		// Collect function complexities for global P90
		for _, cr := range complexityResults {
			for _, fn := range cr.Functions {
				allFuncComplexities = append(allFuncComplexities, fn.Complexity)
			}
		}

		// Hotspot files (functions with complexity > 10)
		repoHotspots := extractHotspots(complexityResults, repo.Name, repo.Path)
		allHotspots = append(allHotspots, repoHotspots...)

		// Duplication detection
		dupResult := duplication.Analyze(wr.Files, tokenStreams)

		// File size distribution
		fsResult := filesize.Analyze(wr.Files)

		// Secrets detection
		secretsResult := secrets.Scan(wr.Files)
		for _, f := range secretsResult.Findings {
			allSecretFindings = append(allSecretFindings, models.SecretFinding{
				Path:     f.RelPath,
				Line:     f.Line,
				Name:     f.Name,
				Category: f.Category,
			})
		}

		// Dependency parsing
		depsResult := deps.ParseDependencies(repo.Path)
		rd.depsParsed = depsResult

		// Dependency names for downstream analyzers
		depNames := make([]string, len(depsResult.Dependencies))
		for j, d := range depsResult.Dependencies {
			depNames[j] = d.Name
		}

		// License detection
		licenseResult := security.DetectLicenses(repo.Path)
		for _, issue := range licenseResult.Issues {
			allLicenseIssues = append(allLicenseIssues, models.LicenseIssue{
				Package: issue.Package,
				License: issue.License,
				Issue:   issue.Reason,
				Repo:    repo.Name,
			})
		}

		// CVE lookup
		cveResult := security.LookupCVEs(depsResult.Dependencies, cfg.Offline)
		for _, v := range cveResult.Vulnerabilities {
			allCVEs = append(allCVEs, models.CVE{
				ID:             v.ID,
				Severity:       models.Severity(v.Severity),
				Package:        v.Package,
				CurrentVersion: v.Version,
				FixedIn:        v.FixedVersion,
				Repo:           repo.Name,
			})
		}
		cveSummary.Critical += cveResult.Summary.Critical
		cveSummary.High += cveResult.Summary.High
		cveSummary.Medium += cveResult.Summary.Medium
		cveSummary.Low += cveResult.Summary.Low
		ecosystemsSkipped = appendUnique(ecosystemsSkipped, cveResult.EcosystemsSkipped...)
		for _, w := range cveResult.Warnings {
			allWarnings = append(allWarnings, models.Warning{
				Code:    "cve_lookup",
				Message: w,
				Repo:    repo.Name,
			})
		}

		// Dependency health (from dep ages — offline mode only has age data from lockfile dates)
		depAges := buildDepAges(depsResult.Dependencies)
		healthResult := deps.AnalyzeHealth(depAges)

		// Tech stack detection (merge across repos)
		tsResult := techstack.Detect(repo.Path, depNames)
		globalTechStack = mergeTechStack(globalTechStack, tsResult)

		// AI detection (merge across repos)
		aiResult := aidetect.Detect(depNames, wr.Files)
		globalAIDetect = mergeAIDetect(globalAIDetect, aiResult)

		// Infrastructure detection (merge across repos)
		infraResult := infra.Analyze(repo.Path, wr.Files, depNames)
		globalInfra = mergeInfra(globalInfra, infraResult)

		// Handoff readiness
		handoffResult := handoff.Analyze(repo.Path, wr)

		// Git activity (skip if --no-git)
		var actResult *activity.Result
		if !cfg.NoGit {
			actResult = activity.Analyze(repo.Path)
			globalActivity = mergeActivity(globalActivity, actResult)
		} else {
			actResult = &activity.Result{}
		}

		// Build per-repo metrics for aggregation
		rm := scorer.RepoMetrics{
			LOC:                wr.TotalLOC,
			AvgComplexity:      complexitySummary.AvgComplexity,
			MaxComplexity:      complexitySummary.MaxComplexity,
			AvgNesting:         complexitySummary.AvgNesting,
			MaxNesting:         complexitySummary.MaxNesting,
			DuplicationPct:     dupResult.DuplicationPct,
			PctFilesOver500LOC: fsResult.PctOver500LOC,
			SecretsCount:       secretsResult.SecretsCount,
			CVECritical:        cveResult.Summary.Critical,
			CVEHigh:            cveResult.Summary.High,
			CVEMedium:          cveResult.Summary.Medium,
			CVELow:             cveResult.Summary.Low,
			LicenseIssueCount:  licenseResult.IssueCount,
			EstTestCoveragePct: handoffResult.EstTestCoveragePct,
			EnvVarCount:        handoffResult.EnvVarCount,
			DocDensity:         models.DocDensity(handoffResult.DocDensity),
			MedianAgeMonths:    int(math.Round(healthResult.MedianAgeMonths)),
			UnmaintainedPct:    healthResult.UnmaintainedPct,
			DaysSinceLastCommit: actResult.DaysSinceLastCommit,
			AvgCommitsPerMonth: actResult.CommitVelocity,
			ActiveMonths:       monthBits(actResult.MonthlyCommits),
			IaCDetected:        infraResult.HasIaC,
			CICDDetected:       infraResult.HasCICD,
			MonitoringDetected: infraResult.HasMonitoring,
			HasReadme:          handoffResult.HasReadme,
			HasGitHistory:      actResult.HasGit,
		}
		allRepoMetrics = append(allRepoMetrics, rm)

		// Build repository entry
		pathHash := hashPath(repo.Path)
		repoEntry := models.Repository{
			Name:              repo.Name,
			PathHash:          pathHash,
			Languages:         rd.langDet.Percentages,
			FileCount:         wr.TotalFiles,
			LOC:               wr.TotalLOC,
			Status:            models.RepoStatusAnalyzed,
			DetectedLanguages: rd.langDet.DetectedLanguages,
		}
		if actResult.HeadSHA != "" {
			repoEntry.HeadCommitSHA = actResult.HeadSHA
		}
		allRepositories = append(allRepositories, repoEntry)

		totalLOC += wr.TotalLOC
		totalFiles += wr.TotalFiles
	}

	// Phase 3: Score and grade
	progress.SetPhase(output.PhaseScoring)

	agg := scorer.Aggregate(allRepoMetrics)

	// Sort hotspots by complexity descending, take top 10
	sort.Slice(allHotspots, func(i, j int) bool {
		return allHotspots[i].Complexity > allHotspots[j].Complexity
	})
	if len(allHotspots) > 10 {
		allHotspots = allHotspots[:10]
	}

	// P90 across all repos
	globalP90 := computeP90FromSlice(allFuncComplexities)

	// Score each category
	maintScore := scorer.ScoreMaintainability(scorer.MaintainabilityInput{
		AvgComplexity:      agg.AvgComplexity,
		DuplicationPct:     agg.DuplicationPct,
		AvgNesting:         agg.AvgNesting,
		PctFilesOver500LOC: agg.PctFilesOver500LOC,
	})
	maintGrade := scorer.ScoreToGrade(maintScore)

	secScore := scorer.ScoreSecurity(scorer.SecurityInput{
		SecretsCount:      agg.SecretsCount,
		CVECritical:       agg.CVECritical,
		CVEHigh:           agg.CVEHigh,
		CVEMedium:         agg.CVEMedium,
		CVELow:            agg.CVELow,
		LicenseIssueCount: agg.LicenseIssueCount,
	})
	secGrade := scorer.ScoreToGrade(secScore)

	handoffScore := scorer.ScoreHandoff(scorer.HandoffInput{
		EstTestCoveragePct: agg.EstTestCoveragePct,
		DocDensity:         agg.DocDensity,
		EnvVarCount:        agg.EnvVarCount,
	})
	handoffGrade := scorer.ScoreToGrade(handoffScore)

	// Category scores for overall calculation
	var categoryScores []scorer.CategoryScore
	var scoredCategories []string

	categoryScores = append(categoryScores,
		scorer.CategoryScore{Name: "maintainability", Score: maintScore},
		scorer.CategoryScore{Name: "security", Score: secScore},
		scorer.CategoryScore{Name: "handoff_readiness", Score: handoffScore},
	)
	scoredCategories = append(scoredCategories, "maintainability", "security", "handoff_readiness")

	// Dependency health (N/A if no deps)
	var depHealthMetric *models.DependencyHealth
	hasDeps := false
	for _, rd := range repoResults {
		if len(rd.depsParsed.Dependencies) > 0 {
			hasDeps = true
			break
		}
	}
	if hasDeps {
		dhScore := scorer.ScoreDependencyHealth(scorer.DependencyHealthInput{
			MedianAgeMonths: agg.MedianAgeMonths,
			UnmaintainedPct: agg.UnmaintainedPct,
		})
		dhGrade := scorer.ScoreToGrade(dhScore)
		depHealthMetric = &models.DependencyHealth{
			Grade:            &dhGrade,
			MedianAgeMonths:  agg.MedianAgeMonths,
			UnmaintainedPct:  agg.UnmaintainedPct,
		}
		categoryScores = append(categoryScores, scorer.CategoryScore{Name: "dependency_health", Score: dhScore})
		scoredCategories = append(scoredCategories, "dependency_health")
	} else {
		depHealthMetric = &models.DependencyHealth{
			NAReason: "No dependencies detected",
		}
	}

	// Activity (N/A if --no-git or no git history)
	var activityMetric *models.Activity
	if !cfg.NoGit && agg.HasGitHistory && globalActivity != nil {
		actScore := scorer.ScoreActivity(scorer.ActivityInput{
			DaysSinceLastCommit: agg.DaysSinceLastCommit,
			AvgCommitsPerMonth:  agg.AvgCommitsPerMonth,
			ActiveMonths:        agg.ActiveMonths,
			RepoAgeMonths:       globalActivity.RepoAgeMonths,
		})
		actGrade := scorer.ScoreToGrade(actScore)

		lastCommitStr := ""
		if globalActivity.LastCommitDate != nil {
			lastCommitStr = globalActivity.LastCommitDate.Format(time.RFC3339)
		}

		monthly := make([]int, 12)
		for j := 0; j < 12; j++ {
			monthly[j] = globalActivity.MonthlyCommits[j]
		}

		activityMetric = &models.Activity{
			Grade:              &actGrade,
			LastCommitDate:     lastCommitStr,
			DaysSinceLastCommit: agg.DaysSinceLastCommit,
			CommitVelocity: models.CommitVelocity{
				AvgPerMonth:  agg.AvgCommitsPerMonth,
				Trend:        models.Trend(globalActivity.Trend),
				Last12Months: monthly,
			},
			ActiveMonths:     agg.ActiveMonths,
			TotalMonths:      12,
			ContributorCount: globalActivity.ContributorCount,
		}
		categoryScores = append(categoryScores, scorer.CategoryScore{Name: "development_activity", Score: actScore})
		scoredCategories = append(scoredCategories, "development_activity")
	} else {
		reason := "No git history detected"
		if cfg.NoGit {
			reason = "Git analysis disabled (--no-git)"
		}
		activityMetric = &models.Activity{
			NAReason: reason,
		}
	}

	// Infrastructure score
	infraScore := scorer.ScoreInfra(scorer.InfraInput{
		IaCDetected:        agg.IaCDetected,
		CICDDetected:       agg.CICDDetected,
		MonitoringDetected: agg.MonitoringDetected,
	})
	infraGrade := scorer.ScoreToGrade(infraScore)
	categoryScores = append(categoryScores, scorer.CategoryScore{Name: "sre_infrastructure", Score: infraScore})
	scoredCategories = append(scoredCategories, "sre_infrastructure")

	// Overall score
	overall := scorer.OverallScore(categoryScores)
	overallGrade := scorer.ScoreToGrade(overall)

	// Red flags
	redFlags := scorer.EvaluateRedFlags(scorer.RedFlagInput{
		SecretsCount:        agg.SecretsCount,
		CVECritical:         agg.CVECritical,
		CVEHigh:             agg.CVEHigh,
		EstTestCoveragePct:  agg.EstTestCoveragePct,
		DaysSinceLastCommit: agg.DaysSinceLastCommit,
		UnmaintainedPct:     agg.UnmaintainedPct,
		CICDDetected:        agg.CICDDetected,
		HasReadme:           agg.HasReadme,
		HasGitHistory:       agg.HasGitHistory,
	})

	// Pricing tier
	pricingTier := scorer.DeterminePricingTier(totalLOC)

	// Build tech stack for output
	var techStackOut models.TechStack
	if globalTechStack != nil {
		runtimes := make([]string, len(globalTechStack.Runtimes))
		for j, r := range globalTechStack.Runtimes {
			if r.Version != "" {
				runtimes[j] = r.Name + " " + r.Version
			} else {
				runtimes[j] = r.Name
			}
		}
		techStackOut = models.TechStack{
			Frameworks:       globalTechStack.Frameworks,
			Runtimes:         runtimes,
			Databases:        globalTechStack.Databases,
			ExternalServices: globalTechStack.Services,
		}
	}

	// Build detection block
	var detection models.Detection
	if globalAIDetect != nil {
		llmProvider := ""
		if len(globalAIDetect.LLMProviders) > 0 {
			llmProvider = globalAIDetect.LLMProviders[0]
		}
		vectorDBName := ""
		if len(globalAIDetect.VectorDBProviders) > 0 {
			vectorDBName = globalAIDetect.VectorDBProviders[0]
		}
		detection.AI = models.AIDetection{
			LLMAPI:            globalAIDetect.HasLLMAPI,
			LLMProvider:       llmProvider,
			VectorDatabase:    globalAIDetect.HasVectorDB,
			VectorDBName:      vectorDBName,
			RAGPipeline:       globalAIDetect.HasRAGPipeline,
			MCPServers:        globalAIDetect.HasMCP,
			FineTunedModels:   globalAIDetect.HasFineTuning,
			TrainingPipeline:  globalAIDetect.HasTrainingPipeline,
			ProprietaryDataset: globalAIDetect.HasProprietaryData,
		}
	}
	if globalInfra != nil {
		cicdProvider := ""
		if len(globalInfra.CICDProviders) > 0 {
			cicdProvider = globalInfra.CICDProviders[0]
		}
		detection.Infrastructure = models.InfrastructureDetection{
			Grade:              &infraGrade,
			IaCDetected:        globalInfra.HasIaC,
			IaCTypes:           globalInfra.IaCTools,
			CICDDetected:       globalInfra.HasCICD,
			CICDProvider:       cicdProvider,
			MonitoringDetected: globalInfra.HasMonitoring,
			MonitoringTools:    globalInfra.MonitorTools,
		}
	}

	// Build top risks and strengths
	topRisks := buildTopRisks(redFlags, agg)
	topStrengths := buildTopStrengths(maintScore, secScore, handoffScore, infraScore, agg)

	// Build summary
	summary := models.Summary{
		ScoredCategories: scoredCategories,
		OverallGrade:     &overallGrade,
		TopRisks:         topRisks,
		TopStrengths:     topStrengths,
	}

	// Build the ScanResult
	now := time.Now().UTC()
	result := &models.ScanResult{
		Version:        "1.0",
		ScanID:         generateUUID(),
		Timestamp:      now.Format(time.RFC3339),
		ScannerVersion: version,
		Repositories:   allRepositories,
		TotalLOC:       totalLOC,
		TotalFileCount: totalFiles,
		RepoCount:      len(repos),
		TechStack:      techStackOut,
		Metrics: models.Metrics{
			Maintainability: &models.Maintainability{
				Grade:                &maintGrade,
				CyclomaticComplexity: models.ComplexityStats{
					Avg: agg.AvgComplexity,
					P90: globalP90,
					Max: agg.MaxComplexity,
				},
				NestingDepth: models.NestingStats{
					Avg: agg.AvgNesting,
					Max: agg.MaxNesting,
				},
				DuplicationPct:     agg.DuplicationPct,
				HotspotCount:       len(allHotspots),
				HotspotFiles:       allHotspots,
				PctFilesOver500LOC: agg.PctFilesOver500LOC,
			},
			Security: &models.Security{
				Grade:                &secGrade,
				SecretsFound:         agg.SecretsCount,
				SecretFindings:       allSecretFindings,
				CVEs:                 allCVEs,
				CVESummary:           cveSummary,
				LicenseIssues:        allLicenseIssues,
				LicenseIssueCount:    agg.LicenseIssueCount,
				CVEEcosystemsSkipped: ecosystemsSkipped,
			},
			DependencyHealth: depHealthMetric,
			HandoffReadiness: &models.HandoffReadiness{
				Grade:                &handoffGrade,
				EstTestCoveragePct:  agg.EstTestCoveragePct,
				DocDensity:          agg.DocDensity,
				EnvVarCount:         agg.EnvVarCount,
				HasReadme:           agg.HasReadme,
			},
		},
		Activity:    activityMetric,
		Detection:   detection,
		RedFlags:    redFlags,
		Summary:     summary,
		PricingTier: pricingTier,
		Warnings:    allWarnings,
	}

	// Ensure nil slices become empty arrays in JSON
	if result.Metrics.Security.CVEs == nil {
		result.Metrics.Security.CVEs = []models.CVE{}
	}
	if result.Metrics.Security.LicenseIssues == nil {
		result.Metrics.Security.LicenseIssues = []models.LicenseIssue{}
	}
	if result.Metrics.Security.CVEEcosystemsSkipped == nil {
		result.Metrics.Security.CVEEcosystemsSkipped = []string{}
	}
	if result.Metrics.Maintainability.HotspotFiles == nil {
		result.Metrics.Maintainability.HotspotFiles = []models.HotspotFile{}
	}
	if result.Summary.TopRisks == nil {
		result.Summary.TopRisks = []models.Risk{}
	}
	if result.Summary.TopStrengths == nil {
		result.Summary.TopStrengths = []models.Strength{}
	}
	if result.RedFlags.Flags == nil {
		result.RedFlags.Flags = []models.RedFlag{}
	}
	if result.TechStack.Frameworks == nil {
		result.TechStack.Frameworks = []string{}
	}
	if result.TechStack.Runtimes == nil {
		result.TechStack.Runtimes = []string{}
	}
	if result.TechStack.Databases == nil {
		result.TechStack.Databases = []string{}
	}
	if result.TechStack.ExternalServices == nil {
		result.TechStack.ExternalServices = []string{}
	}

	// Phase 4: Sign
	progress.SetPhase(output.PhaseOutputGeneration)

	if err := output.SignScanResult(result); err != nil {
		return fmt.Errorf("sign scan result: %w", err)
	}

	// Phase 5: Co-sign (unless offline)
	if !cfg.Offline {
		progress.SetPhase(output.PhaseCosigning)
		cosigner := output.NewCosignClient()
		cosignResult, err := cosigner.Cosign(ctx, result)
		if err != nil {
			return fmt.Errorf("co-sign: %w", err)
		}
		if cosignResult != nil && cosignResult.Warning != "" {
			allWarnings = append(allWarnings, models.Warning{
				Code:    "cosign_unavailable",
				Message: cosignResult.Warning,
			})
		}
	}

	// Always update warnings on the result (ensures consistency)
	result.Warnings = allWarnings
	if result.Warnings == nil {
		result.Warnings = []models.Warning{}
	}

	// Stop progress before output
	elapsed := progress.Elapsed()
	progress.Stop()

	// Write JSON output
	if cfg.Format != "terminal" {
		if err := output.WriteScanResult(result, cfg.Output); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	// Write terminal output
	if !cfg.Quiet && cfg.Format != "json" {
		colorEnabled := !cfg.NoColor
		formatter := &output.TerminalFormatter{
			Color:      &output.ColorConfig{Enabled: colorEnabled},
			OutputPath: cfg.Output,
			Duration:   elapsed,
		}
		formatter.Format(cmd.OutOrStdout(), result)
	}

	// Version update check (non-blocking, after output)
	if !cfg.NoUpdateCheck && !cfg.Offline {
		checker := updater.NewChecker(cfg.Home, version)
		if notice := checker.Check(ctx); notice != nil {
			fmt.Fprintln(cmd.ErrOrStderr(), updater.FormatNotice(notice, version))
		}
	}

	// CI quality gate check (after output so JSON is always written)
	if cfg.CI {
		return checkCIGate(cmd, result, cfg)
	}

	return nil
}

// CIGateError is returned when the CI quality gate fails.
// It implements error but signals to the CLI that output was already written
// and only the exit code should change.
type CIGateError struct {
	Reasons []string
}

func (e *CIGateError) Error() string {
	return fmt.Sprintf("CI quality gate failed: %s", strings.Join(e.Reasons, "; "))
}

// checkCIGate evaluates the scan result against CI thresholds.
func checkCIGate(cmd *cobra.Command, result *models.ScanResult, cfg *config.Config) error {
	var reasons []string

	// Check overall grade threshold
	threshold := models.Grade(cfg.CIThreshold)
	if result.Summary.OverallGrade != nil {
		if !scorer.GradeMeetsThreshold(*result.Summary.OverallGrade, threshold) {
			reasons = append(reasons, fmt.Sprintf("overall grade %s is below threshold %s",
				*result.Summary.OverallGrade, threshold))
		}
	} else {
		reasons = append(reasons, fmt.Sprintf("no overall grade computed, threshold is %s", threshold))
	}

	// Check red flags at or above the configured severity
	minSeverity := severityRank(models.Severity(cfg.CIFailOn))
	for _, flag := range result.RedFlags.Flags {
		if severityRank(flag.Severity) >= minSeverity {
			reasons = append(reasons, fmt.Sprintf("red flag [%s] %s: %s",
				flag.Severity, flag.Flag, flag.Detail))
		}
	}

	if len(reasons) > 0 {
		w := cmd.ErrOrStderr()
		fmt.Fprintln(w, "\nCI Quality Gate: FAILED")
		for _, r := range reasons {
			fmt.Fprintf(w, "  - %s\n", r)
		}
		return &CIGateError{Reasons: reasons}
	}

	fmt.Fprintln(cmd.ErrOrStderr(), "\nCI Quality Gate: PASSED")
	return nil
}

// severityRank returns a numeric rank for severity comparison (higher = more severe).
func severityRank(s models.Severity) int {
	switch s {
	case models.SeverityCritical:
		return 4
	case models.SeverityHigh:
		return 3
	case models.SeverityMedium:
		return 2
	case models.SeverityLow:
		return 1
	default:
		return 0
	}
}

// generateUUID generates a UUID v4 using crypto/rand.
func generateUUID() string {
	var uuid [16]byte
	if _, err := rand.Read(uuid[:]); err != nil {
		// Fallback: use timestamp-based pseudo-UUID (non-random but unique)
		now := time.Now().UnixNano()
		for i := 0; i < 16; i++ {
			uuid[i] = byte(now >> (i * 4))
		}
	}
	uuid[6] = (uuid[6] & 0x0f) | 0x40 // version 4
	uuid[8] = (uuid[8] & 0x3f) | 0x80 // variant 10
	return fmt.Sprintf("%08x-%04x-%04x-%04x-%012x",
		uuid[0:4], uuid[4:6], uuid[6:8], uuid[8:10], uuid[10:16])
}

// hashPath produces a SHA-256 hash of the path (no raw paths in JSON).
func hashPath(path string) string {
	h := sha256.Sum256([]byte(path))
	return hex.EncodeToString(h[:8]) // first 8 bytes = 16 hex chars
}

// computeP90FromSlice computes the 90th percentile from a pre-collected slice of values.
func computeP90FromSlice(values []int) int {
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

// monthBits converts MonthlyCommits array to a 12-bit bitmask (month has commits = bit set).
func monthBits(monthly [12]int) int {
	bits := 0
	for i, count := range monthly {
		if count > 0 {
			bits |= 1 << i
		}
	}
	return bits
}

// extractHotspots extracts files with max function complexity > 10.
func extractHotspots(results []*complexity.FileResult, repoName, repoPath string) []models.HotspotFile {
	var hotspots []models.HotspotFile
	for _, r := range results {
		maxC := 0
		for _, f := range r.Functions {
			if f.Complexity > maxC {
				maxC = f.Complexity
			}
		}
		if maxC > 10 {
			relPath, err := filepath.Rel(repoPath, r.Path)
			if err != nil {
				relPath = r.Path // fallback to absolute if Rel fails
			}
			hotspots = append(hotspots, models.HotspotFile{
				FileHash:   hashPath(r.Path),
				Complexity: maxC,
				LOC:        countFileLOC(r),
				Repo:       repoName,
				Path:       relPath,
			})
		}
	}
	return hotspots
}

// countFileLOC estimates LOC from function ranges.
func countFileLOC(r *complexity.FileResult) int {
	maxLine := 0
	for _, f := range r.Functions {
		if f.EndLine > maxLine {
			maxLine = f.EndLine
		}
	}
	return maxLine
}

// appendUnique appends items that aren't already present.
func appendUnique(slice []string, items ...string) []string {
	seen := make(map[string]bool, len(slice))
	for _, s := range slice {
		seen[s] = true
	}
	for _, item := range items {
		if !seen[item] {
			slice = append(slice, item)
			seen[item] = true
		}
	}
	return slice
}

// buildDepAges converts parsed dependencies to DepAge entries.
// Without online registry data, we can't know exact publish dates,
// so we return empty ages (AnalyzeHealth handles zero-length input).
func buildDepAges(dependencies []deps.Dependency) []deps.DepAge {
	// In offline mode / MVP, we don't have registry data for exact publish dates.
	// The health analyzer will return zero metrics for empty ages.
	// TODO: Add online registry lookup for npm, PyPI, Go, Maven, etc.
	return nil
}

// mergeTechStack merges tech stack results from multiple repos.
func mergeTechStack(existing *techstack.Result, next *techstack.Result) *techstack.Result {
	if existing == nil {
		return next
	}
	existing.Frameworks = appendUnique(existing.Frameworks, next.Frameworks...)
	existing.Databases = appendUnique(existing.Databases, next.Databases...)
	existing.Services = appendUnique(existing.Services, next.Services...)
	// Merge runtimes by name
	nameMap := make(map[string]techstack.RuntimeInfo)
	for _, r := range existing.Runtimes {
		nameMap[r.Name] = r
	}
	for _, r := range next.Runtimes {
		if _, ok := nameMap[r.Name]; !ok {
			nameMap[r.Name] = r
			existing.Runtimes = append(existing.Runtimes, r)
		}
	}
	return existing
}

// mergeAIDetect merges AI detection results from multiple repos.
func mergeAIDetect(existing *aidetect.Result, next *aidetect.Result) *aidetect.Result {
	if existing == nil {
		return next
	}
	existing.HasLLMAPI = existing.HasLLMAPI || next.HasLLMAPI
	existing.HasVectorDB = existing.HasVectorDB || next.HasVectorDB
	existing.HasRAGPipeline = existing.HasRAGPipeline || next.HasRAGPipeline
	existing.HasMCP = existing.HasMCP || next.HasMCP
	existing.HasFineTuning = existing.HasFineTuning || next.HasFineTuning
	existing.HasTrainingPipeline = existing.HasTrainingPipeline || next.HasTrainingPipeline
	existing.HasProprietaryData = existing.HasProprietaryData || next.HasProprietaryData
	existing.LLMProviders = appendUnique(existing.LLMProviders, next.LLMProviders...)
	existing.VectorDBProviders = appendUnique(existing.VectorDBProviders, next.VectorDBProviders...)
	return existing
}

// mergeInfra merges infrastructure results from multiple repos.
func mergeInfra(existing *infra.Result, next *infra.Result) *infra.Result {
	if existing == nil {
		return next
	}
	existing.HasIaC = existing.HasIaC || next.HasIaC
	existing.HasCICD = existing.HasCICD || next.HasCICD
	existing.HasMonitoring = existing.HasMonitoring || next.HasMonitoring
	existing.IaCTools = appendUnique(existing.IaCTools, next.IaCTools...)
	existing.CICDProviders = appendUnique(existing.CICDProviders, next.CICDProviders...)
	existing.MonitorTools = appendUnique(existing.MonitorTools, next.MonitorTools...)
	return existing
}

// mergeActivity selects the most recent activity result.
func mergeActivity(existing *activity.Result, next *activity.Result) *activity.Result {
	if existing == nil || !existing.HasGit {
		return next
	}
	if !next.HasGit {
		return existing
	}
	// Keep the one with the most recent commit
	if next.DaysSinceLastCommit < existing.DaysSinceLastCommit {
		return next
	}
	return existing
}

// buildTopRisks extracts the most important risks from red flags and metrics.
func buildTopRisks(flags models.RedFlags, agg scorer.AggregatedMetrics) []models.Risk {
	var risks []models.Risk
	for _, f := range flags.Flags {
		category := "general"
		switch f.Flag {
		case models.RedFlagSecretsDetected, models.RedFlagCriticalCVE:
			category = "security"
		case models.RedFlagNoTests, models.RedFlagNoReadme:
			category = "handoff_readiness"
		case models.RedFlagStaleRepo, models.RedFlagNoGitHistory:
			category = "development_activity"
		case models.RedFlagUnmaintainedDeps:
			category = "dependency_health"
		case models.RedFlagNoCICD:
			category = "sre_infrastructure"
		}
		risks = append(risks, models.Risk{
			Category: category,
			Issue:    f.Detail,
			Severity: f.Severity,
		})
	}
	// Limit to top 5
	if len(risks) > 5 {
		risks = risks[:5]
	}
	return risks
}

// buildTopStrengths identifies top strengths from scores.
func buildTopStrengths(maintScore, secScore, handoffScore, infraScore float64, agg scorer.AggregatedMetrics) []models.Strength {
	var strengths []models.Strength

	if secScore >= 90 && agg.SecretsCount == 0 {
		strengths = append(strengths, models.Strength{
			Category: "security",
			Detail:   "No secrets or critical vulnerabilities detected",
		})
	}
	if maintScore >= 80 {
		strengths = append(strengths, models.Strength{
			Category: "maintainability",
			Detail:   fmt.Sprintf("Low complexity (avg %.1f) and %.1f%% duplication", agg.AvgComplexity, agg.DuplicationPct),
		})
	}
	if handoffScore >= 80 {
		strengths = append(strengths, models.Strength{
			Category: "handoff_readiness",
			Detail:   fmt.Sprintf("%.0f%% est. test coverage with %s documentation", agg.EstTestCoveragePct, agg.DocDensity),
		})
	}
	if infraScore >= 80 {
		strengths = append(strengths, models.Strength{
			Category: "sre_infrastructure",
			Detail:   "CI/CD, IaC, and monitoring detected",
		})
	}
	if agg.HasGitHistory && agg.DaysSinceLastCommit <= 30 {
		strengths = append(strengths, models.Strength{
			Category: "development_activity",
			Detail:   "Active development with recent commits",
		})
	}

	// Limit to top 3
	if len(strengths) > 3 {
		strengths = strengths[:3]
	}
	return strengths
}

// repoData type is defined locally in runScan; redeclare here for helper access.
type repoData struct {
	input      config.RepoInput
	walk       *walker.WalkResult
	langDet    *language.DetectionResult
	depsParsed *deps.ParseResult
}
