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
