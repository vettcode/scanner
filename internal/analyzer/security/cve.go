package security

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vettcode/scanner/internal/analyzer/deps"
	"github.com/vettcode/scanner/internal/analyzer/security/snapshot"
)

const (
	perCallTimeout     = 10 * time.Second // 10s per individual OSV API call
	totalNetworkBudget = 30 * time.Second // 30s total across all CVE queries
)

// CVEResult holds the CVE lookup results.
type CVEResult struct {
	Vulnerabilities   []Vulnerability
	Summary           CVESummary
	EcosystemsSkipped []string // ecosystems not checked (offline mode)
	Warnings          []string // warnings about lookup failures
}

// CVESummary holds CVE counts by severity.
type CVESummary struct {
	Critical int
	High     int
	Medium   int
	Low      int
	Total    int
}

// Vulnerability represents a single CVE finding.
type Vulnerability struct {
	ID          string
	Severity    string // "critical", "high", "medium", "low"
	Package     string
	Version     string
	FixedVersion string
	Ecosystem   string
}

// osvQuery is the request payload for OSV API.
type osvQuery struct {
	Package struct {
		Name      string `json:"name"`
		Ecosystem string `json:"ecosystem"`
	} `json:"package"`
	Version string `json:"version"`
}

// osvResponse is the response from OSV API.
type osvResponse struct {
	Vulns []struct {
		ID       string `json:"id"`
		Summary  string `json:"summary"`
		Severity []struct {
			Type  string `json:"type"`
			Score string `json:"score"`
		} `json:"severity"`
		DatabaseSpecific json.RawMessage `json:"database_specific"`
		Affected []osvAffected `json:"affected"`
	} `json:"vulns"`
}

// ecosystemMap converts our ecosystem names to OSV ecosystem names.
var ecosystemMap = map[string]string{
	"npm":       "npm",
	"pypi":      "PyPI",
	"go":        "Go",
	"packagist": "Packagist",
	"rubygems":  "RubyGems",
	"maven":     "Maven",
}

// offlineEcosystems are ecosystems supported in offline mode.
var offlineEcosystems = map[string]bool{
	"npm":  true,
	"pypi": true,
	"go":   true,
}

// LookupCVEs queries OSV for known vulnerabilities.
// In offline mode, uses the bundled OSV snapshot for supported ecosystems.
// In online mode, queries the OSV API with automatic fallback to the snapshot on timeout.
func LookupCVEs(dependencies []deps.Dependency, offline bool) *CVEResult {
	r := &CVEResult{}
	client := &http.Client{Timeout: perCallTimeout}
	fellBackToSnapshot := false // tracks whether we've already emitted the fallback warning

	// 30s total network budget across all CVE queries
	budgetCtx, budgetCancel := context.WithTimeout(context.Background(), totalNetworkBudget)
	defer budgetCancel()

	// Log snapshot availability in offline mode
	if offline && snapshot.Available() {
		snapshotDate := snapshot.Date()
		if snapshotDate != "" {
			r.Warnings = append(r.Warnings,
				fmt.Sprintf("Using bundled OSV database (last updated: %s). CVE results may be up to 30 days old. To retry: run scan with network access.", snapshotDate))
		}
	}

	for _, dep := range dependencies {
		if dep.Version == "" {
			continue // can't check without a version
		}

		osvEco, ok := ecosystemMap[dep.Ecosystem]
		if !ok {
			continue
		}

		if offline && !offlineEcosystems[dep.Ecosystem] {
			// Track skipped ecosystems (not covered by offline snapshot)
			if !containsStr(r.EcosystemsSkipped, dep.Ecosystem) {
				r.EcosystemsSkipped = append(r.EcosystemsSkipped, dep.Ecosystem)
			}
			continue
		}

		if offline {
			// Offline mode: use bundled OSV snapshot
			lookupSnapshot(r, dep, osvEco)
			continue
		}

		// Check if total network budget is exhausted
		if budgetCtx.Err() != nil {
			if snapshot.Available() && offlineEcosystems[dep.Ecosystem] {
				lookupSnapshot(r, dep, osvEco)
				if !fellBackToSnapshot {
					fellBackToSnapshot = true
					snapshotDate := snapshot.Date()
					r.Warnings = append(r.Warnings,
						fmt.Sprintf("CVE lookup timed out (30s budget). Using bundled OSV database (last updated: %s). CVE results may be up to 30 days old. To retry: run scan with network access.", snapshotDate))
				}
			} else {
				r.Warnings = append(r.Warnings,
					fmt.Sprintf("CVE lookup skipped for %s@%s (%s): network budget exhausted",
						dep.Name, dep.Version, dep.Ecosystem))
			}
			continue
		}

		// Online mode: query OSV API with snapshot fallback on error
		vulns, err := queryOSV(budgetCtx, client, dep.Name, dep.Version, osvEco)
		if err != nil {
			// Fall back to bundled snapshot if available
			if snapshot.Available() && offlineEcosystems[dep.Ecosystem] {
				lookupSnapshot(r, dep, osvEco)
				if !fellBackToSnapshot {
					fellBackToSnapshot = true
					snapshotDate := snapshot.Date()
					r.Warnings = append(r.Warnings,
						fmt.Sprintf("CVE lookup timed out. Using bundled OSV database (last updated: %s). CVE results may be up to 30 days old. To retry: run scan with network access.", snapshotDate))
				}
			} else {
				r.Warnings = append(r.Warnings,
					fmt.Sprintf("CVE lookup failed for %s@%s (%s): %v",
						dep.Name, dep.Version, dep.Ecosystem, err))
			}
			continue
		}
		for _, v := range vulns {
			r.Vulnerabilities = append(r.Vulnerabilities, Vulnerability{
				ID:           v.id,
				Severity:     v.severity,
				Package:      dep.Name,
				Version:      dep.Version,
				FixedVersion: v.fixedVersion,
				Ecosystem:    dep.Ecosystem,
			})
		}
	}

	// Compute summary
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

	return r
}

type vulnInfo struct {
	id           string
	severity     string
	fixedVersion string
}

type osvAffected struct {
	Ranges []struct {
		Events []struct {
			Introduced string `json:"introduced"`
			Fixed      string `json:"fixed"`
		} `json:"events"`
	} `json:"ranges"`
}

// findFixVersionForVersion finds the fix version from the affected range
// that actually contains the installed version. OSV vulns often have multiple
// affected ranges (one per branch), and we need the fix for the user's branch.
func findFixVersionForVersion(affected []osvAffected, version string) string {
	for _, a := range affected {
		for _, r := range a.Ranges {
			var introduced, fixed string
			for _, evt := range r.Events {
				if evt.Introduced != "" {
					introduced = evt.Introduced
				}
				if evt.Fixed != "" {
					fixed = evt.Fixed
				}
			}

			// Check if this range contains our version
			if introduced != "" && introduced != "0" {
				if snapshot.CompareVersions(version, introduced) < 0 {
					continue // version is before this range
				}
			}
			if fixed != "" {
				if snapshot.CompareVersions(version, fixed) >= 0 {
					continue // version is already past this fix
				}
				return fixed // this is the correct fix for our version
			}
			// No fix in this range — vuln is open-ended
			return ""
		}
	}
	// Fallback: no matching range found
	return ""
}

func queryOSV(ctx context.Context, client *http.Client, name, version, ecosystem string) ([]vulnInfo, error) {
	q := osvQuery{Version: version}
	q.Package.Name = name
	q.Package.Ecosystem = ecosystem

	body, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.osv.dev/v1/query", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from OSV API", resp.StatusCode)
	}

	// Limit response to 10 MB to guard against oversized responses
	data, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
	if err != nil {
		return nil, fmt.Errorf("read response: %w", err)
	}

	var osvResp osvResponse
	if err := json.Unmarshal(data, &osvResp); err != nil {
		return nil, fmt.Errorf("unmarshal response: %w", err)
	}

	var results []vulnInfo
	for _, v := range osvResp.Vulns {
		severity := classifySeverity(v.Severity)
		fixedVer := findFixVersionForVersion(v.Affected, version)

		// Skip if the fix version is already <= the installed version.
		// This catches cases where OSV returns a vuln from a different
		// branch (e.g., fix 5.4.50 for a 7.x user).
		if fixedVer != "" && snapshot.CompareVersions(version, fixedVer) >= 0 {
			continue
		}

		results = append(results, vulnInfo{
			id:           v.ID,
			severity:     severity,
			fixedVersion: fixedVer,
		})
	}

	return results, nil
}

func classifySeverity(severities []struct {
	Type  string `json:"type"`
	Score string `json:"score"`
}) string {
	for _, s := range severities {
		if s.Type == "CVSS_V3" {
			return cvssToSeverity(s.Score)
		}
	}
	return "medium" // default if no severity info
}

func cvssToSeverity(score string) string {
	var s float64

	// Try numeric score first
	if _, err := fmt.Sscanf(score, "%f", &s); err == nil {
		return severityFromScore(s)
	}

	// Try parsing CVSS vector string (e.g., "CVSS:3.1/AV:N/AC:L/...")
	// Extract the base score from the vector by computing from impact metrics
	if strings.HasPrefix(score, "CVSS:") {
		s = estimateScoreFromVector(score)
		return severityFromScore(s)
	}

	return "medium"
}

func severityFromScore(s float64) string {
	switch {
	case s >= 9.0:
		return "critical"
	case s >= 7.0:
		return "high"
	case s >= 4.0:
		return "medium"
	default:
		return "low"
	}
}

// estimateScoreFromVector estimates a CVSS score from a vector string.
// This is a simplified estimation based on key metrics.
func estimateScoreFromVector(vector string) float64 {
	parts := strings.Split(vector, "/")
	metrics := make(map[string]string)
	for _, p := range parts {
		kv := strings.SplitN(p, ":", 2)
		if len(kv) == 2 {
			metrics[kv[0]] = kv[1]
		}
	}

	// Score based on Attack Vector + Confidentiality/Integrity/Availability Impact
	score := 5.0 // base medium

	// Attack Vector: Network is worse than Local
	switch metrics["AV"] {
	case "N":
		score += 2.0
	case "A":
		score += 1.0
	case "L", "P":
		score += 0.0
	}

	// Attack Complexity: Low is worse
	if metrics["AC"] == "L" {
		score += 1.0
	}

	// Impact: High C/I/A each adds score
	for _, m := range []string{"C", "I", "A"} {
		if metrics[m] == "H" {
			score += 0.5
		}
	}

	if score > 10.0 {
		score = 10.0
	}
	return score
}

func containsStr(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// lookupSnapshot performs a CVE lookup against the bundled OSV snapshot.
func lookupSnapshot(r *CVEResult, dep deps.Dependency, osvEco string) {
	entries := snapshot.Lookup(osvEco, dep.Name, dep.Version)
	for _, e := range entries {
		r.Vulnerabilities = append(r.Vulnerabilities, Vulnerability{
			ID:           e.ID,
			Severity:     e.Severity,
			Package:      dep.Name,
			Version:      dep.Version,
			FixedVersion: e.FixedVersion,
			Ecosystem:    dep.Ecosystem,
		})
	}
}
