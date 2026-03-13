package techstack

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetect_Frameworks(t *testing.T) {
	deps := []string{"react", "express", "prisma", "stripe"}
	r := Detect(t.TempDir(), deps)
	assert.Contains(t, r.Frameworks, "React")
	assert.Contains(t, r.Frameworks, "Express")
}

func TestDetect_Databases(t *testing.T) {
	deps := []string{"pg", "redis", "mongoose"}
	r := Detect(t.TempDir(), deps)
	assert.Contains(t, r.Databases, "PostgreSQL")
	assert.Contains(t, r.Databases, "Redis")
	assert.Contains(t, r.Databases, "MongoDB")
}

func TestDetect_Services(t *testing.T) {
	deps := []string{"stripe", "@sendgrid/mail", "boto3"}
	r := Detect(t.TempDir(), deps)
	assert.Contains(t, r.Services, "Stripe")
	assert.Contains(t, r.Services, "SendGrid")
	assert.Contains(t, r.Services, "AWS SDK (boto3)")
}

func TestDetect_NodeVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".nvmrc"), []byte("v20.11.0\n"), 0644)

	r := Detect(dir, nil)
	assert.Len(t, r.Runtimes, 1)
	assert.Equal(t, "Node.js", r.Runtimes[0].Name)
	assert.Equal(t, "20.11.0", r.Runtimes[0].Version)
}

func TestDetect_PythonVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".python-version"), []byte("3.12.1"), 0644)

	r := Detect(dir, nil)
	assert.Len(t, r.Runtimes, 1)
	assert.Equal(t, "Python", r.Runtimes[0].Name)
	assert.Equal(t, "3.12.1", r.Runtimes[0].Version)
}

func TestDetect_GoVersion(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "go.mod"), []byte("module example\n\ngo 1.22\n"), 0644)

	r := Detect(dir, nil)
	assert.Len(t, r.Runtimes, 1)
	assert.Equal(t, "Go", r.Runtimes[0].Name)
	assert.Equal(t, "1.22", r.Runtimes[0].Version)
}

func TestDetect_NodeEngineFromPackageJSON(t *testing.T) {
	dir := t.TempDir()
	pkg := `{"engines": {"node": ">=18.0.0"}}`
	os.WriteFile(filepath.Join(dir, "package.json"), []byte(pkg), 0644)

	r := Detect(dir, nil)
	assert.Len(t, r.Runtimes, 1)
	assert.Equal(t, "Node.js", r.Runtimes[0].Name)
	assert.Equal(t, ">=18.0.0", r.Runtimes[0].Version)
}

func TestDetect_Empty(t *testing.T) {
	r := Detect(t.TempDir(), nil)
	assert.Empty(t, r.Frameworks)
	assert.Empty(t, r.Databases)
	assert.Empty(t, r.Services)
	assert.Empty(t, r.Runtimes)
}

func TestDetect_NoDuplicates(t *testing.T) {
	deps := []string{"react", "react-dom"} // both map to React
	r := Detect(t.TempDir(), deps)
	count := 0
	for _, fw := range r.Frameworks {
		if fw == "React" {
			count++
		}
	}
	assert.Equal(t, 1, count, "React should appear exactly once")
}
