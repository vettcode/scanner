package snapshot

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"1.0.0", "1.0.0", 0},
		{"1.0.0", "1.0.1", -1},
		{"1.0.1", "1.0.0", 1},
		{"2.0.0", "1.9.9", 1},
		{"1.10.0", "1.9.0", 1},
		{"1.0", "1.0.0", 0},
		{"1.0.0", "1.0", 0},
		{"v1.2.3", "1.2.3", 0},
		{"0.1.0", "0.2.0", -1},
	}
	for _, tt := range tests {
		t.Run(tt.a+"_vs_"+tt.b, func(t *testing.T) {
			assert.Equal(t, tt.want, CompareVersions(tt.a, tt.b))
		})
	}
}

func TestCompareVersions_PreRelease(t *testing.T) {
	// Release > pre-release
	assert.Equal(t, 1, CompareVersions("1.0.0", "1.0.0rc1"))
	assert.Equal(t, -1, CompareVersions("1.0.0beta1", "1.0.0"))

	// Alpha < beta < rc
	assert.Equal(t, -1, CompareVersions("1.0.0alpha1", "1.0.0beta1"))
	assert.Equal(t, -1, CompareVersions("1.0.0beta1", "1.0.0rc1"))
}

func TestCompareVersions_BuildMetadata(t *testing.T) {
	// Build metadata should be ignored per semver
	assert.Equal(t, 0, CompareVersions("1.0.0+build123", "1.0.0"))
	assert.Equal(t, 0, CompareVersions("1.0.0+build.1", "1.0.0+build.2"))
	assert.Equal(t, -1, CompareVersions("1.0.0+build", "1.0.1"))
}

func TestCompareVersions_NumberedSuffixes(t *testing.T) {
	// rc9 < rc10 (numeric comparison, not lexicographic)
	assert.Equal(t, -1, CompareVersions("1.0.0rc9", "1.0.0rc10"))
	assert.Equal(t, -1, CompareVersions("1.0.0alpha2", "1.0.0alpha10"))
}

func TestVersionAffected(t *testing.T) {
	tests := []struct {
		name       string
		version    string
		introduced string
		fixed      string
		want       bool
	}{
		{"in range", "1.5.0", "1.0.0", "2.0.0", true},
		{"below introduced", "0.9.0", "1.0.0", "2.0.0", false},
		{"at fixed", "2.0.0", "1.0.0", "2.0.0", false},
		{"above fixed", "2.1.0", "1.0.0", "2.0.0", false},
		{"at introduced", "1.0.0", "1.0.0", "2.0.0", true},
		{"no introduced (all before fix)", "0.5.0", "", "1.0.0", true},
		{"zero introduced (all before fix)", "0.5.0", "0", "1.0.0", true},
		{"no fixed (all from introduced)", "5.0.0", "1.0.0", "", true},
		{"no range (all versions)", "99.0.0", "", "", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, versionAffected(tt.version, tt.introduced, tt.fixed))
		})
	}
}

func TestLookup_EmptySnapshot(t *testing.T) {
	// With the placeholder empty snapshot, Lookup should return nil
	results := Lookup("npm", "express", "4.17.1")
	assert.Nil(t, results)
}

func TestLookup_WithOverride(t *testing.T) {
	Override("2026-03-01", map[string][]VulnEntry{
		"npm:express": {
			{ID: "CVE-2024-1000", Severity: "high", FixedVersion: "4.18.0", Introduced: "4.0.0"},
			{ID: "CVE-2024-2000", Severity: "critical", FixedVersion: "4.17.2", Introduced: "4.17.0"},
			{ID: "CVE-2024-3000", Severity: "low", FixedVersion: "3.0.0", Introduced: "2.0.0"},
		},
	})
	defer Reset()

	// 4.17.1 is in [4.0.0, 4.18.0) → matches CVE-2024-1000
	// 4.17.1 is in [4.17.0, 4.17.2) → matches CVE-2024-2000
	// 4.17.1 is NOT in [2.0.0, 3.0.0) → does NOT match CVE-2024-3000
	results := Lookup("npm", "express", "4.17.1")
	require.Len(t, results, 2)
	assert.Equal(t, "CVE-2024-1000", results[0].ID)
	assert.Equal(t, "CVE-2024-2000", results[1].ID)

	// Fixed version should not match
	results = Lookup("npm", "express", "4.18.0")
	assert.Len(t, results, 0)

	// Package not in snapshot
	results = Lookup("npm", "lodash", "4.0.0")
	assert.Nil(t, results)
}

func TestLookup_RejectsNonVersionStrings(t *testing.T) {
	Override("2026-03-01", map[string][]VulnEntry{
		"npm:express": {
			{ID: "CVE-2024-1000", Severity: "high", Introduced: "", FixedVersion: ""},
		},
	})
	defer Reset()

	// Non-version strings should return nil, not match all
	assert.Nil(t, Lookup("npm", "express", "latest"))
	assert.Nil(t, Lookup("npm", "express", "*"))
	assert.Nil(t, Lookup("npm", "express", "dev-master"))
	assert.Nil(t, Lookup("npm", "express", ""))
}

func TestAvailable_EmptySnapshot(t *testing.T) {
	assert.False(t, Available())
}

func TestAvailable_WithOverride(t *testing.T) {
	Override("2026-03-01", map[string][]VulnEntry{
		"npm:test": {{ID: "CVE-1"}},
	})
	defer Reset()
	assert.True(t, Available())
}

func TestDate_EmptySnapshot(t *testing.T) {
	assert.Equal(t, "", Date())
}

func TestFormatKey(t *testing.T) {
	assert.Equal(t, "npm:express", FormatKey("npm", "express"))
	assert.Equal(t, "PyPI:requests", FormatKey("PyPI", "requests"))
}

func TestParseVersion(t *testing.T) {
	parts := parseVersion("v1.2.3")
	assert.Len(t, parts, 3)
	assert.Equal(t, 1, parts[0].num)
	assert.Equal(t, 2, parts[1].num)
	assert.Equal(t, 3, parts[2].num)

	// Pre-release
	parts = parseVersion("1.0.0rc1")
	assert.Len(t, parts, 3)
	assert.Equal(t, 0, parts[2].num)
	assert.Equal(t, "rc1", parts[2].suffix)

	// Build metadata stripped
	parts = parseVersion("1.0.0+build123")
	assert.Len(t, parts, 3)
	assert.Equal(t, 0, parts[2].num)
	assert.Equal(t, "", parts[2].suffix)
}

func TestLooksLikeVersion(t *testing.T) {
	assert.True(t, looksLikeVersion("1.0.0"))
	assert.True(t, looksLikeVersion("v2.3.4"))
	assert.True(t, looksLikeVersion("0.1"))
	assert.False(t, looksLikeVersion("latest"))
	assert.False(t, looksLikeVersion("*"))
	assert.False(t, looksLikeVersion("dev-master"))
	assert.False(t, looksLikeVersion(""))
	assert.False(t, looksLikeVersion("v"))
}

func TestSplitSuffixNum(t *testing.T) {
	prefix, num := splitSuffixNum("rc10")
	assert.Equal(t, "rc", prefix)
	assert.Equal(t, 10, num)

	prefix, num = splitSuffixNum("alpha")
	assert.Equal(t, "alpha", prefix)
	assert.Equal(t, 0, num)

	prefix, num = splitSuffixNum("2")
	assert.Equal(t, "", prefix)
	assert.Equal(t, 2, num)
}
