package infra

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/walker"
)

func TestAnalyze_Dockerfile(t *testing.T) {
	dir := t.TempDir()
	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "Dockerfile"), RelPath: "Dockerfile"},
	}
	r := Analyze(dir, files, nil)
	assert.True(t, r.HasIaC)
	assert.Contains(t, r.IaCTools, "Docker")
}

func TestAnalyze_Terraform(t *testing.T) {
	dir := t.TempDir()
	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "main.tf"), RelPath: "main.tf"},
	}
	r := Analyze(dir, files, nil)
	assert.True(t, r.HasIaC)
	assert.Contains(t, r.IaCTools, "Terraform")
}

func TestAnalyze_GitHubActions(t *testing.T) {
	dir := t.TempDir()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0755))
	os.WriteFile(filepath.Join(workflowDir, "ci.yml"), []byte("name: CI"), 0644)

	files := []walker.FileInfo{
		{Path: filepath.Join(workflowDir, "ci.yml"), RelPath: ".github/workflows/ci.yml"},
	}
	r := Analyze(dir, files, nil)
	assert.True(t, r.HasCICD)
	assert.Contains(t, r.CICDProviders, "GitHub Actions")
}

func TestAnalyze_GitHubActionsFromRoot(t *testing.T) {
	dir := t.TempDir()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	require.NoError(t, os.MkdirAll(workflowDir, 0755))

	// Even with no files passed, root check finds it
	r := Analyze(dir, nil, nil)
	assert.True(t, r.HasCICD)
	assert.Contains(t, r.CICDProviders, "GitHub Actions")
}

func TestAnalyze_MonitoringFromDeps(t *testing.T) {
	dir := t.TempDir()
	deps := []string{"@sentry/node", "prom-client", "unknown-package"}
	r := Analyze(dir, nil, deps)
	assert.True(t, r.HasMonitoring)
	assert.Contains(t, r.MonitorTools, "Sentry")
	assert.Contains(t, r.MonitorTools, "Prometheus")
}

func TestAnalyze_MonitoringFromConfig(t *testing.T) {
	dir := t.TempDir()
	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "prometheus.yml"), RelPath: "prometheus.yml"},
	}
	r := Analyze(dir, files, nil)
	assert.True(t, r.HasMonitoring)
	assert.Contains(t, r.MonitorTools, "Prometheus")
}

func TestAnalyze_NothingDetected(t *testing.T) {
	dir := t.TempDir()
	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "main.go"), RelPath: "main.go"},
	}
	r := Analyze(dir, files, nil)
	assert.False(t, r.HasIaC)
	assert.False(t, r.HasCICD)
	assert.False(t, r.HasMonitoring)
	assert.Empty(t, r.IaCTools)
	assert.Empty(t, r.CICDProviders)
	assert.Empty(t, r.MonitorTools)
}

func TestAnalyze_K8sManifests(t *testing.T) {
	dir := t.TempDir()
	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "deployment.yaml"), RelPath: "deployment.yaml"},
	}
	r := Analyze(dir, files, nil)
	assert.True(t, r.HasIaC)
	assert.Contains(t, r.IaCTools, "Kubernetes")
}

func TestAnalyze_FullStack(t *testing.T) {
	dir := t.TempDir()
	workflowDir := filepath.Join(dir, ".github", "workflows")
	os.MkdirAll(workflowDir, 0755)

	files := []walker.FileInfo{
		{Path: filepath.Join(dir, "Dockerfile"), RelPath: "Dockerfile"},
		{Path: filepath.Join(dir, "main.tf"), RelPath: "main.tf"},
		{Path: filepath.Join(workflowDir, "ci.yml"), RelPath: ".github/workflows/ci.yml"},
	}
	deps := []string{"@sentry/node"}

	r := Analyze(dir, files, deps)
	assert.True(t, r.HasIaC)
	assert.True(t, r.HasCICD)
	assert.True(t, r.HasMonitoring)
}
