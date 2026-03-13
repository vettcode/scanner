package security

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCheckLicense_GPL(t *testing.T) {
	r := &LicenseResult{}
	checkLicense("some-pkg", "GPL-3.0", r)
	assert.Len(t, r.Issues, 1)
	assert.Contains(t, r.Issues[0].Reason, "copyleft")
}

func TestCheckLicense_AGPL(t *testing.T) {
	r := &LicenseResult{}
	checkLicense("server-pkg", "AGPL-3.0-only", r)
	assert.Len(t, r.Issues, 1)
	assert.Contains(t, r.Issues[0].Reason, "Network copyleft")
}

func TestCheckLicense_MIT(t *testing.T) {
	r := &LicenseResult{}
	checkLicense("safe-pkg", "MIT", r)
	assert.Empty(t, r.Issues)
}

func TestCheckLicense_Apache(t *testing.T) {
	r := &LicenseResult{}
	checkLicense("safe-pkg", "Apache-2.0", r)
	assert.Empty(t, r.Issues)
}

func TestDetectLicenses_PHPComposer(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, "composer.json"), []byte(`{
		"license": "GPL-3.0"
	}`), 0644)

	r := DetectLicenses(dir)
	assert.Equal(t, 1, r.IssueCount)
	assert.Len(t, r.Licenses, 1)
	assert.Equal(t, "GPL-3.0", r.Licenses[0].License)
}

func TestDetectLicenses_NoManifests(t *testing.T) {
	r := DetectLicenses(t.TempDir())
	assert.Equal(t, 0, r.IssueCount)
}

func TestExtractLicenseStr(t *testing.T) {
	assert.Equal(t, "MIT", extractLicenseStr("MIT"))
	assert.Equal(t, "ISC", extractLicenseStr(map[string]interface{}{"type": "ISC"}))
	assert.Equal(t, "", extractLicenseStr(nil))
	assert.Equal(t, "", extractLicenseStr(42))
}
