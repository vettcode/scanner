package filesize

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/internal/language"
	"github.com/vettcode/scanner/internal/walker"
)

func TestAnalyze_BasicDistribution(t *testing.T) {
	files := []walker.FileInfo{
		{Path: "a.go", LOC: 50, Tier: language.Tier1},
		{Path: "b.go", LOC: 200, Tier: language.Tier1},
		{Path: "c.go", LOC: 450, Tier: language.Tier1},
		{Path: "d.go", LOC: 800, Tier: language.Tier1},
		{Path: "e.go", LOC: 1500, Tier: language.Tier1},
	}

	d := Analyze(files)
	assert.Equal(t, 5, d.TotalFiles)
	assert.Equal(t, 2, d.FilesOver500LOC) // 800 + 1500
	assert.InDelta(t, 40.0, d.PctOver500LOC, 0.01)
	assert.Equal(t, 1500, d.LargestFile)

	assert.Equal(t, 1, d.Buckets["0-100"])
	assert.Equal(t, 1, d.Buckets["101-300"])
	assert.Equal(t, 1, d.Buckets["301-500"])
	assert.Equal(t, 1, d.Buckets["501-1000"])
	assert.Equal(t, 1, d.Buckets["1001+"])
}

func TestAnalyze_ExcludesTestFiles(t *testing.T) {
	files := []walker.FileInfo{
		{Path: "main.go", LOC: 600, Tier: language.Tier1},
		{Path: "main_test.go", LOC: 900, Tier: language.Tier1, IsTest: true},
	}

	d := Analyze(files)
	assert.Equal(t, 1, d.TotalFiles)
	assert.Equal(t, 1, d.FilesOver500LOC)
	assert.InDelta(t, 100.0, d.PctOver500LOC, 0.01)
}

func TestAnalyze_Empty(t *testing.T) {
	d := Analyze(nil)
	assert.Equal(t, 0, d.TotalFiles)
	assert.Equal(t, 0.0, d.PctOver500LOC)
	assert.Equal(t, 0, d.LargestFile)
}

func TestAnalyze_AllSmallFiles(t *testing.T) {
	files := []walker.FileInfo{
		{Path: "a.go", LOC: 10},
		{Path: "b.go", LOC: 20},
		{Path: "c.go", LOC: 30},
	}
	d := Analyze(files)
	assert.Equal(t, 0, d.FilesOver500LOC)
	assert.Equal(t, 0.0, d.PctOver500LOC)
	assert.Equal(t, 3, d.Buckets["0-100"])
}
