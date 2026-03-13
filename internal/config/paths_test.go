package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestParsePaths_DefaultCurrentDir(t *testing.T) {
	repos, err := ParsePaths(nil, nil)
	require.NoError(t, err)
	require.Len(t, repos, 1)

	cwd, _ := os.Getwd()
	assert.Equal(t, cwd, repos[0].Path)
}

func TestParsePaths_SinglePath(t *testing.T) {
	dir := t.TempDir()
	repos, err := ParsePaths([]string{dir}, nil)
	require.NoError(t, err)
	require.Len(t, repos, 1)

	absDir, _ := filepath.Abs(dir)
	assert.Equal(t, absDir, repos[0].Path)
	assert.Equal(t, filepath.Base(absDir), repos[0].Name)
}

func TestParsePaths_MultiplePaths(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	repos, err := ParsePaths([]string{dir1, dir2}, nil)
	require.NoError(t, err)
	require.Len(t, repos, 2)
}

func TestParsePaths_WithLabels(t *testing.T) {
	dir := t.TempDir()
	repos, err := ParsePaths([]string{dir}, []string{"myapp:" + dir})
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "myapp", repos[0].Name)
}

func TestParsePaths_LabelsOnly(t *testing.T) {
	dir := t.TempDir()
	repos, err := ParsePaths(nil, []string{"myapp:" + dir})
	require.NoError(t, err)
	require.Len(t, repos, 1)
	assert.Equal(t, "myapp", repos[0].Name)
}

func TestParsePaths_NonExistentPath(t *testing.T) {
	_, err := ParsePaths([]string{"/nonexistent/path/xyz"}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestParsePaths_FileNotDir(t *testing.T) {
	f, err := os.CreateTemp("", "testfile")
	require.NoError(t, err)
	f.Close()
	defer os.Remove(f.Name())

	_, err = ParsePaths([]string{f.Name()}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a directory")
}

func TestParsePaths_DuplicatePaths(t *testing.T) {
	dir := t.TempDir()
	_, err := ParsePaths([]string{dir, dir}, nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "duplicate")
}

func TestParsePaths_InvalidLabel(t *testing.T) {
	_, err := ParsePaths(nil, []string{"nocolon"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid --label")
}

func TestParsePaths_LabelsOnlyDeterministicOrder(t *testing.T) {
	dir1 := t.TempDir()
	dir2 := t.TempDir()
	labels := []string{"beta:" + dir1, "alpha:" + dir2}

	// Run multiple times to verify deterministic order
	for i := 0; i < 5; i++ {
		repos, err := ParsePaths(nil, labels)
		require.NoError(t, err)
		require.Len(t, repos, 2)
		// Should always be alphabetical: alpha, beta
		assert.Equal(t, "alpha", repos[0].Name)
		assert.Equal(t, "beta", repos[1].Name)
	}
}

func TestValidateOutputPath_ValidPath(t *testing.T) {
	dir := t.TempDir()
	err := ValidateOutputPath(filepath.Join(dir, "output.json"))
	assert.NoError(t, err)
}

func TestValidateOutputPath_NonExistentDir(t *testing.T) {
	err := ValidateOutputPath("/nonexistent/dir/output.json")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestValidateOutputPath_CurrentDir(t *testing.T) {
	err := ValidateOutputPath("./vettcode-scan-result.json")
	assert.NoError(t, err)
}
