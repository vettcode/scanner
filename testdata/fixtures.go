// Package testdata provides helpers for locating test fixtures.
package testdata

import (
	"os"
	"path/filepath"
	"runtime"
)

// FixturesDir returns the absolute path to the testdata/fixtures directory.
func FixturesDir() string {
	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		panic("testdata: runtime.Caller(0) failed")
	}
	return filepath.Join(filepath.Dir(filename), "fixtures")
}

// FixturePath returns the absolute path to a specific fixture repo.
func FixturePath(name string) string {
	return filepath.Join(FixturesDir(), name)
}

// FixtureExists reports whether a fixture directory exists.
func FixtureExists(name string) bool {
	info, err := os.Stat(FixturePath(name))
	return err == nil && info.IsDir()
}

// Fixture names.
const (
	HealthySaas       = "healthy-saas"
	NeglectedProject  = "neglected-project"
	SecurityNightmare = "security-nightmare"
	JavaEnterprise    = "java-enterprise"
	Tier2Only         = "tier2-only"
)

// AllFixtures returns the names of all expected fixture repos.
func AllFixtures() []string {
	return []string{
		HealthySaas,
		NeglectedProject,
		SecurityNightmare,
		JavaEnterprise,
		Tier2Only,
	}
}
