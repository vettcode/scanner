package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/vettcode/scanner/internal/analyzer/deps"
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
		Affected []struct {
			Ranges []struct {
				Events []struct {
					Fixed string `json:"fixed"`
				} `json:"events"`
			} `json:"ranges"`
		} `json:"affected"`
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
func LookupCVEs(dependencies []deps.Dependency, offline bool) *CVEResult {
	r := &CVEResult{}
	client := &http.Client{Timeout: 30 * time.Second}

	for _, dep := range dependencies {
		if dep.Version == "" {
			continue // can't check without a version
		}

		osvEco, ok := ecosystemMap[dep.Ecosystem]
		if !ok {
			continue
		}

		if offline && !offlineEcosystems[dep.Ecosystem] {
			// Track skipped ecosystems
			if !containsStr(r.EcosystemsSkipped, dep.Ecosystem) {
				r.EcosystemsSkipped = append(r.EcosystemsSkipped, dep.Ecosystem)
			}
			continue
		}

		if offline {
			// In offline mode, skip API calls
			// TODO: bundled OSV snapshot lookup
			continue
		}

		vulns, err := queryOSV(client, dep.Name, dep.Version, osvEco)
		if err != nil {
			warning := fmt.Sprintf("CVE lookup failed for %s@%s (%s): %v", dep.Name, dep.Version, dep.Ecosystem, err)
			r.Warnings = append(r.Warnings, warning)
			continue
		}
		for _, v := range vulns {
			r.Vulnerabilities = append(r.Vulnerabilities, Vulnerability{
				ID:          v.id,
				Severity:    v.severity,
				Package:     dep.Name,
				Version:     dep.Version,
				FixedVersion: v.fixedVersion,
				Ecosystem:   dep.Ecosystem,
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

func queryOSV(client *http.Client, name, version, ecosystem string) ([]vulnInfo, error) {
	q := osvQuery{Version: version}
	q.Package.Name = name
	q.Package.Ecosystem = ecosystem

	body, err := json.Marshal(q)
	if err != nil {
		return nil, fmt.Errorf("marshal query: %w", err)
	}

	resp, err := client.Post("https://api.osv.dev/v1/query", "application/json", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("HTTP %d from OSV API", resp.StatusCode)
	}

	data, err := io.ReadAll(resp.Body)
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
		fixedVer := ""
		if len(v.Affected) > 0 && len(v.Affected[0].Ranges) > 0 {
			for _, evt := range v.Affected[0].Ranges[0].Events {
				if evt.Fixed != "" {
					fixedVer = evt.Fixed
					break
				}
			}
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
