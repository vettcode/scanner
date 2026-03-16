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
