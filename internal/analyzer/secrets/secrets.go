package secrets

import (
	"bufio"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Signal represents a confidence adjustment applied to a finding.
type Signal struct {
	Name  string
	Delta int
}

// Finding represents a single detected secret with its location and confidence.
type Finding struct {
	Path       string   // file path (for terminal display only, never in JSON)
	RelPath    string   // relative path
	Line       int      // line number
	Name       string   // pattern name (e.g., "AWS Access Key")
	Category   string   // category (e.g., "aws", "api_key", "private_key")
	Confidence int      // 0-100 final confidence score
	Signals    []Signal // signals that adjusted the confidence
}

// Result holds the secrets detection results.
type Result struct {
	SecretsCount    int
	SuppressedCount int // findings with confidence 30-49 (not reported)
	FileCount       int // number of files with reported secrets
	ByCategory      map[string]int
	Findings        []Finding
}

// SecretPattern defines a regex-based secret detection rule.
type SecretPattern struct {
	Name     string
	Category string
	Pattern  *regexp.Regexp
}

// confidenceThreshold is the minimum confidence score for a finding to be reported.
const confidenceThreshold = 50

// baseConfidence maps secret categories to their starting confidence score.
// Specific patterns (AWS, GCP, etc.) start high because the format itself is strong evidence.
// Generic patterns start low because they rely on heuristic matching.
var baseConfidence = map[string]int{
	"aws":               85,
	"gcp":               85,
	"azure":             85,
	"api_key":           85,
	"token":             80,
	"bearer":            75,
	"private_key":       70,
	"connection_string": 75,
	"generic":           50,
	"entropy":           40,
}

// patterns is the list of secret detection patterns.
var patterns = []SecretPattern{
	// AWS
	{Name: "AWS Access Key", Category: "aws", Pattern: regexp.MustCompile(`AKIA[0-9A-Z]{16}`)},
	{Name: "AWS Secret Key", Category: "aws", Pattern: regexp.MustCompile(`(?i)aws_secret_access_key\s*[=:]\s*["']?[A-Za-z0-9/+=]{40}`)},

	// GCP
	{Name: "GCP API Key", Category: "gcp", Pattern: regexp.MustCompile(`AIza[0-9A-Za-z_-]{35}`)},
	{Name: "GCP Service Account", Category: "gcp", Pattern: regexp.MustCompile(`"type"\s*:\s*"service_account"`)},

	// Azure
	{Name: "Azure Storage Key", Category: "azure", Pattern: regexp.MustCompile(`(?i)AccountKey\s*=\s*[A-Za-z0-9/+=]{86}==`)},

	// API Keys
	{Name: "OpenAI API Key", Category: "api_key", Pattern: regexp.MustCompile(`sk-[A-Za-z0-9]{48}`)},
	{Name: "Anthropic API Key", Category: "api_key", Pattern: regexp.MustCompile(`sk-ant-[A-Za-z0-9_-]{90,}`)},
	{Name: "Stripe Secret Key", Category: "api_key", Pattern: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`)},
	{Name: "Stripe Publishable Key", Category: "api_key", Pattern: regexp.MustCompile(`pk_live_[0-9a-zA-Z]{24,}`)},
	{Name: "SendGrid API Key", Category: "api_key", Pattern: regexp.MustCompile(`SG\.[0-9A-Za-z_-]{22}\.[0-9A-Za-z_-]{43}`)},
	{Name: "Twilio Auth Token", Category: "api_key", Pattern: regexp.MustCompile(`(?i)twilio.*['\"][0-9a-f]{32}['\"]`)},
	{Name: "Slack Token", Category: "api_key", Pattern: regexp.MustCompile(`xox[bpors]-[0-9a-zA-Z]{10,}`)},
	{Name: "Slack Webhook", Category: "api_key", Pattern: regexp.MustCompile(`https://hooks\.slack\.com/services/T[0-9A-Z]{8,}/B[0-9A-Z]{8,}/[0-9a-zA-Z]{24}`)},
	{Name: "Mailchimp API Key", Category: "api_key", Pattern: regexp.MustCompile(`[0-9a-f]{32}-us[0-9]{1,2}`)},
	{Name: "Mailgun API Key", Category: "api_key", Pattern: regexp.MustCompile(`key-[0-9a-zA-Z]{32}`)},
	{Name: "Square Access Token", Category: "api_key", Pattern: regexp.MustCompile(`sq0atp-[0-9A-Za-z_-]{22}`)},
	{Name: "Square OAuth Secret", Category: "api_key", Pattern: regexp.MustCompile(`sq0csp-[0-9A-Za-z_-]{43}`)},
	{Name: "Shopify Access Token", Category: "api_key", Pattern: regexp.MustCompile(`shpat_[0-9a-fA-F]{32}`)},
	{Name: "Shopify Shared Secret", Category: "api_key", Pattern: regexp.MustCompile(`shpss_[0-9a-fA-F]{32}`)},
	{Name: "Datadog API Key", Category: "api_key", Pattern: regexp.MustCompile(`(?i)dd[-_]?api[-_]?key\s*[=:]\s*["']?[0-9a-f]{32}`)},

	// Tokens
	{Name: "GitHub PAT", Category: "token", Pattern: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`)},
	{Name: "GitHub OAuth", Category: "token", Pattern: regexp.MustCompile(`gho_[0-9a-zA-Z]{36}`)},
	{Name: "GitHub App Token", Category: "token", Pattern: regexp.MustCompile(`(ghu|ghs)_[0-9a-zA-Z]{36}`)},
	{Name: "GitLab PAT", Category: "token", Pattern: regexp.MustCompile(`glpat-[0-9a-zA-Z_-]{20}`)},
	{Name: "npm Token", Category: "token", Pattern: regexp.MustCompile(`npm_[0-9a-zA-Z]{36}`)},
	{Name: "PyPI Token", Category: "token", Pattern: regexp.MustCompile(`pypi-[0-9a-zA-Z_-]{50,}`)},
	{Name: "NuGet API Key", Category: "token", Pattern: regexp.MustCompile(`oy2[0-9a-z]{43}`)},
	{Name: "Heroku API Key", Category: "token", Pattern: regexp.MustCompile(`(?i)heroku.*['\"][0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}['\"]`)},
	{Name: "Firebase Token", Category: "token", Pattern: regexp.MustCompile(`(?i)firebase\s*[=:]\s*["'][A-Za-z0-9_-]{30,}`)},
	{Name: "JWT Token", Category: "token", Pattern: regexp.MustCompile(`eyJ[A-Za-z0-9_-]{10,}\.eyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}`)},

	// Private Keys
	{Name: "RSA Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`)},
	{Name: "DSA Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN DSA PRIVATE KEY-----`)},
	{Name: "EC Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`)},
	{Name: "Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN PRIVATE KEY-----`)},
	{Name: "OpenSSH Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN OPENSSH PRIVATE KEY-----`)},
	{Name: "PGP Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN PGP PRIVATE KEY BLOCK-----`)},

	// Connection Strings
	{Name: "Database URL", Category: "connection_string", Pattern: regexp.MustCompile(`(?i)(mysql|postgres|postgresql|mongodb|redis)://[^\s"']+:[^\s"']+@[^\s"']+`)},
	{Name: "AMQP URL", Category: "connection_string", Pattern: regexp.MustCompile(`(?i)amqps?://[^\s"']+:[^\s"']+@[^\s"']+`)},
	{Name: "SMTP Credentials", Category: "connection_string", Pattern: regexp.MustCompile(`(?i)smtp://[^\s"']+:[^\s"']+@[^\s"']+`)},

	// Generic patterns — require qualifying prefix to reduce false positives on
	// common variable names. Bare "key", "token", "secret" are too broad.
	{Name: "Generic Secret", Category: "generic", Pattern: regexp.MustCompile(`(?i)(api_key|api_secret|apikey|secret_key|secret_token|auth_token|access_token|access_key|private_key|encryption_key|signing_key|master_key|service_key|client_secret|app_secret|password|passwd|pwd)\s*[=:]\s*["'][^"']{8,}["']`)},
	{Name: "Bearer Token", Category: "bearer", Pattern: regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-.]{20,}`)},
}

// testPathPatterns are file patterns for test/fixture/mock files.
// These produce a -30 signal — test code might accidentally contain real secrets.
var testPathPatterns = []string{
	"_test.", ".test.", ".spec.", "__tests__",
	"test_", "test/", "tests/", "spec/", "fixtures/", "testdata/", "test-data/",
	"mock", "fake", "stub",
	"seed",
	"devenv/",
}

// docsPathPatterns are file patterns for documentation/example/script files.
// These produce a -40 signal — content is purely illustrative, never production secrets.
var docsPathPatterns = []string{
	"examples/", "example/",
	"readme",
	"docs/", "doc/",
	"script/", "scripts/",
}

// skipFilePatterns are file patterns that should be hard-skipped from scanning.
// These are excluded for performance (not accuracy) — vendor/minified/locale
// files are never useful to scan.
var skipFilePatterns = []string{
	"/locales/", "/locale/", "/i18n/", "/translations/",
	".min.js", ".min.css", ".bundle.js",
	"/vendor/", "/node_modules/",
}

// --- Signal pattern groups ---

// placeholderLinePattern matches placeholder/example/TODO markers in a line.
var placeholderLinePattern = regexp.MustCompile(`(?i)(example|placeholder|dummy|sample|your[_-]|TODO|FIXME|CHANGEME|REPLACE|x{3,}|y{3,}|z{3,}|a{3,})`)

// regexDefPatterns match regex/pattern definition lines in all Tier 1 languages.
var regexDefPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)regexp\.MustCompile|regexp\.Compile`), // Go
	regexp.MustCompile(`(?i)\bre\.compile\b`),                     // Python
	regexp.MustCompile(`(?i)\bnew RegExp\b`),                      // JavaScript/TypeScript
	regexp.MustCompile(`(?i)Pattern\.compile`),                    // Java
	regexp.MustCompile(`(?i)\bpreg_match|preg_replace`),           // PHP
	regexp.MustCompile(`(?i)\bRegexp\.new\b`),                     // Ruby
}

// envLookupPatterns match environment variable lookup expressions.
var envLookupPatterns = []*regexp.Regexp{
	regexp.MustCompile(`\bos\.environ\b|\bos\.getenv\b`), // Python
	regexp.MustCompile(`\bprocess\.env\b`),                // Node.js
}

// templateLinePatterns match template/interpolation syntax on a line.
var templateLinePatterns = []*regexp.Regexp{
	regexp.MustCompile(`\{\{.*\.Env\..*\}\}`),   // Go template env: {{ .Env.TOKEN }}
	regexp.MustCompile(`\$\{[:{]?[A-Z_]+\}`),    // Shell/template env: ${SECRET_KEY}
	regexp.MustCompile(`\$\{\{.*\}\}`),           // GitHub Actions: ${{ secrets.TOKEN }}
	regexp.MustCompile(`\$\{[a-zA-Z]`),           // Template interpolation: ${var}
	regexp.MustCompile(`\{\$[a-zA-Z]`),           // PHP interpolation: {$attribute}
	regexp.MustCompile(`%\([a-zA-Z_]+\)s`),       // Python old-style format: %(password)s
	regexp.MustCompile(`\{\{.+\}\}`),             // Mustache/Jinja2/Django: {{ variable }}
	regexp.MustCompile(`%[sdvfwqtx]`),            // Printf-style format: %s, %d, %v
	regexp.MustCompile(`\\\([a-zA-Z]`),           // Swift interpolation: \(variable)
}

// --- Value-level patterns ---

var genericSecretValuePattern = regexp.MustCompile(`(?i)(api_key|api_secret|apikey|secret_key|secret_token|auth_token|access_token|access_key|private_key|encryption_key|signing_key|master_key|service_key|client_secret|app_secret|password|passwd|pwd)\s*[=:]\s*["']([^"']{8,})["']`)

var assignPattern = regexp.MustCompile(`(?i)(api_key|api_secret|apikey|secret_key|secret_token|auth_token|access_token|access_key|private_key|encryption_key|signing_key|master_key|service_key|client_secret|app_secret|password|passwd|pwd)\s*[=:]\s*["']([^"']{16,})["']`)

var screamingCaseIdentifier = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$`)
var snakeCaseIdentifier = regexp.MustCompile(`^[a-z][a-z0-9]*([_.-][a-z0-9]+)+$`)
var dottedIdentifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)
var commonPlaceholderPattern = regexp.MustCompile(`(?i)^(password\d*|changeme|pa\$\$word|secret\d*|12345678\d*)$`)
var formatBracePattern = regexp.MustCompile(`\{[a-zA-Z_][a-zA-Z0-9_]*\}`)
var printfFormatPattern = regexp.MustCompile(`%[sdvfwqtxboegp]`)

// Scan scans files for hardcoded secrets using confidence-based scoring.
// Each finding gets a 0-100 confidence score from multiple signals.
// Only findings with confidence >= 50 are reported.
func Scan(files []walker.FileInfo) *Result {
	r := &Result{
		ByCategory: make(map[string]int),
	}

	for _, f := range files {
		// Hard pre-filter: vendor, minified, locale (performance, not accuracy)
		if isSkippedFile(f.Path) {
			continue
		}

		findings := scanFile(f.Path, f.RelPath)

		// Context signal: files with many findings are likely template/config
		if len(findings) > 10 {
			for i := range findings {
				findings[i].Signals = append(findings[i].Signals, Signal{"high_finding_count", -15})
				findings[i].Confidence = clamp(findings[i].Confidence - 15)
			}
		}

		fileHasReported := false
		for _, finding := range findings {
			if finding.Confidence >= confidenceThreshold {
				r.SecretsCount++
				r.ByCategory[finding.Category]++
				r.Findings = append(r.Findings, finding)
				fileHasReported = true
			} else if finding.Confidence >= 30 {
				r.SuppressedCount++
			}
			// < 30: silently dropped
		}
		if fileHasReported {
			r.FileCount++
		}
	}

	return r
}

// scanFile returns all candidate findings for a file with confidence scores.
func scanFile(path, relPath string) []Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	// Compute file-level signals once
	fileSignals := computeFileSignals(path)
	fileDelta := sumDeltas(fileSignals)

	var findings []Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Compute line-level signals
		lineSignals := computeLineSignals(line)
		lineDelta := sumDeltas(lineSignals)

		patternMatched := false
		for _, p := range patterns {
			if p.Pattern.MatchString(line) {
				base := baseConfidence[p.Category]
				confidence := base + fileDelta + lineDelta
				signals := combineSignals(fileSignals, lineSignals)

				// For generic secrets, apply value-level signals
				if p.Category == "generic" {
					if m := genericSecretValuePattern.FindStringSubmatch(line); len(m) >= 3 {
						valSignals := computeValueSignals(m[2])
						signals = append(signals, valSignals...)
						confidence += sumDeltas(valSignals)
					}
				}

				findings = append(findings, Finding{
					Path:       path,
					RelPath:    relPath,
					Line:       lineNum,
					Name:       p.Name,
					Category:   p.Category,
					Confidence: clamp(confidence),
					Signals:    signals,
				})
				patternMatched = true
				break // one finding per line max
			}
		}

		// Shannon entropy check for high-entropy strings not caught by patterns
		if !patternMatched {
			if val, ok := extractHighEntropyValue(line); ok {
				base := baseConfidence["entropy"]
				confidence := base + fileDelta + lineDelta
				signals := combineSignals(fileSignals, lineSignals)

				valSignals := computeValueSignals(val)
				signals = append(signals, valSignals...)
				confidence += sumDeltas(valSignals)

				findings = append(findings, Finding{
					Path:       path,
					RelPath:    relPath,
					Line:       lineNum,
					Name:       "High-Entropy Secret",
					Category:   "entropy",
					Confidence: clamp(confidence),
					Signals:    signals,
				})
			}
		}
	}
	// Partial results are returned on scanner error
	_ = scanner.Err()

	return findings
}

// --- Signal computation ---

// computeFileSignals returns signals based on the file path.
func computeFileSignals(path string) []Signal {
	var signals []Signal
	pathLower := strings.ToLower(path)

	// Test/fixture/mock paths (-30)
	for _, p := range testPathPatterns {
		if strings.Contains(pathLower, p) {
			signals = append(signals, Signal{"test_path", -30})
			break
		}
	}

	// Documentation/example/script paths (-40) — purely illustrative content
	for _, p := range docsPathPatterns {
		if strings.Contains(pathLower, p) {
			signals = append(signals, Signal{"docs_path", -40})
			break
		}
	}

	// Template config files (.env.example, config.sample, etc.)
	base := filepath.Base(pathLower)
	if strings.Contains(base, ".example") || strings.Contains(base, ".sample") || strings.Contains(base, ".template") {
		signals = append(signals, Signal{"config_template", -35})
	}

	return signals
}

// computeLineSignals returns signals based on the content of a single line.
func computeLineSignals(line string) []Signal {
	var signals []Signal

	// Comment line
	if isCommentLine(line) {
		signals = append(signals, Signal{"comment", -25})
	}

	// Placeholder/TODO/example markers
	if placeholderLinePattern.MatchString(line) {
		signals = append(signals, Signal{"placeholder", -40})
	}

	// Regex pattern definition
	for _, p := range regexDefPatterns {
		if p.MatchString(line) {
			signals = append(signals, Signal{"regex_def", -50})
			break
		}
	}

	// Environment variable lookup
	for _, p := range envLookupPatterns {
		if p.MatchString(line) {
			signals = append(signals, Signal{"env_lookup", -40})
			break
		}
	}

	// Template/interpolation syntax
	for _, p := range templateLinePatterns {
		if p.MatchString(line) {
			signals = append(signals, Signal{"template", -35})
			break
		}
	}

	return signals
}

// computeValueSignals returns signals based on the extracted secret value.
func computeValueSignals(val string) []Signal {
	var signals []Signal

	// Natural language (spaces indicate human-readable text, not a secret)
	if strings.Contains(val, " ") {
		signals = append(signals, Signal{"natural_language", -30})
	}

	// Variable references ($var, ->, ..)
	if strings.Contains(val, "$") || strings.Contains(val, "->") || strings.Contains(val, "..") {
		signals = append(signals, Signal{"variable_ref", -35})
	}

	// Identifier patterns (SCREAMING_CASE, snake_case, dotted)
	if screamingCaseIdentifier.MatchString(val) || snakeCaseIdentifier.MatchString(val) || dottedIdentifier.MatchString(val) {
		signals = append(signals, Signal{"identifier", -30})
	}

	// String interpolation in value (Ruby #{}, JS ${}, PHP {$}, Python %(x)s, Swift \(x), etc.)
	if strings.Contains(val, "#{") || strings.Contains(val, "${") || strings.Contains(val, "{$") ||
		strings.Contains(val, "%(") || strings.Contains(val, "\\(") ||
		formatBracePattern.MatchString(val) || strings.Contains(val, "{{") ||
		printfFormatPattern.MatchString(val) {
		signals = append(signals, Signal{"interpolation", -35})
	}

	// URL paths or URIs
	if strings.Contains(val, "/") || strings.Contains(val, "://") {
		signals = append(signals, Signal{"url_path", -25})
	}

	// Common placeholder passwords
	if commonPlaceholderPattern.MatchString(val) {
		signals = append(signals, Signal{"placeholder_value", -45})
	}

	return signals
}

// --- Entropy detection ---

// extractHighEntropyValue extracts a value from an assignment pattern and checks
// if it has high Shannon entropy. Returns the value and true if it qualifies.
func extractHighEntropyValue(line string) (string, bool) {
	matches := assignPattern.FindStringSubmatch(line)
	if len(matches) < 3 {
		return "", false
	}

	value := matches[2]
	entropy := shannonEntropy(value)

	if isHexString(value) {
		if entropy <= 4.5 {
			return "", false
		}
	} else {
		if entropy <= 4.0 {
			return "", false
		}
	}

	return value, true
}

// shannonEntropy computes Shannon entropy of a string.
func shannonEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	freq := make(map[rune]int)
	runeCount := 0
	for _, c := range s {
		freq[c]++
		runeCount++
	}

	entropy := 0.0
	length := float64(runeCount)
	for _, count := range freq {
		p := float64(count) / length
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func isHexString(s string) bool {
	for _, c := range s {
		if !((c >= '0' && c <= '9') || (c >= 'a' && c <= 'f') || (c >= 'A' && c <= 'F')) {
			return false
		}
	}
	return true
}

// --- Helpers ---

func sumDeltas(signals []Signal) int {
	total := 0
	for _, s := range signals {
		total += s.Delta
	}
	return total
}

func combineSignals(a, b []Signal) []Signal {
	combined := make([]Signal, 0, len(a)+len(b))
	combined = append(combined, a...)
	combined = append(combined, b...)
	return combined
}

func clamp(n int) int {
	if n < 0 {
		return 0
	}
	if n > 100 {
		return 100
	}
	return n
}

// isCommentLine returns true if the line is a comment (after trimming whitespace).
func isCommentLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "<!--")
}

func isSkippedFile(path string) bool {
	pathLower := strings.ToLower(path)
	for _, p := range skipFilePatterns {
		if strings.Contains(pathLower, p) {
			return true
		}
	}
	return false
}
