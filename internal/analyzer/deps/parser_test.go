package deps

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeTestFile(t *testing.T, dir, name, content string) {
	t.Helper()
	path := filepath.Join(dir, name)
	os.MkdirAll(filepath.Dir(path), 0755)
	require.NoError(t, os.WriteFile(path, []byte(content), 0644))
}

func TestParseNPM(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "package.json", `{
		"dependencies": {
			"express": "^4.18.0",
			"react": "^18.2.0"
		},
		"devDependencies": {
			"jest": "^29.0.0"
		}
	}`)

	deps := parseNPM(dir)
	assert.Len(t, deps, 3)

	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
		assert.Equal(t, "npm", d.Ecosystem)
		assert.Equal(t, "JavaScript", d.Language)
	}
	assert.True(t, names["express"])
	assert.True(t, names["react"])
	assert.True(t, names["jest"])
}

func TestParsePython(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "requirements.txt", `
# Core deps
flask==2.3.0
requests>=2.28
pandas
numpy==1.24.0
`)
	deps := parsePython(dir)
	assert.GreaterOrEqual(t, len(deps), 4)
	for _, d := range deps {
		assert.Equal(t, "pypi", d.Ecosystem)
	}
}

func TestParseGo(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "go.mod", `module github.com/example/app

go 1.22

require (
	github.com/gin-gonic/gin v1.9.1
	github.com/stretchr/testify v1.8.4
	golang.org/x/crypto v0.15.0 // indirect
)
`)
	deps := parseGo(dir)
	assert.Len(t, deps, 2) // excludes indirect
	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
		assert.Equal(t, "go", d.Ecosystem)
	}
	assert.True(t, names["github.com/gin-gonic/gin"])
	assert.True(t, names["github.com/stretchr/testify"])
}

func TestParseGo_SingleLineRequire(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "go.mod", `module github.com/example/small

go 1.22

require github.com/pkg/errors v0.9.1
require golang.org/x/sys v0.5.0 // indirect
`)
	deps := parseGo(dir)
	require.Len(t, deps, 1) // excludes indirect
	assert.Equal(t, "github.com/pkg/errors", deps[0].Name)
	assert.Equal(t, "0.9.1", deps[0].Version)
}

func TestParsePHP(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "composer.json", `{
		"require": {
			"php": "^8.1",
			"laravel/framework": "^10.0",
			"ext-pdo": "*"
		},
		"require-dev": {
			"phpunit/phpunit": "^10.0"
		}
	}`)
	deps := parsePHP(dir)
	assert.Len(t, deps, 2) // excludes php and ext-pdo
	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
		assert.Equal(t, "packagist", d.Ecosystem)
	}
	assert.True(t, names["laravel/framework"])
	assert.True(t, names["phpunit/phpunit"])
}

func TestParseRuby_Gemfile(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "Gemfile", `
source 'https://rubygems.org'

gem 'rails', '~> 7.0'
gem 'puma'
# gem 'commented_out'
`)
	deps := parseRuby(dir)
	assert.Len(t, deps, 2)
	names := make(map[string]bool)
	for _, d := range deps {
		names[d.Name] = true
		assert.Equal(t, "rubygems", d.Ecosystem)
	}
	assert.True(t, names["rails"])
	assert.True(t, names["puma"])
}

func TestParseRuby_GemfileLock(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "Gemfile.lock", `GEM
  remote: https://rubygems.org/
  specs:
    actioncable (7.0.4)
    actionpack (7.0.4)
    puma (5.6.5)

PLATFORMS
  ruby

DEPENDENCIES
  rails (~> 7.0)
`)
	deps := parseRuby(dir)
	assert.Len(t, deps, 3)
	assert.Equal(t, "actioncable", deps[0].Name)
	assert.Equal(t, "7.0.4", deps[0].Version)
}

func TestParseJava_PomXML(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "pom.xml", `<?xml version="1.0"?>
<project>
  <dependencies>
    <dependency>
      <groupId>org.springframework.boot</groupId>
      <artifactId>spring-boot-starter</artifactId>
      <version>3.1.0</version>
    </dependency>
    <dependency>
      <groupId>org.junit.jupiter</groupId>
      <artifactId>junit-jupiter</artifactId>
    </dependency>
  </dependencies>
</project>`)
	deps := parseJava(dir)
	assert.Len(t, deps, 2)
	assert.Equal(t, "org.springframework.boot:spring-boot-starter", deps[0].Name)
	assert.Equal(t, "3.1.0", deps[0].Version)
	assert.Equal(t, "maven", deps[0].Ecosystem)
}

func TestParseJava_BuildGradle(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "build.gradle", `
plugins {
    id 'java'
}

dependencies {
    implementation 'org.springframework.boot:spring-boot-starter:3.1.0'
    testImplementation 'org.junit.jupiter:junit-jupiter:5.9.0'
}
`)
	deps := parseJava(dir)
	assert.Len(t, deps, 2)
	assert.Equal(t, "org.springframework.boot:spring-boot-starter", deps[0].Name)
	assert.Equal(t, "3.1.0", deps[0].Version)
}

func TestParseDependencies_MultiEcosystem(t *testing.T) {
	dir := t.TempDir()
	writeTestFile(t, dir, "package.json", `{"dependencies":{"express":"^4.18.0"}}`)
	writeTestFile(t, dir, "go.mod", "module example\n\ngo 1.22\n\nrequire (\n\tgithub.com/gin-gonic/gin v1.9.1\n)\n")

	r := ParseDependencies(dir)
	assert.GreaterOrEqual(t, len(r.Dependencies), 2)
	assert.Contains(t, r.Ecosystems, "npm")
	assert.Contains(t, r.Ecosystems, "go")
}

func TestParseDependencies_NoDeps(t *testing.T) {
	r := ParseDependencies(t.TempDir())
	assert.Empty(t, r.Dependencies)
	assert.Empty(t, r.Ecosystems)
}

func TestCleanVersion(t *testing.T) {
	assert.Equal(t, "4.18.0", cleanVersion("^4.18.0"))
	assert.Equal(t, "4.18.0", cleanVersion("~4.18.0"))
	assert.Equal(t, "2.28", cleanVersion(">=2.28"))
	assert.Equal(t, "2.3.0", cleanVersion("==2.3.0"))
	assert.Equal(t, "1.0.0", cleanVersion("v1.0.0"))
	assert.Equal(t, "", cleanVersion(""))
}
