package secrets

import (
	"bufio"
	"math"
	"os"
	"regexp"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Result holds the secrets detection results.
type Result struct {
	SecretsCount    int
	FileCount       int // number of files with secrets
	ByCategory      map[string]int
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
	{Name: "Stripe Secret Key", Category: "api_key", Pattern: regexp.MustCompile(`sk_live_[0-9a-zA-Z]{24,}`)},
	{Name: "Stripe Publishable Key", Category: "api_key", Pattern: regexp.MustCompile(`pk_live_[0-9a-zA-Z]{24,}`)},
	{Name: "SendGrid API Key", Category: "api_key", Pattern: regexp.MustCompile(`SG\.[0-9A-Za-z_-]{22}\.[0-9A-Za-z_-]{43}`)},
	{Name: "Twilio Auth Token", Category: "api_key", Pattern: regexp.MustCompile(`(?i)twilio.*['\"][0-9a-f]{32}['\"]`)},
	{Name: "Slack Token", Category: "api_key", Pattern: regexp.MustCompile(`xox[bpors]-[0-9a-zA-Z]{10,}`)},
	{Name: "Slack Webhook", Category: "api_key", Pattern: regexp.MustCompile(`https://hooks\.slack\.com/services/T[0-9A-Z]{8,}/B[0-9A-Z]{8,}/[0-9a-zA-Z]{24}`)},

	// Tokens
	{Name: "GitHub PAT", Category: "token", Pattern: regexp.MustCompile(`ghp_[0-9a-zA-Z]{36}`)},
	{Name: "GitHub OAuth", Category: "token", Pattern: regexp.MustCompile(`gho_[0-9a-zA-Z]{36}`)},
	{Name: "npm Token", Category: "token", Pattern: regexp.MustCompile(`npm_[0-9a-zA-Z]{36}`)},
	{Name: "PyPI Token", Category: "token", Pattern: regexp.MustCompile(`pypi-[0-9a-zA-Z_-]{50,}`)},

	// Private Keys
	{Name: "RSA Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN RSA PRIVATE KEY-----`)},
	{Name: "DSA Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN DSA PRIVATE KEY-----`)},
	{Name: "EC Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN EC PRIVATE KEY-----`)},
	{Name: "Private Key", Category: "private_key", Pattern: regexp.MustCompile(`-----BEGIN PRIVATE KEY-----`)},

	// Connection Strings
	{Name: "Database URL", Category: "connection_string", Pattern: regexp.MustCompile(`(?i)(mysql|postgres|postgresql|mongodb|redis)://[^\s"']+:[^\s"']+@[^\s"']+`)},

	// Generic patterns
	{Name: "Generic Secret", Category: "generic", Pattern: regexp.MustCompile(`(?i)(password|passwd|pwd|secret|api_key|apikey|api_secret|access_token|auth_token|private_key)\s*[=:]\s*["'][^"']{8,}["']`)},
	{Name: "Bearer Token", Category: "token", Pattern: regexp.MustCompile(`(?i)bearer\s+[a-zA-Z0-9_\-.]{20,}`)},
}

// testFilePatterns are file patterns that indicate test/fixture files.
var testFilePatterns = []string{
	"_test.", ".test.", ".spec.", "__tests__",
	"test_", "tests/", "fixtures/", "testdata/",
	"mock", "fake", "stub",
}

// allowlistPatterns are patterns that indicate false positives.
var allowlistPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)example|placeholder|dummy|sample|your[_-]?`),
	regexp.MustCompile(`(?i)TODO|FIXME|CHANGEME|REPLACE`),
	regexp.MustCompile(`(?i)xxx+|yyy+|zzz+|aaa+`),
}

// Scan scans files for hardcoded secrets.
func Scan(files []walker.FileInfo) *Result {
	r := &Result{
		ByCategory: make(map[string]int),
	}

	for _, f := range files {
		if isTestOrFixture(f.Path) {
			continue
		}

		categories := scanFile(f.Path)
		if len(categories) > 0 {
			r.FileCount++
			for _, cat := range categories {
				r.SecretsCount++
				r.ByCategory[cat]++
			}
		}
	}

	return r
}

// scanFile returns the categories of secrets found (one per detected secret).
func scanFile(path string) []string {
	f, err := os.Open(path)
	if err != nil {
		return nil
	}
	defer f.Close()

	var categories []string
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)

	lineNum := 0
	for scanner.Scan() {
		lineNum++
		line := scanner.Text()

		patternMatched := false
		for _, p := range patterns {
			if p.Pattern.MatchString(line) {
				if !isAllowlisted(line) {
					categories = append(categories, p.Category)
					patternMatched = true
					break // one secret per line max
				}
			}
		}

		// Shannon entropy check for high-entropy strings not caught by patterns
		if !patternMatched {
			if hasHighEntropySecret(line) {
				categories = append(categories, "entropy")
			}
		}
	}
	// Check for scanner errors (e.g., lines exceeding buffer limit)
	if err := scanner.Err(); err != nil {
		// Partial results are still returned
	}

	return categories
}

// assignPattern matches assignment patterns with potential secret values.
var assignPattern = regexp.MustCompile(`(?i)(key|token|secret|password|credential)\s*[=:]\s*["']([^"']{16,})["']`)

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
