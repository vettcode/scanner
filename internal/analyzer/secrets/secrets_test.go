package secrets

import (
	"math"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vettcode/scanner/internal/walker"
)

func writeFile(t *testing.T, dir, name, content string) string {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
	return path
}

func TestScan_AWSKey(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.go", `package config
const awsKey = "AKIAIOSFODNN7ABCDEFG"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	assert.Equal(t, 1, r.SecretsCount)
}

func TestScan_PrivateKey(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "key.pem", `-----BEGIN RSA PRIVATE KEY-----
MIIEpAIBAAKCAQEA0Z
-----END RSA PRIVATE KEY-----
`)
	files := []walker.FileInfo{{Path: path, RelPath: "key.pem"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1)
}

func TestScan_GenericSecret(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "app.py", `
password = "s3cr3t_p@ssw0rd_12345"
api_key = "abc123def456ghi789jkl"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "app.py"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1)
}

func TestScan_NoSecretsInCleanCode(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "main.go", `package main

import "fmt"

func main() {
	fmt.Println("hello world")
}
`)
	files := []walker.FileInfo{{Path: path, RelPath: "main.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount)
}

func TestScan_SkipsTestFiles(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config_test.go", `package config
const testKey = "AKIAIOSFODNN7EXAMPLE"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config_test.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount) // test files are skipped
}

func TestScan_AllowlistFilters(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.go", `package config
const apiKey = "your-api-key-here-placeholder"
const secret = "CHANGEME_this_is_example"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount) // filtered by allowlist
}

func TestScan_DatabaseURL(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "db.go", `package db
var connStr = "postgres://user:pass123@localhost:5432/mydb"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "db.go"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1)
}

func TestShannonEntropy(t *testing.T) {
	// Low entropy (repeated chars)
	assert.Less(t, shannonEntropy("aaaaaaaaaaaa"), 1.0)

	// High entropy (random-looking)
	high := shannonEntropy("aB3$kL9pQ2wX7mR5nY")
	assert.Greater(t, high, 3.5)

	// Empty
	assert.Equal(t, 0.0, shannonEntropy(""))
}

func TestIsHexString(t *testing.T) {
	assert.True(t, isHexString("abcdef0123456789"))
	assert.True(t, isHexString("ABCDEF"))
	assert.False(t, isHexString("xyz"))
	assert.False(t, isHexString("abc!def"))
}

func TestScan_EntropyDetectsHighEntropyAfterPatternMatch(t *testing.T) {
	dir := t.TempDir()
	// File has both a pattern-matched secret AND a high-entropy secret on different lines
	path := writeFile(t, dir, "mixed.go", `package config
const awsKey = "AKIAIOSFODNN7ABCDEFG"
const secret = "a9f8b7c6d5e4f3a2b1c0d9e8f7a6b5c4"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "mixed.go"}}
	r := Scan(files)
	// Should detect at least 2: the AWS key via pattern + the hex secret via entropy
	assert.GreaterOrEqual(t, r.SecretsCount, 2, "entropy detection should work independently of pattern matching")
}

func TestShannonEntropy_KnownValue(t *testing.T) {
	// "ab" has entropy = 1.0 (2 symbols, equal probability)
	e := shannonEntropy("ab")
	assert.InDelta(t, 1.0, e, 0.001)

	// "aaab" has lower entropy than "aabb"
	e1 := shannonEntropy("aaab")
	e2 := shannonEntropy("aabb")
	assert.Less(t, e1, e2)

	// Math check: "abcd" should have entropy = log2(4) = 2.0
	e3 := shannonEntropy("abcd")
	assert.InDelta(t, math.Log2(4), e3, 0.001)
}

// --- False positive tests (spec: zero false positives for legitimate high-entropy strings) ---

func TestScan_NoFalsePositives_UUIDs(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "ids.go", `package ids

// UUIDs are high-entropy but not secrets
var requestID = "550e8400-e29b-41d4-a716-446655440000"
var sessionID = "6ba7b810-9dad-11d1-80b4-00c04fd430c8"
var correlationID = "f47ac10b-58cc-4372-a567-0e02b2c3d479"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "ids.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "UUIDs should not be flagged as secrets")
}

func TestScan_NoFalsePositives_GitHashes(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "version.go", `package version

// Git commit hashes are high-entropy hex strings but not secrets
var commitSHA = "a1b2c3d4e5f6a7b8c9d0e1f2a3b4c5d6e7f8a9b0"
var shortSHA = "a1b2c3d"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "version.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "git hashes should not be flagged as secrets")
}

func TestScan_NoFalsePositives_CommonPatterns(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.go", `package config

// These look secret-like but are examples/placeholders
var apiKey = "your-api-key-here"
var secret = "CHANGEME"
var token = "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "placeholder/example values should be filtered by allowlist")
}

func TestScan_NoFalsePositives_SnakeCaseIdentifiers(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.go", `package config

// Snake_case and kebab-case identifiers are feature flags / config keys, not secrets
const legacyKey = "legacy_cloud_anthropic_web_search"
const featureToken = "enable_dark_mode_beta_v2"
const credentialName = "oauth-client-credential-grant"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "snake_case/kebab-case identifiers should not be flagged as high-entropy secrets")
}

func TestSnakeCaseIdentifier(t *testing.T) {
	// Should match: multi-word lowercase identifiers
	assert.True(t, snakeCaseIdentifier.MatchString("legacy_cloud_anthropic_web_search"))
	assert.True(t, snakeCaseIdentifier.MatchString("enable_dark_mode"))
	assert.True(t, snakeCaseIdentifier.MatchString("oauth-client-credential"))
	assert.True(t, snakeCaseIdentifier.MatchString("feature_flag_v2"))

	// Should NOT match: real secrets or single words
	assert.False(t, snakeCaseIdentifier.MatchString("aB3cD4eF5gH6iJ7k"))  // mixed case
	assert.False(t, snakeCaseIdentifier.MatchString("singleword"))          // no separator
	assert.False(t, snakeCaseIdentifier.MatchString("HAS_UPPER_CASE"))      // uppercase
	assert.False(t, snakeCaseIdentifier.MatchString("sk_live_abc123XYZ"))   // mixed case (real key)
	assert.False(t, snakeCaseIdentifier.MatchString(""))                    // empty
}

func TestScan_NoFalsePositives_CommentedLines(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "docker-compose.yml", `services:
  mssql:
    image: mcr.microsoft.com/mssql/server
#   environment:
#     SA_PASSWORD: "Forge123"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "docker-compose.yml"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "commented-out lines should not trigger secrets")
}

func TestScan_NoFalsePositives_VariableReferences(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "db.php", `<?php
$args = [
    'password' => '--password='.$connection['password'],
    'secret' => $config['api_secret'],
];
$value = ' --password="${:LARAVEL_LOAD_PASSWORD}"';
$env = ['PGPASSWORD' => $config['password']];
`)
	files := []walker.FileInfo{{Path: path, RelPath: "db.php"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "variable references should not trigger generic secret detection")
}

func TestScan_NoFalsePositives_TemplateInterpolation(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "template.blade.php", `<template x-for="(page, index) in pages" :key="` + "`page-${page.type}-${page.value}-${page.id || index}`" + `">
`)
	path2 := writeFile(t, dir, "messages.php", `<?php
$customKey = "validation.custom.{$attribute}.{$lowerRule}";
`)
	files := []walker.FileInfo{
		{Path: path, RelPath: "template.blade.php"},
		{Path: path2, RelPath: "messages.php"},
	}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "template interpolation should not trigger entropy detection")
}

func TestIsCommentLine(t *testing.T) {
	assert.True(t, isCommentLine("  // this is a comment"))
	assert.True(t, isCommentLine("# yaml comment"))
	assert.True(t, isCommentLine("  * javadoc line"))
	assert.True(t, isCommentLine("/* block comment start"))
	assert.True(t, isCommentLine("<!-- html comment"))
	assert.False(t, isCommentLine(`password = "hardcoded123"`))
	assert.False(t, isCommentLine(`  const key = "abc"`))
}

func TestIsVariableReference(t *testing.T) {
	assert.True(t, isVariableReference("$connection['password']"))
	assert.True(t, isVariableReference("$config['api_secret']"))
	assert.True(t, isVariableReference("--password=.$connection"))
	assert.True(t, isVariableReference("self->getPassword()"))
	assert.False(t, isVariableReference("realPassword!2024"))
	assert.False(t, isVariableReference("sk_live_abcdefg12345"))
}

func TestScan_GenericSecret_HardcodedPassword(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "db.go", `package db
var password = "realPassword!2024"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "db.go"}}
	r := Scan(files)
	// Hardcoded password with a real-looking value should be detected
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "hardcoded password should be detected")
}

// --- Multiple secrets in one file (spec: exact count) ---

func TestScan_MultipleSecrets_ExactCount(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "leaked.rb", `
# This file contains exactly 3 pattern-matched secrets
AWS_ACCESS_KEY = "AKIAIOSFODNN7ABCDEFG"
STRIPE_KEY = "sk_live_51H7abcdefghijklmnopqrstuvwxyz12"
GITHUB_TOKEN = "ghp_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefgh12"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "leaked.rb"}}
	r := Scan(files)
	assert.Equal(t, 3, r.SecretsCount, "should detect exactly 3 pattern-matched secrets (AWS key, Stripe key, GitHub PAT)")
}

func TestScan_ByCategory_Populated(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "multi.py", `
AWS_KEY = "AKIAIOSFODNN7EXAMPLE"
DB_URL = "postgres://admin:secretpass@prod.db.com:5432/app"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "multi.py"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1)
	// ByCategory should be populated when secrets are found
	if r.SecretsCount > 0 {
		totalByCategory := 0
		for _, count := range r.ByCategory {
			totalByCategory += count
		}
		assert.Equal(t, r.SecretsCount, totalByCategory, "ByCategory counts should sum to SecretsCount")
	}
}

// --- Edge cases ---

func TestScan_EmptyFiles(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "empty.go", "")
	files := []walker.FileInfo{{Path: path, RelPath: "empty.go"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount)
}

func TestScan_NoFiles(t *testing.T) {
	r := Scan(nil)
	assert.Equal(t, 0, r.SecretsCount)
}

func TestScan_BinaryLikeContent(t *testing.T) {
	dir := t.TempDir()
	// File with binary-like content should not crash
	content := make([]byte, 256)
	for i := range content {
		content[i] = byte(i)
	}
	path := filepath.Join(dir, "binary.dat")
	require.NoError(t, os.WriteFile(path, content, 0644))
	files := []walker.FileInfo{{Path: path, RelPath: "binary.dat"}}
	r := Scan(files)
	// Should not panic or crash, count is whatever the scanner finds
	assert.GreaterOrEqual(t, r.SecretsCount, 0)
}

// TestScan_SkipsVariousTestFiles verifies that the secrets scanner skips test files.
// Note: The scanner uses path-based pattern matching via isTestOrFixture() (e.g.,
// "_test.go", "test_", "spec/") rather than the walker.FileInfo.IsTest flag. The
// IsTest flag is set here for correctness, but the scanner's skip logic is purely
// path-based. If you add new test file patterns, update isTestOrFixture() in secrets.go.
func TestScan_SkipsVariousTestFiles(t *testing.T) {
	dir := t.TempDir()
	testFiles := map[string]string{
		"app_test.go":        `const key = "AKIAIOSFODNN7EXAMPLE"`,
		"test_secrets.py":    `AWS_KEY = "AKIAIOSFODNN7EXAMPLE"`,
		"spec/secret_spec.rb": `KEY = "AKIAIOSFODNN7EXAMPLE"`,
	}
	var files []walker.FileInfo
	for name, content := range testFiles {
		p := writeFile(t, dir, name, content)
		files = append(files, walker.FileInfo{Path: p, RelPath: name, IsTest: true})
	}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "all test files should be skipped")
}

// --- Tests for expanded secret patterns (SC-084) ---

func TestScan_AnthropicKey(t *testing.T) {
	dir := t.TempDir()
	// Anthropic keys are ~93+ chars after sk-ant- prefix
	key := "sk-ant-" + "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_abcdefghijklmnopqrstuvwxyz01234567"
	path := writeFile(t, dir, "ai.py", `ANTHROPIC_KEY = "`+key+`"`)
	files := []walker.FileInfo{{Path: path, RelPath: "ai.py"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "Anthropic API key should be detected")
	assert.Contains(t, r.ByCategory, "api_key")
}

func TestScan_GitLabPAT(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "ci.sh", `export GITLAB_TOKEN="glpat-abcdefghij1234567890"`)
	files := []walker.FileInfo{{Path: path, RelPath: "ci.sh"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "GitLab PAT should be detected")
}

func TestScan_JWTToken(t *testing.T) {
	dir := t.TempDir()
	// A syntactically valid JWT (header.payload.signature)
	path := writeFile(t, dir, "auth.js", `const token = "eyJhbGciOiJIUzI1NiJ9.eyJzdWIiOiIxMjM0NTY3ODkwIn0.abcdefghijKLMN"`)
	files := []walker.FileInfo{{Path: path, RelPath: "auth.js"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "JWT token should be detected")
}

func TestScan_OpenSSHKey(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "id_ed25519", `-----BEGIN OPENSSH PRIVATE KEY-----
b3BlbnNzaC1rZXktdjEAAAAABG5vbmUAAAAEbm9uZQAAAAAAAAABAAAAMwAAAAtzc2gtZW
-----END OPENSSH PRIVATE KEY-----`)
	files := []walker.FileInfo{{Path: path, RelPath: "id_ed25519"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "OpenSSH private key should be detected")
	assert.Contains(t, r.ByCategory, "private_key")
}

func TestScan_ShopifyToken(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "shop.rb", `SHOPIFY_TOKEN = "shpat_abcdef0123456789abcdef0123456789"`)
	files := []walker.FileInfo{{Path: path, RelPath: "shop.rb"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "Shopify access token should be detected")
}

func TestScan_AMQPConnectionString(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "mq.py", `BROKER_URL = "amqp://user:password@rabbitmq.prod:5672/vhost"`)
	files := []walker.FileInfo{{Path: path, RelPath: "mq.py"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "AMQP connection string should be detected")
	assert.Contains(t, r.ByCategory, "connection_string")
}

func TestScan_GitHubAppToken(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "app.go", `var installToken = "ghs_ABCDEFGHIJKLMNOPQRSTUVWXYZabcdef12"`)
	files := []walker.FileInfo{{Path: path, RelPath: "app.go"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1, "GitHub App token should be detected")
}

func TestScan_SkipsExampleDirs(t *testing.T) {
	dir := t.TempDir()
	// Secrets in examples/ directories should be skipped (demo code, not production)
	path := writeFile(t, dir, "examples/auth/index.js",
		`const session = { secret: 'realPassword!2024' }`)
	files := []walker.FileInfo{{Path: path, RelPath: "examples/auth/index.js"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "examples/ directory should be skipped")
}

func TestScan_SkipsTestDirSingular(t *testing.T) {
	dir := t.TempDir()
	// Secrets in test/ (singular) directories should be skipped
	path := writeFile(t, dir, "test/unit/config.js",
		`const auth = { password: 's00pers3cret' }`)
	files := []walker.FileInfo{{Path: path, RelPath: "test/unit/config.js"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "test/ directory should be skipped")
}

func TestScan_SkipsReadmeFiles(t *testing.T) {
	dir := t.TempDir()
	// READMEs contain documentation examples, not real secrets
	path := writeFile(t, dir, "README.md", `
  auth: {
    username: 'janedoe',
    password: 's00pers3cret'
  },
`)
	files := []walker.FileInfo{{Path: path, RelPath: "README.md"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "README files should be skipped")
}

func TestScan_NoFalsePositives_NaturalLanguagePhrases(t *testing.T) {
	dir := t.TempDir()
	// Phrases with spaces (like "keyboard cat") are not real secrets
	path := writeFile(t, dir, "app.js", `
const session = { secret: "keyboard cat" }
app.use(cookieSession({ secret: "shhhh, very secret" }))
var config = { password: "some password here" }
`)
	files := []walker.FileInfo{{Path: path, RelPath: "app.js"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount,
		"natural language phrases with spaces should not be flagged as secrets")
}

func TestScan_StillDetectsRealGenericSecrets(t *testing.T) {
	dir := t.TempDir()
	// Real passwords without spaces should still be caught
	path := writeFile(t, dir, "config.go", `
var password = "realPassword!2024"
var secret = "aB3$kL9pQ2wX7mR5n"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	assert.GreaterOrEqual(t, r.SecretsCount, 1,
		"real secrets without spaces should still be detected")
}

func TestScan_FindingsHaveLocation(t *testing.T) {
	dir := t.TempDir()
	path := writeFile(t, dir, "config.go", `package config
const awsKey = "AKIAIOSFODNN7ABCDEFG"
`)
	files := []walker.FileInfo{{Path: path, RelPath: "config.go"}}
	r := Scan(files)
	require.Equal(t, 1, r.SecretsCount)
	require.Len(t, r.Findings, 1)
	assert.Equal(t, "config.go", r.Findings[0].RelPath)
	assert.Equal(t, 2, r.Findings[0].Line)
	assert.Equal(t, "AWS Access Key", r.Findings[0].Name)
	assert.Equal(t, "aws", r.Findings[0].Category)
}

func TestScan_SkipsRegexPatternDefinitions(t *testing.T) {
	dir := t.TempDir()
	tests := []struct {
		name    string
		file    string
		content string
	}{
		{"Go", "detector.go", `var p = regexp.MustCompile("-----BEGIN RSA PRIVATE KEY-----")`},
		{"Python", "detector.py", `p = re.compile(r"-----BEGIN RSA PRIVATE KEY-----")`},
		{"JavaScript", "detector.js", `const p = new RegExp("-----BEGIN RSA PRIVATE KEY-----")`},
		{"Java", "Detector.java", `Pattern p = Pattern.compile("-----BEGIN RSA PRIVATE KEY-----");`},
		{"PHP", "detector.php", `preg_match('/-----BEGIN RSA PRIVATE KEY-----/', $line);`},
		{"Ruby", "detector.rb", `p = Regexp.new("-----BEGIN RSA PRIVATE KEY-----")`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writeFile(t, dir, tt.file, tt.content)
			files := []walker.FileInfo{{Path: path, RelPath: tt.file}}
			r := Scan(files)
			assert.Equal(t, 0, r.SecretsCount, "%s regex pattern definition should be allowlisted", tt.name)
		})
	}
}

func TestScan_SkipsTemplateEnvVars(t *testing.T) {
	dir := t.TempDir()
	// Template variables referencing env vars are not real secrets
	path := writeFile(t, dir, "release.yml", `
      token: "{{ .Env.GITHUB_TOKEN }}"
      secret_key: ${SECRET_KEY}
      api_key: process.env.API_KEY
`)
	files := []walker.FileInfo{{Path: path, RelPath: "release.yml"}}
	r := Scan(files)
	assert.Equal(t, 0, r.SecretsCount, "template env var references should be allowlisted")
}
