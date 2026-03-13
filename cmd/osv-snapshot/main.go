// Command osv-snapshot fetches OSV vulnerability data for supported ecosystems
// and builds a compressed snapshot for embedding in the scanner binary.
//
// Usage:
//
//	go run ./cmd/osv-snapshot [-o path] [-ecosystems npm,PyPI,Go]
//
// The output is a gzip-compressed JSON file matching the snapshot.snapshotData format.
// Copy the output to internal/analyzer/security/snapshot/data.gz before building.
package main

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// OSV ecosystem dump base URL (GCS public bucket).
const osvBucketURL = "https://osv-vulnerabilities.storage.googleapis.com"

// maxDownloadSize limits ecosystem dump downloads to 500 MB.
const maxDownloadSize = 500 * 1024 * 1024

// osvVuln is the subset of OSV schema fields we need.
type osvVuln struct {
	ID       string `json:"id"`
	Severity []struct {
		Type  string `json:"type"`
		Score string `json:"score"`
	} `json:"severity"`
	Affected []struct {
		Package struct {
			Name      string `json:"name"`
			Ecosystem string `json:"ecosystem"`
		} `json:"package"`
		Ranges []struct {
			Type   string `json:"type"`
			Events []struct {
				Introduced string `json:"introduced"`
				Fixed      string `json:"fixed"`
			} `json:"events"`
		} `json:"ranges"`
	} `json:"affected"`
}

// snapshotEntry matches snapshot.VulnEntry.
type snapshotEntry struct {
	ID       string `json:"id"`
	Severity string `json:"sev"`
	Fixed    string `json:"fix,omitempty"`
	Intro    string `json:"intro,omitempty"`
}

// snapshotData matches snapshot.snapshotData.
type snapshotData struct {
	Date    string                     `json:"date"`
	Entries map[string][]snapshotEntry `json:"entries"`
}

func main() {
	outPath := flag.String("o", "internal/analyzer/security/snapshot/data.gz", "output path for compressed snapshot")
	ecosystems := flag.String("ecosystems", "npm,PyPI,Go", "comma-separated OSV ecosystem names")
	flag.Parse()

	ecoList := strings.Split(*ecosystems, ",")
	data := snapshotData{
		Date:    time.Now().UTC().Format("2006-01-02"),
		Entries: make(map[string][]snapshotEntry),
	}

	client := &http.Client{Timeout: 5 * time.Minute}

	for _, eco := range ecoList {
		eco = strings.TrimSpace(eco)
		log.Printf("Fetching %s ecosystem...", eco)

		vulns, skipped, err := fetchEcosystem(client, eco)
		if err != nil {
			log.Fatalf("Failed to fetch %s: %v", eco, err)
		}

		added := 0
		for _, v := range vulns {
			entries := processVuln(v)
			for key, ents := range entries {
				data.Entries[key] = append(data.Entries[key], ents...)
				added += len(ents)
			}
		}
		log.Printf("  %s: %d vulnerability entries from %d advisories (%d entries skipped/malformed)",
			eco, added, len(vulns), skipped)
	}

	// Write compressed output atomically
	if err := writeSnapshotAtomic(data, *outPath); err != nil {
		log.Fatalf("Failed to write snapshot: %v", err)
	}

	// Report stats
	totalPkgs := len(data.Entries)
	totalVulns := 0
	for _, ents := range data.Entries {
		totalVulns += len(ents)
	}

	fi, _ := os.Stat(*outPath)
	sizeMB := float64(fi.Size()) / (1024 * 1024)
	log.Printf("Snapshot written: %s (%.1f MB, %d packages, %d vulnerability entries)",
		*outPath, sizeMB, totalPkgs, totalVulns)
}

// fetchEcosystem downloads and parses the OSV ecosystem dump zip.
// Returns parsed vulnerabilities and the count of skipped/malformed entries.
func fetchEcosystem(client *http.Client, ecosystem string) ([]osvVuln, int, error) {
	url := fmt.Sprintf("%s/%s/all.zip", osvBucketURL, ecosystem)

	resp, err := client.Get(url)
	if err != nil {
		return nil, 0, fmt.Errorf("GET %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return nil, 0, fmt.Errorf("HTTP %d for %s", resp.StatusCode, url)
	}

	body, err := io.ReadAll(io.LimitReader(resp.Body, maxDownloadSize))
	if err != nil {
		return nil, 0, fmt.Errorf("read body: %w", err)
	}

	zr, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return nil, 0, fmt.Errorf("open zip: %w", err)
	}

	var vulns []osvVuln
	skipped := 0
	for _, f := range zr.File {
		if !strings.HasSuffix(f.Name, ".json") {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			skipped++
			continue
		}
		var v osvVuln
		err = json.NewDecoder(rc).Decode(&v)
		rc.Close()
		if err != nil {
			skipped++
			continue
		}
		vulns = append(vulns, v)
	}

	return vulns, skipped, nil
}

// processVuln converts an OSV vulnerability into snapshot entries keyed by ecosystem:package.
// Each introduced/fixed pair in a range produces a separate entry.
func processVuln(v osvVuln) map[string][]snapshotEntry {
	result := make(map[string][]snapshotEntry)
	severity := classifySeverity(v)

	for _, aff := range v.Affected {
		if aff.Package.Name == "" || aff.Package.Ecosystem == "" {
			continue
		}

		key := aff.Package.Ecosystem + ":" + aff.Package.Name

		for _, rng := range aff.Ranges {
			if rng.Type != "ECOSYSTEM" && rng.Type != "SEMVER" {
				continue
			}

			// Process events pairwise: each "introduced" starts a range,
			// the next "fixed" closes it. This handles multiple ranges per entry.
			introduced := ""
			for _, evt := range rng.Events {
				if evt.Introduced != "" {
					introduced = evt.Introduced
				}
				if evt.Fixed != "" {
					entry := snapshotEntry{
						ID:       v.ID,
						Severity: severity,
						Fixed:    evt.Fixed,
						Intro:    introduced,
					}
					result[key] = append(result[key], entry)
					introduced = "" // reset for next pair
				}
			}
			// If there's an introduced without a corresponding fixed,
			// it means all versions from introduced onward are affected.
			if introduced != "" {
				entry := snapshotEntry{
					ID:       v.ID,
					Severity: severity,
					Intro:    introduced,
				}
				result[key] = append(result[key], entry)
			}
		}
	}

	return result
}

func classifySeverity(v osvVuln) string {
	for _, s := range v.Severity {
		if s.Type == "CVSS_V3" {
			return cvssToSeverity(s.Score)
		}
	}
	return "medium"
}

func cvssToSeverity(score string) string {
	var s float64
	if _, err := fmt.Sscanf(score, "%f", &s); err == nil {
		return severityFromScore(s)
	}

	// Try CVSS vector string
	if strings.HasPrefix(score, "CVSS:") {
		parts := strings.Split(score, "/")
		metrics := make(map[string]string)
		for _, p := range parts {
			kv := strings.SplitN(p, ":", 2)
			if len(kv) == 2 {
				metrics[kv[0]] = kv[1]
			}
		}
		s := 5.0
		switch metrics["AV"] {
		case "N":
			s += 2.0
		case "A":
			s += 1.0
		}
		if metrics["AC"] == "L" {
			s += 1.0
		}
		for _, m := range []string{"C", "I", "A"} {
			if metrics[m] == "H" {
				s += 0.5
			}
		}
		if s > 10.0 {
			s = 10.0
		}
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

// writeSnapshotAtomic writes the snapshot to a temporary file then atomically
// renames it to the target path, preventing corrupt output on failure.
func writeSnapshotAtomic(data snapshotData, path string) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, "snapshot-*.gz.tmp")
	if err != nil {
		return fmt.Errorf("create temp file: %w", err)
	}
	tmpPath := tmp.Name()
	defer os.Remove(tmpPath) // clean up on failure

	gw := gzip.NewWriter(tmp)
	enc := json.NewEncoder(gw)
	if err := enc.Encode(data); err != nil {
		gw.Close()
		tmp.Close()
		return fmt.Errorf("encode snapshot: %w", err)
	}
	if err := gw.Close(); err != nil {
		tmp.Close()
		return fmt.Errorf("close gzip: %w", err)
	}
	if err := tmp.Close(); err != nil {
		return fmt.Errorf("close temp file: %w", err)
	}

	if err := os.Rename(tmpPath, path); err != nil {
		return fmt.Errorf("rename to final path: %w", err)
	}
	return nil
}
