package gitcheck

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseGitVersion(t *testing.T) {
	tests := []struct {
		input    string
		major    int
		minor    int
		hasError bool
	}{
		{"git version 2.39.3 (Apple Git-146)", 2, 39, false},
		{"git version 2.20.0", 2, 20, false},
		{"git version 1.8.5", 1, 8, false},
		{"git version 2.45.1.windows.1", 2, 45, false},
		{"not a version", 0, 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			major, minor, err := parseGitVersion(tt.input)
			if tt.hasError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.major, major)
				assert.Equal(t, tt.minor, minor)
			}
		})
	}
}

func TestCheck_ReturnsResult(t *testing.T) {
	// This test just verifies the function runs without panic.
	// Actual git availability depends on the environment.
	result := Check()
	if result.Available {
		assert.NotEmpty(t, result.Version)
		// On most dev machines, git is 2.20+
		if result.Supported {
			assert.Empty(t, result.Message)
		}
	} else {
		assert.NotEmpty(t, result.Message)
	}
}

func TestVersionCheck_TooOld(t *testing.T) {
	major, minor, err := parseGitVersion("git version 1.9.0")
	assert.NoError(t, err)

	supported := !(major < MinMajor || (major == MinMajor && minor < MinMinor))
	assert.False(t, supported)
}

func TestVersionCheck_ExactMinimum(t *testing.T) {
	major, minor, err := parseGitVersion("git version 2.20.0")
	assert.NoError(t, err)

	supported := !(major < MinMajor || (major == MinMajor && minor < MinMinor))
	assert.True(t, supported)
}

func TestVersionCheck_AboveMinimum(t *testing.T) {
	major, minor, err := parseGitVersion("git version 2.39.3")
	assert.NoError(t, err)

	supported := !(major < MinMajor || (major == MinMajor && minor < MinMinor))
	assert.True(t, supported)
}
