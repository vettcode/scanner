// Package snapshot provides an embedded OSV database snapshot for offline CVE lookup.
// The snapshot covers npm, PyPI, and Go ecosystems and is updated at each scanner release.
// Build the snapshot with: go run ./cmd/osv-snapshot
package snapshot

import (
	"bytes"
	"compress/gzip"
	_ "embed"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"sync"
)

//go:embed data.gz
var embeddedData []byte

// VulnEntry represents a single vulnerability record in the snapshot.
type VulnEntry struct {
	ID           string `json:"id"`
	Severity     string `json:"sev"`            // critical/high/medium/low
	FixedVersion string `json:"fix,omitempty"`   // version that fixes the vuln
	Introduced   string `json:"intro,omitempty"` // version that introduced the vuln ("0" = all)
}

// snapshotData is the top-level serialized format.
type snapshotData struct {
	Date    string                 `json:"date"`    // YYYY-MM-DD snapshot date
	Entries map[string][]VulnEntry `json:"entries"` // key: "ecosystem:package"
}

var (
	mu    sync.Mutex
	once  sync.Once
	index map[string][]VulnEntry
	date  string
)

func load() {
	index = make(map[string][]VulnEntry)

	if len(embeddedData) == 0 {
		return
	}

	gr, err := gzip.NewReader(bytes.NewReader(embeddedData))
	if err != nil {
		return
	}
	defer gr.Close()

	var data snapshotData
	if err := json.NewDecoder(gr).Decode(&data); err != nil {
		return
	}

	date = data.Date
	index = data.Entries
}

// Lookup returns vulnerability entries matching the given package in the bundled snapshot.
// It checks version ranges: a vulnerability matches if version >= introduced AND version < fixed.
// Returns nil for empty version strings or non-version strings.
func Lookup(ecosystem, name, version string) []VulnEntry {
	if version == "" || !looksLikeVersion(version) {
		return nil
	}

	once.Do(load)

	key := ecosystem + ":" + name
	entries, ok := index[key]
	if !ok {
		return nil
	}

	var matches []VulnEntry
	for _, e := range entries {
		if versionAffected(version, e.Introduced, e.FixedVersion) {
			matches = append(matches, e)
		}
	}
	return matches
}

// Date returns the snapshot date (YYYY-MM-DD), or "" if no snapshot is loaded.
func Date() string {
	once.Do(load)
	return date
}

// Available returns true if a non-empty snapshot is loaded.
func Available() bool {
	once.Do(load)
	return len(index) > 0
}

// Stats returns the number of packages and total vulnerability entries in the snapshot.
func Stats() (packages, vulns int) {
	once.Do(load)
	packages = len(index)
	for _, entries := range index {
		vulns += len(entries)
	}
	return
}

// Override replaces the loaded snapshot with the given data.
// This is intended for testing only.
func Override(snapshotDate string, entries map[string][]VulnEntry) {
	mu.Lock()
	defer mu.Unlock()
	once.Do(func() {}) // ensure once is done
	date = snapshotDate
	index = entries
}

// Reset clears the snapshot state, allowing load() to run again.
// This is intended for testing only.
func Reset() {
	mu.Lock()
	defer mu.Unlock()
	once = sync.Once{}
	index = nil
	date = ""
}

// looksLikeVersion returns true if s starts with a digit or 'v' followed by a digit.
// Rejects non-version strings like "latest", "*", "dev-master".
func looksLikeVersion(s string) bool {
	s = strings.TrimPrefix(s, "v")
	if len(s) == 0 {
		return false
	}
	return s[0] >= '0' && s[0] <= '9'
}

// versionAffected checks if version falls within the affected range [introduced, fixed).
// If introduced is empty or "0", the vulnerability affects all versions before fixed.
// If fixed is empty, the vulnerability affects all versions from introduced onward.
func versionAffected(version, introduced, fixed string) bool {
	// If introduced is specified and non-zero, version must be >= introduced
	if introduced != "" && introduced != "0" {
		if compareVersions(version, introduced) < 0 {
			return false
		}
	}

	// If fixed is specified, version must be < fixed
	if fixed != "" {
		if compareVersions(version, fixed) >= 0 {
			return false
		}
	}

	return true
}

// compareVersions compares two version strings.
// Returns -1 if a < b, 0 if a == b, 1 if a > b.
// Handles semver (X.Y.Z), PEP 440 basics, and pre-release suffixes.
func compareVersions(a, b string) int {
	partsA := parseVersion(a)
	partsB := parseVersion(b)

	maxLen := len(partsA)
	if len(partsB) > maxLen {
		maxLen = len(partsB)
	}

	for i := 0; i < maxLen; i++ {
		var pA, pB versionPart
		if i < len(partsA) {
			pA = partsA[i]
		}
		if i < len(partsB) {
			pB = partsB[i]
		}

		if pA.num != pB.num {
			if pA.num < pB.num {
				return -1
			}
			return 1
		}
		// If numeric parts are equal, compare string suffixes
		if pA.suffix != pB.suffix {
			return compareSuffix(pA.suffix, pB.suffix)
		}
	}
	return 0
}

type versionPart struct {
	num    int
	suffix string // e.g., "rc1", "beta2", "a1"
}

func parseVersion(v string) []versionPart {
	// Strip leading 'v' prefix
	v = strings.TrimPrefix(v, "v")

	// Strip build metadata (semver: +build should be ignored for precedence)
	if idx := strings.IndexByte(v, '+'); idx >= 0 {
		v = v[:idx]
	}

	// Split on dots
	segments := strings.Split(v, ".")
	parts := make([]versionPart, 0, len(segments))

	for _, seg := range segments {
		p := versionPart{}
		// Also split on '-' for semver pre-release (e.g., "1.0.0-rc.1")
		// Handle the first '-' separated part
		if dashIdx := strings.IndexByte(seg, '-'); dashIdx >= 0 {
			numStr := seg[:dashIdx]
			p.suffix = strings.ToLower(seg[dashIdx+1:])
			if numStr != "" {
				p.num, _ = strconv.Atoi(numStr)
			}
			parts = append(parts, p)
			continue
		}

		// Find where digits end and suffix begins
		numStr := seg
		for i, c := range seg {
			if c < '0' || c > '9' {
				numStr = seg[:i]
				p.suffix = strings.ToLower(seg[i:])
				break
			}
		}
		if numStr != "" {
			p.num, _ = strconv.Atoi(numStr)
		}
		parts = append(parts, p)
	}

	return parts
}

// compareSuffix compares version suffixes.
// Empty suffix (release) > pre-release suffixes (alpha, beta, rc).
// Handles numbered suffixes correctly (rc2 < rc10).
func compareSuffix(a, b string) int {
	// No suffix (release) is greater than any pre-release suffix
	if a == "" && b != "" {
		return 1
	}
	if a != "" && b == "" {
		return -1
	}

	// Extract alphabetic prefix and numeric suffix for proper ordering
	aPrefix, aNum := splitSuffixNum(a)
	bPrefix, bNum := splitSuffixNum(b)

	if aPrefix != bPrefix {
		if aPrefix < bPrefix {
			return -1
		}
		return 1
	}

	// Same prefix, compare numeric parts
	if aNum != bNum {
		if aNum < bNum {
			return -1
		}
		return 1
	}
	return 0
}

// splitSuffixNum splits a suffix like "rc10" into ("rc", 10).
func splitSuffixNum(s string) (string, int) {
	i := len(s)
	for i > 0 && s[i-1] >= '0' && s[i-1] <= '9' {
		i--
	}
	if i == len(s) {
		return s, 0
	}
	num, _ := strconv.Atoi(s[i:])
	return s[:i], num
}

// FormatKey builds the index key for an ecosystem and package name.
func FormatKey(ecosystem, name string) string {
	return fmt.Sprintf("%s:%s", ecosystem, name)
}
