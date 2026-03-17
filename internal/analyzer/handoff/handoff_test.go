package handoff

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/internal/language"
	"github.com/vettcode/scanner/internal/walker"
)

func TestComputeTestCoverage(t *testing.T) {
	wr := &walker.WalkResult{
		Files: []walker.FileInfo{
			{Language: "Go", Tier: language.Tier1, LOC: 2000, IsTest: false},
			{Language: "Go", Tier: language.Tier1, LOC: 250, IsTest: true},
		},
	}
	pct := computeTestCoverage(wr)
	assert.InDelta(t, 50.0, pct, 0.01) // 250/2000 * 4 * 100 = 50%
}

func TestComputeTestCoverage_NoTests(t *testing.T) {
	wr := &walker.WalkResult{
		Files: []walker.FileInfo{
			{Language: "Go", Tier: language.Tier1, LOC: 1000, IsTest: false},
		},
	}
	assert.Equal(t, 0.0, computeTestCoverage(wr))
}

func TestComputeTestCoverage_Empty(t *testing.T) {
	assert.Equal(t, 0.0, computeTestCoverage(nil))
	assert.Equal(t, 0.0, computeTestCoverage(&walker.WalkResult{}))
}

func TestComputeDocDensity_High(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Hi"), 0644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)

	wr := &walker.WalkResult{
		Files: []walker.FileInfo{
			{Language: "Markdown", LOC: 10},
			{Language: "Markdown", LOC: 10},
			{Language: "Markdown", LOC: 10},
			{Language: "Markdown", LOC: 10},
			{Language: "Markdown", LOC: 10},
		},
	}
	assert.Equal(t, "high", computeDocDensity(dir, wr))
}

func TestComputeDocDensity_Low(t *testing.T) {
	dir := t.TempDir()
	assert.Equal(t, "low", computeDocDensity(dir, nil))
}

func TestCountEnvVars(t *testing.T) {
	dir := t.TempDir()
	envContent := `# Database
DB_HOST=localhost
DB_PORT=5432
DB_NAME=mydb

# API
API_KEY=secret
API_URL=https://api.example.com
`
	os.WriteFile(filepath.Join(dir, ".env.example"), []byte(envContent), 0644)
	assert.Equal(t, 5, countEnvVars(dir))
}

func TestCountEnvVars_NoFile(t *testing.T) {
	assert.Equal(t, 0, countEnvVars(t.TempDir()))
}

func TestAnalyze_FullProject(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "README.md"), []byte("# Project\n"), 0644)
	os.WriteFile(filepath.Join(dir, ".env.example"), []byte("KEY=val\nSECRET=s\n"), 0644)
	os.WriteFile(filepath.Join(dir, "jest.config.js"), []byte("module.exports = {}"), 0644)
	os.WriteFile(filepath.Join(dir, ".nycrc"), []byte("{}"), 0644)
	os.MkdirAll(filepath.Join(dir, "docs"), 0755)

	wr := &walker.WalkResult{
		Files: []walker.FileInfo{
			{Language: "TypeScript", Tier: language.Tier1, LOC: 1000, IsTest: false},
			{Language: "TypeScript", Tier: language.Tier1, LOC: 100, IsTest: true},
			{Language: "Markdown", LOC: 10},
		},
	}

	r := Analyze(dir, wr)
	assert.True(t, r.HasReadme)
	assert.True(t, r.HasEnvTemplate)
	assert.True(t, r.HasTestConfig)
	assert.True(t, r.HasCoverageConfig)
	assert.Equal(t, 2, r.EnvVarCount)
	assert.InDelta(t, 40.0, r.EstTestCoveragePct, 0.5) // 100/1000 * 4 * 100 = 40% — Tier 1 only, Markdown excluded
}

func TestAnalyze_BareProject(t *testing.T) {
	dir := t.TempDir()
	wr := &walker.WalkResult{
		Files: []walker.FileInfo{
			{Language: "Go", Tier: language.Tier1, LOC: 100, IsTest: false},
		},
	}
	r := Analyze(dir, wr)
	assert.False(t, r.HasReadme)
	assert.False(t, r.HasEnvTemplate)
	assert.False(t, r.HasTestConfig)
	assert.False(t, r.HasCoverageConfig)
	assert.Equal(t, 0, r.EnvVarCount)
	assert.Equal(t, "low", r.DocDensity)
}
