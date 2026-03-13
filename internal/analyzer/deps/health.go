package deps

import (
	"sort"
	"time"
)

// HealthResult holds the dependency health analysis results.
type HealthResult struct {
	TotalDeps        int
	MedianAgeDays    int
	MedianAgeMonths  float64
	UnmaintainedPct  float64   // % of deps with no update in 2+ years
	UnmaintainedCount int
	OldestDep        *OldestDep
	OutdatedCount    int       // deps behind latest version (online only)
}

// OldestDep holds info about the oldest dependency.
type OldestDep struct {
	Name     string
	AgeDays  int
}

// DepAge holds the age of a dependency (from registry lookup).
type DepAge struct {
	Name          string
	LastPublished time.Time
	AgeDays       int
}

// AnalyzeHealth computes dependency health metrics from age data.
// Ages must be provided by the caller (from registry lookups or cache).
func AnalyzeHealth(ages []DepAge) *HealthResult {
	r := &HealthResult{
		TotalDeps: len(ages),
	}

	if len(ages) == 0 {
		return r
	}

	// Sort by age for median calculation
	sort.Slice(ages, func(i, j int) bool {
		return ages[i].AgeDays < ages[j].AgeDays
	})

	// Median age
	mid := len(ages) / 2
	if len(ages)%2 == 0 {
		r.MedianAgeDays = (ages[mid-1].AgeDays + ages[mid].AgeDays) / 2
	} else {
		r.MedianAgeDays = ages[mid].AgeDays
	}
	r.MedianAgeMonths = float64(r.MedianAgeDays) / 30.44 // avg days per month

	// Unmaintained: no update in 2+ years (730 days)
	twoYears := 730
	for _, a := range ages {
		if a.AgeDays >= twoYears {
			r.UnmaintainedCount++
		}
	}
	r.UnmaintainedPct = float64(r.UnmaintainedCount) / float64(len(ages)) * 100.0

	// Oldest dependency
	oldest := ages[len(ages)-1]
	r.OldestDep = &OldestDep{
		Name:    oldest.Name,
		AgeDays: oldest.AgeDays,
	}

	return r
}
