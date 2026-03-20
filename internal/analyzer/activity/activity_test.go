package activity

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupGitRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Initialize git repo
	run := func(args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, string(out))
	}

	run("init")
	run("config", "user.email", "test@example.com")
	run("config", "user.name", "Test")

	// Create a file and commit
	os.WriteFile(filepath.Join(dir, "main.go"), []byte("package main\n"), 0644)
	run("add", "main.go")
	run("commit", "-m", "initial commit")

	return dir
}

func TestAnalyze_GitRepo(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	dir := setupGitRepo(t)
	r := Analyze(dir)

	assert.True(t, r.HasGit)
	assert.NotNil(t, r.LastCommitDate)
	assert.Equal(t, 0, r.DaysSinceLastCommit)
	assert.NotEmpty(t, r.HeadSHA)
	assert.Equal(t, 1, r.ContributorCount)
	assert.GreaterOrEqual(t, r.TotalCommits, 1)
	assert.GreaterOrEqual(t, r.ActiveMonths, 1)
}

func TestAnalyze_NonGitDir(t *testing.T) {
	dir := t.TempDir()
	r := Analyze(dir)
	assert.False(t, r.HasGit)
	assert.Nil(t, r.LastCommitDate)
	assert.Equal(t, 0, r.ContributorCount)
}

func TestComputeTrend_Increasing(t *testing.T) {
	monthly := [12]int{10, 10, 10, 10, 10, 10, 2, 2, 2, 2, 2, 2}
	assert.Equal(t, "increasing", computeTrend(monthly))
}

func TestComputeTrend_Declining(t *testing.T) {
	monthly := [12]int{2, 2, 2, 2, 2, 2, 10, 10, 10, 10, 10, 10}
	assert.Equal(t, "declining", computeTrend(monthly))
}

func TestComputeTrend_Stable(t *testing.T) {
	monthly := [12]int{5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5, 5}
	assert.Equal(t, "stable", computeTrend(monthly))
}

func TestComputeTrend_AllZeros(t *testing.T) {
	monthly := [12]int{}
	assert.Equal(t, "stable", computeTrend(monthly))
}

func TestAnalyze_FullClone_NotShallow(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	dir := setupGitRepo(t)
	r := Analyze(dir)

	assert.True(t, r.HasGit)
	assert.False(t, r.IsShallowClone)
}

func TestAnalyze_ShallowClone(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not found")
	}

	// Create a source repo with multiple commits
	srcDir := setupGitRepo(t)
	run := func(dir string, args ...string) {
		cmd := exec.Command("git", args...)
		cmd.Dir = dir
		cmd.Env = append(os.Environ(),
			"GIT_AUTHOR_NAME=Test",
			"GIT_AUTHOR_EMAIL=test@example.com",
			"GIT_COMMITTER_NAME=Test",
			"GIT_COMMITTER_EMAIL=test@example.com",
		)
		out, err := cmd.CombinedOutput()
		require.NoError(t, err, "git %v failed: %s", args, string(out))
	}

	// Add a second commit
	os.WriteFile(filepath.Join(srcDir, "extra.go"), []byte("package extra\n"), 0644)
	run(srcDir, "add", "extra.go")
	run(srcDir, "commit", "-m", "second commit")

	// Shallow clone with depth 1 (file:// protocol required for local shallow clone)
	cloneDir := t.TempDir()
	run(cloneDir, "clone", "--depth", "1", "file://"+srcDir, "shallow")
	shallowDir := filepath.Join(cloneDir, "shallow")

	r := Analyze(shallowDir)
	assert.True(t, r.HasGit)
	assert.True(t, r.IsShallowClone)
}
