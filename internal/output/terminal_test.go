package output

import (
	"bytes"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/pkg/models"
)

func fullTestResult() *models.ScanResult {
	gradeB := models.GradeB
	gradeBM := models.GradeBM
	gradeA := models.GradeA
	gradeC := models.GradeC
	gradeCP := models.GradeCP
	gradeBP := models.GradeBP

	return &models.ScanResult{
		Version:        "1.0",
		ScanID:         "test-id",
		Timestamp:      "2026-03-13",
		ScannerVersion: "0.1.0",
		Repositories: []models.Repository{
			{
				Name:     "backend",
				LOC:      30000,
				Languages: map[string]float64{"Go": 80, "Python": 20},
			},
			{
				Name:     "frontend",
				LOC:      10000,
				Languages: map[string]float64{"TypeScript": 100},
			},
		},
		TotalLOC:       40000,
		TotalFileCount: 500,
		RepoCount:      2,
		TechStack: models.TechStack{
			Frameworks: []string{"Gin", "Next.js"},
			Runtimes:   []string{"Go 1.23", "Node 20"},
			Databases:  []string{"PostgreSQL"},
		},
		Metrics: models.Metrics{
			Maintainability: &models.Maintainability{
				Grade: &gradeB,
				CyclomaticComplexity: models.ComplexityStats{Avg: 7.2, P90: 15, Max: 35},
				DuplicationPct: 4.5,
				HotspotCount:   3,
				HotspotFiles: []models.HotspotFile{
					{FileHash: "abc123", Complexity: 35, LOC: 450, Repo: "backend"},
					{FileHash: "def456", Complexity: 28, LOC: 320, Repo: "backend"},
				},
			},
			Security: &models.Security{
				Grade:        &gradeBM,
				SecretsFound: 0,
				CVESummary:   models.CVESummary{Critical: 0, High: 1, Medium: 3, Low: 2},
				OutdatedDeps: models.OutdatedDeps{Total: 45, Outdated: 8},
				LicenseIssueCount: 1,
			},
			DependencyHealth: &models.DependencyHealth{
				Grade:           &gradeC,
				MedianAgeMonths: 14,
				UnmaintainedPct: 12,
				UnmaintainedCount: 3,
				Oldest: &models.OldestDep{Package: "lodash", AgeYears: 4.2, Repo: "frontend"},
			},
			HandoffReadiness: &models.HandoffReadiness{
				Grade:              &gradeBP,
				EstTestCoveragePct: 62,
				DocDensity:         models.DocDensityMedium,
				EnvVarCount:        8,
			},
		},
		Activity: &models.Activity{
			Grade:               &gradeA,
			LastCommitDate:      "2026-03-10",
			DaysSinceLastCommit: 3,
			CommitVelocity: models.CommitVelocity{
				AvgPerMonth: 38,
				Trend:       models.TrendStable,
			},
			ActiveMonths: 11,
		},
		Detection: models.Detection{
			AI: models.AIDetection{
				LLMAPI:      true,
				LLMProvider: "OpenAI",
				RAGPipeline: true,
			},
			Infrastructure: models.InfrastructureDetection{
				Grade:              &gradeCP,
				IaCDetected:        true,
				IaCTypes:           []string{"Terraform", "Docker"},
				CICDDetected:       true,
				CICDProvider:       "GitHub Actions",
				MonitoringDetected: false,
			},
		},
		Summary: models.Summary{
			OverallGrade: &gradeB,
		},
		Warnings: []models.Warning{},
	}
}

func TestTerminalFormatter_Format_ContainsAllSections(t *testing.T) {
	result := fullTestResult()
	formatter := &TerminalFormatter{
		Color:      &ColorConfig{Enabled: false},
		OutputPath: "./vettcode-scan-result.json",
		Duration:   2*time.Minute + 15*time.Second,
	}

	var buf bytes.Buffer
	formatter.Format(&buf, result)
	out := buf.String()

	// Header
	assert.Contains(t, out, "VettCode Scan Complete")
	assert.Contains(t, out, "2026-03-13")
	assert.Contains(t, out, "2 scanned")
	assert.Contains(t, out, "40,000")

	// Tech stack
	assert.Contains(t, out, "Gin")
	assert.Contains(t, out, "Next.js")
	assert.Contains(t, out, "Go 1.23")
	assert.Contains(t, out, "PostgreSQL")

	// Duration
	assert.Contains(t, out, "2m15s")

	// Maintainability
	assert.Contains(t, out, "MAINTAINABILITY")
	assert.Contains(t, out, "7.2")
	assert.Contains(t, out, "4.5%")

	// Security
	assert.Contains(t, out, "SECURITY")
	assert.Contains(t, out, "8/45")

	// Dependency health
	assert.Contains(t, out, "DEPENDENCY HEALTH")
	assert.Contains(t, out, "14 months")
	assert.Contains(t, out, "lodash")

	// Activity
	assert.Contains(t, out, "DEVELOPMENT ACTIVITY")
	assert.Contains(t, out, "3 days ago")
	assert.Contains(t, out, "38/mo avg")
	assert.Contains(t, out, "Stable")

	// AI detection
	assert.Contains(t, out, "AI DETECTION")
	assert.Contains(t, out, "OpenAI")
	assert.Contains(t, out, "RAG Pipeline")

	// Infrastructure
	assert.Contains(t, out, "INFRASTRUCTURE")
	assert.Contains(t, out, "Terraform")
	assert.Contains(t, out, "GitHub Actions")

	// Handoff
	assert.Contains(t, out, "HANDOFF READINESS")
	assert.Contains(t, out, "62%")
	assert.Contains(t, out, "Medium")

	// Overall grade
	assert.Contains(t, out, "OVERALL GRADE")

	// Footer
	assert.Contains(t, out, "vettcode-scan-result.json")
	assert.Contains(t, out, "platform.vettcode.com")
	assert.Contains(t, out, "vettcode.com/deep")
}

func TestTerminalFormatter_NoColor(t *testing.T) {
	result := fullTestResult()
	formatter := &TerminalFormatter{
		Color: &ColorConfig{Enabled: false},
	}

	var buf bytes.Buffer
	formatter.Format(&buf, result)
	out := buf.String()

	// No ANSI escape codes
	assert.NotContains(t, out, "\033[")
}

func TestTerminalFormatter_WithColor(t *testing.T) {
	result := fullTestResult()
	formatter := &TerminalFormatter{
		Color: &ColorConfig{Enabled: true},
	}

	var buf bytes.Buffer
	formatter.Format(&buf, result)
	out := buf.String()

	// Should contain ANSI escape codes
	assert.Contains(t, out, "\033[")
}

func TestTerminalFormatter_NilSections(t *testing.T) {
	result := &models.ScanResult{
		Timestamp: "2026-03-13",
		RepoCount: 1,
		TotalLOC:  1000,
		Metrics:   models.Metrics{},    // all nil
		Summary:   models.Summary{},
		Warnings:  []models.Warning{},
	}

	formatter := &TerminalFormatter{
		Color: &ColorConfig{Enabled: false},
	}

	var buf bytes.Buffer
	formatter.Format(&buf, result)
	out := buf.String()

	// Should show N/A for nil sections
	assert.Contains(t, out, "N/A")
}

func TestFormatNumber(t *testing.T) {
	assert.Equal(t, "0", formatNumber(0))
	assert.Equal(t, "999", formatNumber(999))
	assert.Equal(t, "1,000", formatNumber(1000))
	assert.Equal(t, "42,600", formatNumber(42600))
	assert.Equal(t, "1,000,000", formatNumber(1000000))
}

func TestFormatDuration(t *testing.T) {
	assert.Equal(t, "500ms", formatDuration(500*time.Millisecond))
	assert.Equal(t, "3.5s", formatDuration(3500*time.Millisecond))
	assert.Equal(t, "2m15s", formatDuration(2*time.Minute+15*time.Second))
}

func TestAggregateLanguages(t *testing.T) {
	repos := []models.Repository{
		{LOC: 3000, Languages: map[string]float64{"Go": 100}},
		{LOC: 1000, Languages: map[string]float64{"Python": 100}},
	}
	result := aggregateLanguages(repos)
	assert.Contains(t, result, "Go (75%)")
	assert.Contains(t, result, "Python (25%)")
}

func TestAggregateLanguages_Empty(t *testing.T) {
	result := aggregateLanguages(nil)
	assert.Equal(t, "", result)
}

func TestColorConfig_GradeColor(t *testing.T) {
	c := &ColorConfig{Enabled: true}
	// A grades should be bold green
	assert.Contains(t, c.gradeColor("A"), ansiGreen)
	assert.Contains(t, c.gradeColor("A"), ansiBold)
	// F should be bold red
	assert.Contains(t, c.gradeColor("F"), ansiRed)
	// Disabled = no escapes
	noColor := &ColorConfig{Enabled: false}
	assert.Equal(t, "A", noColor.gradeColor("A"))
}

func TestColorConfig_YesNo(t *testing.T) {
	c := &ColorConfig{Enabled: false}
	assert.Equal(t, "Yes (OpenAI)", c.yesNo(true, "OpenAI"))
	assert.Equal(t, "Yes", c.yesNo(true, ""))
	assert.Equal(t, "No", c.yesNo(false, ""))
}
