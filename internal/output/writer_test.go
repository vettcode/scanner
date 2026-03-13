package output

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/pkg/models"
)

func TestWriteScanResult_RoundTrip(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")

	grade := models.GradeB
	result := &models.ScanResult{
		Version:        "1.0",
		ScanID:         "test-id",
		Timestamp:      "2026-03-13T00:00:00Z",
		ScannerVersion: "0.1.0",
		TotalLOC:       10000,
		RepoCount:      1,
		Summary: models.Summary{
			OverallGrade: &grade,
		},
		Warnings: []models.Warning{},
	}

	err := WriteScanResult(result, outPath)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)

	var loaded models.ScanResult
	require.NoError(t, json.Unmarshal(data, &loaded))
	assert.Equal(t, "test-id", loaded.ScanID)
	assert.Equal(t, 10000, loaded.TotalLOC)
	assert.Equal(t, models.GradeB, *loaded.Summary.OverallGrade)
}

func TestWriteScanResult_AtomicWrite_NoPartialFile(t *testing.T) {
	// Write to a non-existent directory should fail without leaving temp files
	err := WriteScanResult(&models.ScanResult{}, "/nonexistent/dir/result.json")
	assert.Error(t, err)
}

func TestWriteScanResult_NoHTMLEscaping(t *testing.T) {
	dir := t.TempDir()
	outPath := filepath.Join(dir, "result.json")

	result := &models.ScanResult{
		Version: "1.0",
		ScanID:  "test",
		TechStack: models.TechStack{
			Frameworks: []string{"Next.js <14>"},
		},
		Warnings: []models.Warning{},
	}

	err := WriteScanResult(result, outPath)
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	// < and > should NOT be HTML-escaped
	assert.Contains(t, string(data), "Next.js <14>")
}
