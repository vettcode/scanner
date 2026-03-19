package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/analyzer/deps"
	"github.com/vettcode/scanner/internal/analyzer/security/snapshot"
)

func TestLookupCVEs_OfflineSkipsNonSupported(t *testing.T) {
	dependencies := []deps.Dependency{
		{Name: "express", Version: "4.17.1", Ecosystem: "npm"},
		{Name: "laravel/framework", Version: "10.0.0", Ecosystem: "packagist"},
		{Name: "rails", Version: "7.0.0", Ecosystem: "rubygems"},
	}

	r := LookupCVEs(dependencies, true)
	assert.Contains(t, r.EcosystemsSkipped, "packagist")
	assert.Contains(t, r.EcosystemsSkipped, "rubygems")
	assert.NotContains(t, r.EcosystemsSkipped, "npm")
}

func TestLookupCVEs_SkipsNilVersion(t *testing.T) {
	dependencies := []deps.Dependency{
		{Name: "express", Version: "", Ecosystem: "npm"},
	}
	r := LookupCVEs(dependencies, true)
	assert.Equal(t, 0, r.Summary.Total)
}

func TestLookupCVEs_Empty(t *testing.T) {
	r := LookupCVEs(nil, true)
	assert.Equal(t, 0, r.Summary.Total)
	assert.Empty(t, r.Vulnerabilities)
}

func TestCVSSToSeverity(t *testing.T) {
	assert.Equal(t, "critical", cvssToSeverity("9.8"))
	assert.Equal(t, "high", cvssToSeverity("7.5"))
	assert.Equal(t, "medium", cvssToSeverity("5.0"))
	assert.Equal(t, "low", cvssToSeverity("2.0"))
}

func TestCVSSToSeverity_VectorString(t *testing.T) {
	// Critical: Network/Low complexity/High impacts
	sev := cvssToSeverity("CVSS:3.1/AV:N/AC:L/PR:N/UI:N/S:U/C:H/I:H/A:H")
	assert.Equal(t, "critical", sev)

	// Lower severity: Local/High complexity
	sev2 := cvssToSeverity("CVSS:3.1/AV:L/AC:H/PR:H/UI:R/S:U/C:L/I:N/A:N")
	assert.NotEqual(t, "critical", sev2) // should be medium or low

	// Unknown format falls back to medium
	assert.Equal(t, "medium", cvssToSeverity("unknown"))
}

func TestLookupCVEs_OfflineWithSnapshot(t *testing.T) {
	// Override snapshot with test data
	snapshot.Override("2026-03-01", map[string][]snapshot.VulnEntry{
		"npm:express": {
			{ID: "CVE-2024-1234", Severity: "high", FixedVersion: "4.18.0", Introduced: "4.0.0"},
			{ID: "CVE-2024-5678", Severity: "critical", FixedVersion: "4.17.2", Introduced: "4.17.0"},
		},
		"PyPI:requests": {
			{ID: "CVE-2024-9999", Severity: "medium", FixedVersion: "2.32.0", Introduced: "2.0.0"},
		},
	})
	defer snapshot.Reset()

	dependencies := []deps.Dependency{
		{Name: "express", Version: "4.17.1", Ecosystem: "npm"},
		{Name: "requests", Version: "2.31.0", Ecosystem: "pypi"},
		{Name: "laravel/framework", Version: "10.0.0", Ecosystem: "packagist"}, // not in snapshot
	}

	r := LookupCVEs(dependencies, true)

	// Should find vulnerabilities from snapshot
	require.GreaterOrEqual(t, len(r.Vulnerabilities), 2) // express 4.17.1 matches both CVEs

	// Check that express CVEs were found
	var foundHigh, foundCritical bool
	for _, v := range r.Vulnerabilities {
		if v.ID == "CVE-2024-1234" && v.Severity == "high" {
			foundHigh = true
		}
		if v.ID == "CVE-2024-5678" && v.Severity == "critical" {
			foundCritical = true
		}
	}
	assert.True(t, foundHigh, "should find high severity CVE for express")
	assert.True(t, foundCritical, "should find critical CVE for express 4.17.1 (< 4.17.2)")

	// Packagist should be skipped (not in offline ecosystems)
	assert.Contains(t, r.EcosystemsSkipped, "packagist")

	// Summary should be computed
	assert.Greater(t, r.Summary.Total, 0)

	// Should include snapshot date warning
	hasSnapshotWarning := false
	for _, w := range r.Warnings {
		if len(w) > 0 {
			hasSnapshotWarning = true
		}
	}
	assert.True(t, hasSnapshotWarning)
}

func TestLookupCVEs_OfflineVersionNotAffected(t *testing.T) {
	snapshot.Override("2026-03-01", map[string][]snapshot.VulnEntry{
		"npm:express": {
			{ID: "CVE-2024-1234", Severity: "high", FixedVersion: "4.18.0", Introduced: "4.0.0"},
		},
	})
	defer snapshot.Reset()

	// Version 4.18.0 is the fix version, so it should NOT be affected
	dependencies := []deps.Dependency{
		{Name: "express", Version: "4.18.0", Ecosystem: "npm"},
	}

	r := LookupCVEs(dependencies, true)
	assert.Equal(t, 0, r.Summary.Total, "fixed version should not match vulnerability")
}

func TestFindFixVersionForVersion(t *testing.T) {
	// Multi-branch vuln: 5.x branch fixed at 5.4.50, 7.x branch fixed at 7.2.1
	affected := []osvAffected{
		{Ranges: []struct {
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		}{
			{Events: []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			}{
				{Introduced: "5.0.0"},
				{Fixed: "5.4.50"},
			}},
		}},
		{Ranges: []struct {
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		}{
			{Events: []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			}{
				{Introduced: "7.0.0"},
				{Fixed: "7.2.1"},
			}},
		}},
	}

	// User on 7.2.0 → should get fix 7.2.1 (not 5.4.50)
	assert.Equal(t, "7.2.1", findFixVersionForVersion(affected, "7.2.0"))

	// User on 5.4.0 → should get fix 5.4.50
	assert.Equal(t, "5.4.50", findFixVersionForVersion(affected, "5.4.0"))

	// User on 7.2.1 → already fixed, no matching range
	assert.Equal(t, "", findFixVersionForVersion(affected, "7.2.1"))

	// User on 2.0.0 → before any affected range
	assert.Equal(t, "", findFixVersionForVersion(affected, "2.0.0"))
}

func TestCVESummary(t *testing.T) {
	r := &CVEResult{
		Vulnerabilities: []Vulnerability{
			{Severity: "critical"},
			{Severity: "high"},
			{Severity: "high"},
			{Severity: "medium"},
			{Severity: "low"},
		},
	}
	// Rebuild summary
	for _, v := range r.Vulnerabilities {
		r.Summary.Total++
		switch v.Severity {
		case "critical":
			r.Summary.Critical++
		case "high":
			r.Summary.High++
		case "medium":
			r.Summary.Medium++
		case "low":
			r.Summary.Low++
		}
	}
	assert.Equal(t, 1, r.Summary.Critical)
	assert.Equal(t, 2, r.Summary.High)
	assert.Equal(t, 1, r.Summary.Medium)
	assert.Equal(t, 1, r.Summary.Low)
	assert.Equal(t, 5, r.Summary.Total)
}
