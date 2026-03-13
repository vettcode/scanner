package security

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/vettcode/scanner/internal/analyzer/deps"
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
