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

// parseNPM extracts dependencies from package.json.
func parseNPM(root string) []Dependency {
	path := filepath.Join(root, "package.json")
	data, err := os.ReadFile(path)
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

	var deps []Dependency
	seen := make(map[string]bool)

	addDeps := func(m map[string]string) {
		for name, version := range m {
			if !seen[name] {
				seen[name] = true
				deps = append(deps, Dependency{
					Name:      name,
					Version:   cleanVersion(version),
					Ecosystem: "npm",
					Language:  "JavaScript",
				})
			}
		}
	}

	addDeps(pkg.Dependencies)
	addDeps(pkg.DevDependencies)

	return deps
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
				// Skip indirect dependencies
				if strings.Contains(trimmed, "// indirect") {
					continue
				}
				deps = append(deps, Dependency{
					Name:      matches[1],
					Version:   matches[2],
					Ecosystem: "go",
					Language:  "Go",
				})
			}
		} else {
			// Handle single-line require: require github.com/pkg/errors v0.9.1
			matches := singleRe.FindStringSubmatch(trimmed)
			if len(matches) >= 3 {
				if !strings.Contains(trimmed, "// indirect") {
					deps = append(deps, Dependency{
						Name:      matches[1],
						Version:   matches[2],
						Ecosystem: "go",
						Language:  "Go",
					})
				}
			}
		}
	}
	return deps
}

// parsePHP extracts dependencies from composer.json.
func parsePHP(root string) []Dependency {
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
				deps = append(deps, Dependency{
					Name:      name,
					Version:   cleanVersion(version),
					Ecosystem: "packagist",
					Language:  "PHP",
				})
			}
		}
	}

	addDeps(composer.Require)
	addDeps(composer.RequireDev)

	return deps
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

	var deps []Dependency
	content := string(data)

	// Simple regex extraction of dependencies from pom.xml
	re := regexp.MustCompile(`<dependency>\s*<groupId>([^<]+)</groupId>\s*<artifactId>([^<]+)</artifactId>\s*(?:<version>([^<]+)</version>)?`)
	matches := re.FindAllStringSubmatch(content, -1)

	for _, m := range matches {
		name := m[1] + ":" + m[2]
		version := ""
		if len(m) >= 4 {
			version = m[3]
		}
		deps = append(deps, Dependency{
			Name:      name,
			Version:   cleanVersion(version),
			Ecosystem: "maven",
			Language:  "Java",
		})
	}
	return deps
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
