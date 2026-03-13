package security

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
)

// LicenseResult holds the license detection results.
type LicenseResult struct {
	Issues       []LicenseIssue
	IssueCount   int
	Licenses     []DetectedLicense
}

// LicenseIssue represents a license compatibility issue.
type LicenseIssue struct {
	Package    string
	License    string
	Reason     string
}

// DetectedLicense represents a detected license for a dependency.
type DetectedLicense struct {
	Package string
	License string
}

// problematicLicenses are licenses that may cause issues in M&A.
var problematicLicenses = map[string]string{
	"GPL-2.0":        "Strong copyleft — requires derivative works to be GPL-licensed",
	"GPL-3.0":        "Strong copyleft — requires derivative works to be GPL-licensed",
	"AGPL-3.0":       "Network copyleft — server-side use triggers distribution requirements",
	"SSPL-1.0":       "Server Side Public License — restricts SaaS use",
	"EUPL-1.1":       "European Union Public License — copyleft with compatibility complexities",
	"EUPL-1.2":       "European Union Public License — copyleft with compatibility complexities",
	"CC-BY-SA-4.0":   "Share-alike — derivative works must use same license",
	"CC-BY-NC-4.0":   "Non-commercial — restricts commercial use",
	"CC-BY-NC-SA-4.0": "Non-commercial share-alike — restricts commercial use and requires same license",
}

// DetectLicenses scans dependency metadata for license information.
func DetectLicenses(root string) *LicenseResult {
	r := &LicenseResult{}

	// Check npm packages
	detectNPMLicenses(root, r)

	// Check Python packages
	detectPythonLicenses(root, r)

	// Check composer.json
	detectPHPLicenses(root, r)

	r.IssueCount = len(r.Issues)
	return r
}

func detectNPMLicenses(root string, r *LicenseResult) {
	data, err := os.ReadFile(filepath.Join(root, "package.json"))
	if err != nil {
		return
	}

	var pkg struct {
		Dependencies map[string]string `json:"dependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return
	}

	// Check node_modules for license info
	for name := range pkg.Dependencies {
		pkgPath := filepath.Join(root, "node_modules", name, "package.json")
		data, err := os.ReadFile(pkgPath)
		if err != nil {
			continue
		}

		var depPkg struct {
			License interface{} `json:"license"`
		}
		if json.Unmarshal(data, &depPkg) != nil {
			continue
		}

		license := extractLicenseStr(depPkg.License)
		if license == "" {
			continue
		}

		r.Licenses = append(r.Licenses, DetectedLicense{
			Package: name,
			License: license,
		})

		checkLicense(name, license, r)
	}
}

func detectPythonLicenses(root string, r *LicenseResult) {
	// Check pyproject.toml for license field
	data, err := os.ReadFile(filepath.Join(root, "pyproject.toml"))
	if err != nil {
		return
	}

	content := string(data)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "license") && strings.Contains(line, "=") {
			parts := strings.SplitN(line, "=", 2)
			if len(parts) == 2 {
				license := strings.Trim(strings.TrimSpace(parts[1]), `"'{}`)
				if license != "" {
					r.Licenses = append(r.Licenses, DetectedLicense{
						Package: "project",
						License: license,
					})
				}
			}
		}
	}
}

func detectPHPLicenses(root string, r *LicenseResult) {
	data, err := os.ReadFile(filepath.Join(root, "composer.json"))
	if err != nil {
		return
	}

	var composer struct {
		License string `json:"license"`
	}
	if json.Unmarshal(data, &composer) != nil {
		return
	}

	if composer.License != "" {
		r.Licenses = append(r.Licenses, DetectedLicense{
			Package: "project",
			License: composer.License,
		})
		checkLicense("project", composer.License, r)
	}
}

func checkLicense(pkg, license string, r *LicenseResult) {
	upper := strings.ToUpper(strings.TrimSpace(license))

	// Check most specific licenses first (AGPL before GPL, SSPL before others)
	checkOrder := []string{
		"AGPL-3.0", "SSPL-1.0",
		"GPL-3.0", "GPL-2.0",
		"EUPL-1.2", "EUPL-1.1",
		"CC-BY-NC-SA-4.0", "CC-BY-NC-4.0", "CC-BY-SA-4.0",
	}
	for _, spdx := range checkOrder {
		if strings.Contains(upper, strings.ToUpper(spdx)) {
			if reason, ok := problematicLicenses[spdx]; ok {
				r.Issues = append(r.Issues, LicenseIssue{
					Package: pkg,
					License: license,
					Reason:  reason,
				})
				return
			}
		}
	}
}

func extractLicenseStr(v interface{}) string {
	switch val := v.(type) {
	case string:
		return val
	case map[string]interface{}:
		if t, ok := val["type"]; ok {
			if s, ok := t.(string); ok {
				return s
			}
		}
	}
	return ""
}
