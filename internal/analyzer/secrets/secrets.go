package secrets

import (
	"bufio"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Finding represents a single detected secret with its location.
type Finding struct {
	Path     string // file path (for terminal display only, never in JSON)
	RelPath  string // relative path
	Line     int    // line number
	Name     string // pattern name (e.g., "AWS Access Key")
	Category string // category (e.g., "aws", "api_key", "private_key")
}

// Result holds the secrets detection results.
type Result struct {
	SecretsCount int
	FileCount    int // number of files with secrets
	ByCategory   map[string]int
	Findings     []Finding
}

// SecretPattern defines a regex-based secret detection rule.
type SecretPattern struct {
	Name     string
	Category string
	Pattern  *regexp.Regexp
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
	{Name: "Bearer Token", Category: "token", Pattern: regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-.]{20,}`)},
}

// testFilePatterns are file patterns that indicate test/fixture/example files.
var testFilePatterns = []string{
	"_test.", ".test.", ".spec.", "__tests__",
	"test_", "test/", "tests/", "spec/", "fixtures/", "testdata/",
	"mock", "fake", "stub",
	"examples/", "example/",
	"readme",
	"docs/", "doc/",
	"script/", "scripts/", "seed",
}

// skipFilePatterns are file patterns that should be excluded from secrets scanning
// because they contain human-readable content, not code with real secrets.
var skipFilePatterns = []string{
	"/locales/", "/locale/", "/i18n/", "/translations/",
	".min.js", ".min.css", ".bundle.js",
	"/vendor/", "/node_modules/",
}

// allowlistPatterns are patterns that indicate false positives.
var allowlistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)example|placeholder|dummy|sample|your[_-]?`),
	regexp.MustCompile(`(?i)TODO|FIXME|CHANGEME|REPLACE`),
	regexp.MustCompile(`(?i)xxx+|yyy+|zzz+|aaa+`),
	// Template variables / env references are not real secrets
	regexp.MustCompile(`\{\{.*\.Env\..*\}\}`),           // Go template env: {{ .Env.TOKEN }}
	regexp.MustCompile(`\$\{[:{]?[A-Z_]+\}`),             // Shell/template env: ${SECRET_KEY}, ${:LARAVEL_LOAD_PASSWORD}
	regexp.MustCompile(`\$\{\{.*\}\}`),                  // GitHub Actions / template: ${{ secrets.TOKEN }}
	regexp.MustCompile(`\bos\.environ\b|\bos\.getenv\b`), // Python env lookups
	regexp.MustCompile(`\bprocess\.env\b`),               // Node.js env lookups
	regexp.MustCompile(`\$\{[a-zA-Z]`),                   // Template interpolation: ${var}, ${page.type}
	regexp.MustCompile(`\{\$[a-zA-Z]`),                   // PHP interpolation: {$attribute}
	// Regex/pattern definition lines — all Tier 1 languages
	regexp.MustCompile(`(?i)regexp\.MustCompile|regexp\.Compile`), // Go
	regexp.MustCompile(`(?i)\bre\.compile\b`),                     // Python
	regexp.MustCompile(`(?i)\bnew RegExp\b`),                      // JavaScript/TypeScript
	regexp.MustCompile(`(?i)Pattern\.compile`),                    // Java
	regexp.MustCompile(`(?i)\bpreg_match|preg_replace`),           // PHP
	regexp.MustCompile(`(?i)\bRegexp\.new\b`),                     // Ruby
}

// Scan scans files for hardcoded secrets.
func Scan(files []walker.FileInfo) *Result {
	r := &Result{
		ByCategory: make(map[string]int),
	}

	for _, f := range files {
		if isTestOrFixture(f.Path) || isSkippedFile(f.Path) {
			continue
		}

		findings := scanFile(f.Path, f.RelPath)
		if len(findings) > 0 {
			r.FileCount++
			for _, finding := range findings {
				r.SecretsCount++
				r.ByCategory[finding.Category]++
				r.Findings = append(r.Findings, finding)
			}
		}
	}

	return r
}

// scanFile returns the findings for secrets detected in the file.
func scanFile(path, relPath string) []Finding {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var findings []Finding
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		// Skip commented-out lines
		if isCommentLine(line) {
			continue
		}

		patternMatched := false
		for _, p := range patterns {
			if p.Pattern.MatchString(line) {
				if isAllowlisted(line) {
					continue
				}
				// For generic secrets, filter out false positives
			if p.Category == "generic" {
				if m := genericSecretValuePattern.FindStringSubmatch(line); len(m) >= 3 {
					val := m[2]
					if isGenericSecretFalsePositive(val) {
						continue
					}
				}
			}
				findings = append(findings, Finding{
					Path:     path,
					RelPath:  relPath,
					Line:     lineNum,
					Name:     p.Name,
					Category: p.Category,
				})
				patternMatched = true
				break // one secret per line max
			}
		}

		// Shannon entropy check for high-entropy strings not caught by patterns
		if !patternMatched {
			if hasHighEntropySecret(line) {
				findings = append(findings, Finding{
					Path:     path,
					RelPath:  relPath,
					Line:     lineNum,
					Name:     "High-Entropy Secret",
					Category: "entropy",
				})
			}
		}
	}
	// Check for scanner errors (e.g., lines exceeding buffer limit)
	if err := scanner.Err(); err != nil {
		// Partial results are still returned
	}

	return findings
}

// genericSecretValuePattern extracts the value from a generic secret assignment.
var genericSecretValuePattern = regexp.MustCompile(`(?i)(api_key|api_secret|apikey|secret_key|secret_token|auth_token|access_token|access_key|private_key|encryption_key|signing_key|master_key|service_key|client_secret|app_secret|password|passwd|pwd)\s*[=:]\s*["']([^"']{8,})["']`)

// screamingCaseIdentifier matches SCREAMING_CASE constant names like HTTP_API_KEY.
var screamingCaseIdentifier = regexp.MustCompile(`^[A-Z][A-Z0-9]*(_[A-Z0-9]+)+$`)

// commonPlaceholderPattern matches well-known dummy/placeholder password values.
var commonPlaceholderPattern = regexp.MustCompile(`(?i)^(password\d*|changeme|pa\$\$word|secret\d*|12345678\d*)$`)

// assignPattern matches assignment patterns with potential secret values.
// Requires a qualifying prefix (api_, secret_, auth_, etc.) to avoid false positives
// on common variable names like key = "cache_key_name".
var assignPattern = regexp.MustCompile(`(?i)(api_key|api_secret|apikey|secret_key|secret_token|auth_token|access_token|access_key|private_key|encryption_key|signing_key|master_key|service_key|client_secret|app_secret|password|passwd|pwd)\s*[=:]\s*["']([^"']{16,})["']`)

// isGenericSecretFalsePositive consolidates all checks for generic secret values
// that are clearly not real secrets.
func isGenericSecretFalsePositive(val string) bool {
	// Natural language phrases (spaces)
	if isNaturalLanguage(val) {
		return true
	}
	// Variable references ($var, ->, ..)
	if isVariableReference(val) {
		return true
	}
	// SCREAMING_CASE identifiers (HTTP_API_KEY, DISCOURSE_API, etc.)
	if screamingCaseIdentifier.MatchString(val) {
		return true
	}
	// Snake_case or dotted identifiers (config keys, i18n keys)
	if snakeCaseIdentifier.MatchString(val) {
		return true
	}
	if dottedIdentifier.MatchString(val) {
		return true
	}
	// String interpolation (Ruby #{}, JS ${}, PHP {$})
	if strings.Contains(val, "#{") || strings.Contains(val, "${") || strings.Contains(val, "{$") {
		return true
	}
	// URL paths or URIs
	if strings.Contains(val, "/") || strings.Contains(val, "://") {
		return true
	}
	// Common placeholder passwords (exact match only)
	if commonPlaceholderPattern.MatchString(val) {
		return true
	}
	return false
}

// isNaturalLanguage returns true if the value looks like a human-readable phrase
// rather than a real secret. Real secrets (API keys, tokens, passwords) don't
// contain spaces; values like "keyboard cat" or "shhhh, very secret" do.
func isNaturalLanguage(s string) bool {
	return strings.Contains(s, " ")
}

// isVariableReference returns true if the value contains variable dereferences
// rather than a hardcoded secret. Covers PHP ($var, $obj->prop, $arr['key']),
// Python/Ruby (self.x, obj.x), JS/TS (obj.prop), and string concatenation.
func isVariableReference(s string) bool {
	// PHP/Shell variable: $something
	if strings.Contains(s, "$") {
		return true
	}
	// Object property access or string concat operators
	if strings.Contains(s, "->") || strings.Contains(s, "..") {
		return true
	}
	return false
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

// snakeCaseIdentifier matches values that are purely lowercase letters,
// digits, underscores, and hyphens — feature flag names, config keys, etc.
// These are never real secrets even if entropy is borderline high.
var snakeCaseIdentifier = regexp.MustCompile(`^[a-z][a-z0-9]*([_.-][a-z0-9]+)+$`)

// dottedIdentifier matches dotted path identifiers like i18n keys
// (e.g. "user_notifications.digest.new_topics") or Ruby/Java namespaces.
var dottedIdentifier = regexp.MustCompile(`^[a-zA-Z][a-zA-Z0-9_]*(\.[a-zA-Z][a-zA-Z0-9_]*)+$`)

// hasHighEntropySecret checks for high-entropy strings that look like secrets.
func hasHighEntropySecret(line string) bool {
	// Look for assignment patterns with high-entropy values
	matches := assignPattern.FindStringSubmatch(line)
	if len(matches) < 3 {
		return false
	}

	value := matches[2]
	if isAllowlisted(value) {
		return false
	}

	// Snake_case, kebab-case, or dot-separated identifiers are config/i18n keys, not secrets.
	if snakeCaseIdentifier.MatchString(value) {
		return false
	}
	if dottedIdentifier.MatchString(value) {
		return false
	}

	// String interpolation (Ruby #{}, Python f-string, etc.) indicates computed values, not secrets.
	if strings.Contains(value, "#{") || strings.Contains(value, "${") || strings.Contains(value, "{$") {
		return false
	}

	// Values containing path separators or URL patterns are not secrets.
	if strings.Contains(value, "/") || strings.Contains(value, "://") {
		return false
	}

	entropy := shannonEntropy(value)
	// Hex strings: threshold 4.5, base64: threshold 4.0
	if isHexString(value) {
		return entropy > 4.5
	}
	return entropy > 4.0
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

func isAllowlisted(s string) bool {
	for _, p := range allowlistPatterns {
		if p.MatchString(s) {
			return true
		}
	}
	return false
}

func isTestOrFixture(path string) bool {
	pathLower := strings.ToLower(path)
	for _, p := range testFilePatterns {
		if strings.Contains(pathLower, p) {
			return true
		}
	}
	return false
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
