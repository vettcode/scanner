package deps

import (
	"bufio"
	"encoding/json"
	"os"
	"path/filepath"
	"regexp"
	"strings"
)

// Dependency represents a single dependency.
type Dependency struct {
	Name      string
	Version   string // locked version, may be empty
	Ecosystem string // "npm", "pypi", "go", "packagist", "rubygems", "maven"
	Language  string
	Direct    bool   // true if declared in manifest; false if transitive/indirect
}

// ParseResult holds all dependencies found in a repository.
type ParseResult struct {
	Dependencies []Dependency
	Ecosystems   []string // ecosystems detected
}

// ParseDependencies scans a repository root for dependency manifests and lockfiles.
func ParseDependencies(root string) *ParseResult {
	r := &ParseResult{}
	ecoSet := make(map[string]bool)

	// npm (package.json + lockfiles)
	if deps := parseNPM(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["npm"] = true
	}

	// Python (requirements.txt, Pipfile.lock, poetry.lock, pyproject.toml)
	if deps := parsePython(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["pypi"] = true
	}

	// Go (go.mod/go.sum)
	if deps := parseGo(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["go"] = true
	}

	// PHP (composer.json/composer.lock)
	if deps := parsePHP(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["packagist"] = true
	}

	// Ruby (Gemfile.lock)
	if deps := parseRuby(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["rubygems"] = true
	}

	// Java (pom.xml, build.gradle)
	if deps := parseJava(root); len(deps) > 0 {
		r.Dependencies = append(r.Dependencies, deps...)
		ecoSet["maven"] = true
	}

	for eco := range ecoSet {
		r.Ecosystems = append(r.Ecosystems, eco)
	}

	return r
}

// parseNPM extracts dependencies from package.json, using package-lock.json
// for exact locked versions when available. Without the lockfile, version
// ranges like "^8" get cleaned to "8" which causes false-positive CVE matches.
func parseNPM(root string) []Dependency {
	pkgPath := filepath.Join(root, "package.json")
	data, err := os.ReadFile(pkgPath)
	if err != nil {
		return nil
	}

	var pkg struct {
		Dependencies    map[string]string `json:"dependencies"`
		DevDependencies map[string]string `json:"devDependencies"`
	}
	if json.Unmarshal(data, &pkg) != nil {
		return nil
	}

	// Build locked version map from lockfiles if available.
	// Tries: package-lock.json, yarn.lock, pnpm-lock.yaml (first found wins).
	locked := parseNPMLockVersions(root)

	var deps []Dependency
	seen := make(map[string]bool)

	addDeps := func(m map[string]string) {
		for name, version := range m {
			if !seen[name] {
				seen[name] = true
				v := cleanVersion(version)
				// Prefer the exact locked version over the cleaned range.
				if lv, ok := locked[name]; ok {
					v = lv
				}
				// Skip deps without a usable semver version (e.g. "8" from "^8").
				// These cause false-positive CVE matches. A valid npm version
				// has at least two dot-separated segments (e.g. "8.5.8" or "14.2").
				if !looksLikeSemver(v) {
					continue
				}
				deps = append(deps, Dependency{
					Name:      name,
					Version:   v,
					Ecosystem: "npm",
					Language:  "JavaScript",
					Direct:    true,
				})
			}
		}
	}

	addDeps(pkg.Dependencies)
	addDeps(pkg.DevDependencies)

	return deps
}

// looksLikeSemver returns true if the version string has at least two
// dot-separated numeric segments (e.g. "8.5", "14.2.35"). Single numbers
// like "8" (from cleaned ranges like "^8") are not reliable for CVE matching.
// Also rejects constraint expressions containing | (OR), spaces, or commas.
func looksLikeSemver(v string) bool {
	// Reject constraint expressions (e.g., "^10.5.35|^11.5.3|^12.0.1")
	if strings.ContainsAny(v, "| ,") {
		return false
	}
	parts := strings.SplitN(v, ".", 3)
	if len(parts) < 2 {
		return false
	}
	// First two parts should start with a digit.
	for _, p := range parts[:2] {
		if len(p) == 0 || p[0] < '0' || p[0] > '9' {
			return false
		}
	}
	return true
}

// parseNPMLockVersions tries package-lock.json, yarn.lock, and pnpm-lock.yaml
// (in that order) to resolve exact locked versions for direct dependencies.
func parseNPMLockVersions(root string) map[string]string {
	if result := parsePackageLockJSON(root); result != nil {
		return result
	}
	if result := parseYarnLock(root); result != nil {
		return result
	}
	return parsePnpmLock(root)
}

// parsePackageLockJSON reads package-lock.json (v1/v2/v3) for locked versions.
func parsePackageLockJSON(root string) map[string]string {
	data, err := os.ReadFile(filepath.Join(root, "package-lock.json"))
	if err != nil {
		return nil
	}

	var lockfile struct {
		Packages map[string]struct {
			Version string `json:"version"`
		} `json:"packages"`
		Dependencies map[string]struct {
			Version string `json:"version"`
		} `json:"dependencies"`
	}
	if json.Unmarshal(data, &lockfile) != nil {
		return nil
	}

	result := make(map[string]string)

	// v2/v3: top-level deps are at "node_modules/<name>"
	if len(lockfile.Packages) > 0 {
		for key, entry := range lockfile.Packages {
			if !strings.HasPrefix(key, "node_modules/") {
				continue
			}
			// Only top-level: skip nested (e.g. "node_modules/next/node_modules/postcss")
			name := strings.TrimPrefix(key, "node_modules/")
			if strings.Contains(name, "node_modules/") {
				continue
			}
			if entry.Version != "" {
				result[name] = entry.Version
			}
		}
		return result
	}

	// v1 fallback
	for name, entry := range lockfile.Dependencies {
		if entry.Version != "" {
			result[name] = entry.Version
		}
	}
	return result
}

// parseYarnLock reads yarn.lock (classic v1 format) for locked versions.
// Format:
//
//	package-name@^range:
//	  version "1.2.3"
func parseYarnLock(root string) map[string]string {
	f, err := os.Open(filepath.Join(root, "yarn.lock"))
	if err != nil {
		return nil
	}
	defer f.Close()

	result := make(map[string]string)
	var currentNames []string

	yarnPkgRe := regexp.MustCompile(`^"?([^@\s]+)@`)
	yarnVerRe := regexp.MustCompile(`^\s+version\s+"([^"]+)"`)

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.HasPrefix(line, "#") || line == "" {
			continue
		}

		// Package header line(s): "postcss@^8, postcss@^8.4.0:"
		if !strings.HasPrefix(line, " ") && strings.Contains(line, "@") {
			currentNames = nil
			// Split by ", " for combined entries
			for _, entry := range strings.Split(strings.TrimSuffix(line, ":"), ", ") {
				entry = strings.Trim(entry, "\"")
				m := yarnPkgRe.FindStringSubmatch(entry)
				if len(m) >= 2 {
					currentNames = append(currentNames, m[1])
				}
			}
			continue
		}

		// Version line
		if m := yarnVerRe.FindStringSubmatch(line); len(m) >= 2 && len(currentNames) > 0 {
			for _, name := range currentNames {
				if _, exists := result[name]; !exists {
					result[name] = m[1]
				}
			}
			currentNames = nil
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// parsePnpmLock reads pnpm-lock.yaml for locked versions.
// Format (v6+):
//
//	dependencies:
//	  package-name:
//	    specifier: ^1.2.3
//	    version: 1.2.3
func parsePnpmLock(root string) map[string]string {
	f, err := os.Open(filepath.Join(root, "pnpm-lock.yaml"))
	if err != nil {
		return nil
	}
	defer f.Close()

	result := make(map[string]string)
	var currentPkg string
	inDeps := false

	scanner := bufio.NewScanner(f)
	pnpmPkgRe := regexp.MustCompile(`^\s{4}'?([^:'\s]+)'?:$`)
	pnpmVerRe := regexp.MustCompile(`^\s{6}version:\s+'?([^'\s]+)'?`)

	for scanner.Scan() {
		line := scanner.Text()

		// Section headers
		if line == "dependencies:" || line == "devDependencies:" || line == "optionalDependencies:" {
			inDeps = true
			currentPkg = ""
			continue
		}
		// New top-level section ends deps
		if len(line) > 0 && line[0] != ' ' {
			inDeps = false
			currentPkg = ""
			continue
		}

		if !inDeps {
			continue
		}

		// Package name at 4-space indent
		if m := pnpmPkgRe.FindStringSubmatch(line); len(m) >= 2 {
			currentPkg = m[1]
			continue
		}

		// Version at 6-space indent
		if currentPkg != "" {
			if m := pnpmVerRe.FindStringSubmatch(line); len(m) >= 2 {
				ver := m[1]
				// pnpm may include peer suffixes like "8.5.8(postcss@8.5.8)" — strip them
				if idx := strings.IndexByte(ver, '('); idx > 0 {
					ver = ver[:idx]
				}
				if _, exists := result[currentPkg]; !exists {
					result[currentPkg] = ver
				}
				currentPkg = ""
			}
		}
	}

	if len(result) == 0 {
		return nil
	}
	return result
}

// parsePython extracts dependencies from requirements.txt.
func parsePython(root string) []Dependency {
	var deps []Dependency
	seen := make(map[string]bool)

	// requirements.txt
	for _, name := range []string{"requirements.txt", "requirements-dev.txt"} {
		path := filepath.Join(root, name)
		if parsed := parseRequirementsTxt(path); len(parsed) > 0 {
			for _, d := range parsed {
				if !seen[d.Name] {
					seen[d.Name] = true
					deps = append(deps, d)
				}
			}
		}
	}

	// pyproject.toml (simple extraction)
	if pyDeps := parsePyprojectDeps(filepath.Join(root, "pyproject.toml")); len(pyDeps) > 0 {
		for _, d := range pyDeps {
			if !seen[d.Name] {
				seen[d.Name] = true
				deps = append(deps, d)
			}
		}
	}

	return deps
}

func parseRequirementsTxt(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	re := regexp.MustCompile(`^([a-zA-Z0-9_.-]+)\s*([><=!~]+\s*[\d.]+)?`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") || strings.HasPrefix(line, "-") {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			version := ""
			if len(matches) >= 3 {
				version = strings.TrimSpace(matches[2])
			}
			deps = append(deps, Dependency{
				Name:      matches[1],
				Version:   cleanVersion(version),
				Ecosystem: "pypi",
				Language:  "Python",
				Direct:    true,
			})
		}
	}
	return deps
}

func parsePyprojectDeps(path string) []Dependency {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var deps []Dependency
	inDeps := false
	re := regexp.MustCompile(`^\s*"([a-zA-Z0-9_.-]+)"`)

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "dependencies = [" || trimmed == `dependencies = [` {
			inDeps = true
			continue
		}
		if inDeps {
			if trimmed == "]" {
				break
			}
			matches := re.FindStringSubmatch(trimmed)
			if len(matches) >= 2 {
				deps = append(deps, Dependency{
					Name:      matches[1],
					Ecosystem: "pypi",
					Language:  "Python",
				})
			}
		}
	}
	return deps
}

// parseGo extracts dependencies from go.mod.
func parseGo(root string) []Dependency {
	path := filepath.Join(root, "go.mod")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var deps []Dependency
	inRequire := false
	blockRe := regexp.MustCompile(`^\s*([^\s]+)\s+v([^\s]+)`)
	singleRe := regexp.MustCompile(`^require\s+([^\s]+)\s+v([^\s]+)`)

	for _, line := range strings.Split(string(data), "\n") {
		trimmed := strings.TrimSpace(line)

		if trimmed == "require (" {
			inRequire = true
			continue
		}
		if inRequire && trimmed == ")" {
			inRequire = false
			continue
		}

		if inRequire {
			matches := blockRe.FindStringSubmatch(trimmed)
			if len(matches) >= 3 {
				deps = append(deps, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Ecosystem: "go",
					Language:  "Go",
					Direct:    !strings.Contains(trimmed, "// indirect"),
				})
			}
		} else {
			// Handle single-line require: require github.com/pkg/errors v0.9.1
			matches := singleRe.FindStringSubmatch(trimmed)
			if len(matches) >= 3 {
				deps = append(deps, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Ecosystem: "go",
					Language:  "Go",
					Direct:    !strings.Contains(trimmed, "// indirect"),
				})
			}
		}
	}
	return deps
}

// parsePHP extracts dependencies from composer.lock (preferred) or composer.json.
func parsePHP(root string) []Dependency {
	// Try composer.lock first for exact versions
	lockVersions := parseComposerLock(root)

	path := filepath.Join(root, "composer.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var composer struct {
		Require    map[string]string `json:"require"`
		RequireDev map[string]string `json:"require-dev"`
	}
	if json.Unmarshal(data, &composer) != nil {
		return nil
	}

	var deps []Dependency
	seen := make(map[string]bool)

	addDeps := func(m map[string]string) {
		for name, version := range m {
			if name == "php" || strings.HasPrefix(name, "ext-") {
				continue // skip PHP version and extension requirements
			}
			if !seen[name] {
				seen[name] = true
				// Prefer locked version from composer.lock
				v := lockVersions[name]
				if v == "" {
					v = cleanVersion(version)
				}
				if !looksLikeSemver(v) {
					continue
				}
				deps = append(deps, Dependency{
					Name:      name,
					Version:   v,
					Ecosystem: "packagist",
					Language:  "PHP",
					Direct:    true,
				})
			}
		}
	}

	addDeps(composer.Require)
	addDeps(composer.RequireDev)

	return deps
}

// parseComposerLock reads composer.lock and returns a map of package name to locked version.
func parseComposerLock(root string) map[string]string {
	path := filepath.Join(root, "composer.lock")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var lock struct {
		Packages    []composerLockPackage `json:"packages"`
		PackagesDev []composerLockPackage `json:"packages-dev"`
	}
	if json.Unmarshal(data, &lock) != nil {
		return nil
	}

	versions := make(map[string]string)
	for _, p := range lock.Packages {
		v := strings.TrimPrefix(p.Version, "v")
		if looksLikeSemver(v) {
			versions[p.Name] = v
		}
	}
	for _, p := range lock.PackagesDev {
		if _, exists := versions[p.Name]; !exists {
			v := strings.TrimPrefix(p.Version, "v")
			if looksLikeSemver(v) {
				versions[p.Name] = v
			}
		}
	}
	return versions
}

type composerLockPackage struct {
	Name    string `json:"name"`
	Version string `json:"version"`
}

// parseRuby extracts dependencies from Gemfile.lock.
func parseRuby(root string) []Dependency {
	path := filepath.Join(root, "Gemfile.lock")
	f, err := os.Open(path)
	if err != nil {
		// Fall back to Gemfile
		return parseGemfile(filepath.Join(root, "Gemfile"))
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	inSpecs := false
	re := regexp.MustCompile(`^\s{4}([a-zA-Z0-9_.-]+)\s+\(([^)]+)\)`)

	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if trimmed == "specs:" {
			inSpecs = true
			continue
		}
		if inSpecs && !strings.HasPrefix(line, "    ") && trimmed != "" {
			inSpecs = false
		}

		if inSpecs {
			matches := re.FindStringSubmatch(line)
			if len(matches) >= 3 {
				deps = append(deps, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Ecosystem: "rubygems",
					Language:  "Ruby",
					Direct:    true,
				})
			}
		}
	}
	return deps
}

func parseGemfile(path string) []Dependency {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var deps []Dependency
	scanner := bufio.NewScanner(f)
	re := regexp.MustCompile(`gem\s+['"]([a-zA-Z0-9_.-]+)['"]`)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "#") {
			continue
		}
		matches := re.FindStringSubmatch(line)
		if len(matches) >= 2 {
			deps = append(deps, Dependency{
				Name:      matches[1],
				Ecosystem: "rubygems",
				Language:  "Ruby",
				Direct:    true,
			})
		}
	}
	return deps
}

// parseJava extracts dependencies from pom.xml or build.gradle.
func parseJava(root string) []Dependency {
	// Try pom.xml first
	if deps := parsePomXML(filepath.Join(root, "pom.xml")); len(deps) > 0 {
		return deps
	}
	// Try build.gradle
	if deps := parseBuildGradle(filepath.Join(root, "build.gradle")); len(deps) > 0 {
		return deps
	}
	return parseBuildGradle(filepath.Join(root, "build.gradle.kts"))
}

func parsePomXML(path string) []Dependency {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	content := string(data)

	// Extract Maven properties for ${...} resolution.
	props := parseMavenProperties(content)

	var deps []Dependency

	// Simple regex extraction of dependencies from pom.xml
	re := regexp.MustCompile(`<dependency>\s*<groupId>([^<]+)</groupId>\s*<artifactId>([^<]+)</artifactId>\s*(?:<version>([^<]+)</version>)?`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		name := m[1] + ":" + m[2]
		version := ""
		if len(m) >= 4 {
			version = resolveMavenProperty(m[3], props)
		}
		deps = append(deps, Dependency{
			Name:      name,
			Version:   cleanVersion(version),
			Ecosystem: "maven",
			Language:  "Java",
			Direct:    true,
		})
	}
	return deps
}

// parseMavenProperties extracts <properties> key-value pairs from a POM.
func parseMavenProperties(content string) map[string]string {
	props := make(map[string]string)
	re := regexp.MustCompile(`<properties>([\s\S]*?)</properties>`)
	block := re.FindStringSubmatch(content)
	if len(block) < 2 {
		return props
	}
	propRe := regexp.MustCompile(`<([a-zA-Z0-9._-]+)>([^<]*)</([a-zA-Z0-9._-]+)>`)
	for _, m := range propRe.FindAllStringSubmatch(block[1], -1) {
		if m[1] == m[3] { // opening and closing tags match
			props[m[1]] = strings.TrimSpace(m[2])
		}
	}
	return props
}

// resolveMavenProperty replaces ${property.name} references with their values.
func resolveMavenProperty(version string, props map[string]string) string {
	if !strings.Contains(version, "${") {
		return version
	}
	re := regexp.MustCompile(`\$\{([^}]+)\}`)
	return re.ReplaceAllStringFunc(version, func(match string) string {
		key := match[2 : len(match)-1] // strip ${ and }
		if val, ok := props[key]; ok {
			return val
		}
		return match // leave unresolved if not found
	})
}

func parseBuildGradle(path string) []Dependency {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil
	}

	var deps []Dependency
	// Match patterns like: implementation 'group:artifact:version'
	re := regexp.MustCompile(`(?:implementation|api|compile|testImplementation|runtimeOnly)\s+['"]([^'"]+)['"]`)
	matches := re.FindAllStringSubmatch(string(data), -1)

	for _, m := range matches {
		parts := strings.SplitN(m[1], ":", 3)
		name := m[1]
		version := ""
		if len(parts) >= 2 {
			name = parts[0] + ":" + parts[1]
		}
		if len(parts) >= 3 {
			version = parts[2]
		}
		deps = append(deps, Dependency{
			Name:      name,
			Version:   cleanVersion(version),
			Ecosystem: "maven",
			Language:  "Java",
			Direct:    true,
		})
	}
	return deps
}

func cleanVersion(v string) string {
	v = strings.TrimSpace(v)
	v = strings.TrimPrefix(v, "^")
	v = strings.TrimPrefix(v, "~")
	v = strings.TrimPrefix(v, ">=")
	v = strings.TrimPrefix(v, "==")
	v = strings.TrimPrefix(v, "v")
	return v
}
