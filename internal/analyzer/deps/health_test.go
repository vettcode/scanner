package deps

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAnalyzeHealth_BasicMetrics(t *testing.T) {
	now := time.Now()
	ages := []DepAge{
		{Name: "fresh-pkg", LastPublished: now.AddDate(0, -1, 0), AgeDays: 30},
		{Name: "recent-pkg", LastPublished: now.AddDate(0, -6, 0), AgeDays: 180},
		{Name: "old-pkg", LastPublished: now.AddDate(-1, 0, 0), AgeDays: 365},
		{Name: "ancient-pkg", LastPublished: now.AddDate(-3, 0, 0), AgeDays: 1095},
	}

	r := AnalyzeHealth(ages)
	assert.Equal(t, 4, r.TotalDeps)
	assert.Equal(t, 272, r.MedianAgeDays) // median of sorted [30, 180, 365, 1095] = (180+365)/2 = 272
	assert.Equal(t, 1, r.UnmaintainedCount)
	assert.InDelta(t, 25.0, r.UnmaintainedPct, 0.01)
	assert.Equal(t, "ancient-pkg", r.OldestDep.Name)
	assert.Equal(t, 1095, r.OldestDep.AgeDays)
}

func TestAnalyzeHealth_AllFresh(t *testing.T) {
	ages := []DepAge{
		{Name: "a", AgeDays: 10},
		{Name: "b", AgeDays: 20},
		{Name: "c", AgeDays: 30},
	}
	r := AnalyzeHealth(ages)
	assert.Equal(t, 0, r.UnmaintainedCount)
	assert.Equal(t, 0.0, r.UnmaintainedPct)
	assert.Equal(t, 20, r.MedianAgeDays) // middle of [10, 20, 30]
}

func TestAnalyzeHealth_AllUnmaintained(t *testing.T) {
	ages := []DepAge{
		{Name: "a", AgeDays: 800},
		{Name: "b", AgeDays: 1000},
	}
	r := AnalyzeHealth(ages)
	assert.Equal(t, 2, r.UnmaintainedCount)
	assert.InDelta(t, 100.0, r.UnmaintainedPct, 0.01)
}

func TestAnalyzeHealth_Empty(t *testing.T) {
	r := AnalyzeHealth(nil)
	assert.Equal(t, 0, r.TotalDeps)
	assert.Nil(t, r.OldestDep)
}

func TestAnalyzeHealth_SingleDep(t *testing.T) {
	ages := []DepAge{{Name: "solo", AgeDays: 100}}
	r := AnalyzeHealth(ages)
	assert.Equal(t, 100, r.MedianAgeDays)
	assert.Equal(t, "solo", r.OldestDep.Name)
}
