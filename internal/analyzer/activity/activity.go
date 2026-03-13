package activity

import (
	"fmt"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

// Result holds the development activity analysis results.
type Result struct {
	LastCommitDate     *time.Time
	DaysSinceLastCommit int
	CommitVelocity     float64   // avg commits/month over last 12 months
	MonthlyCommits     [12]int   // commits per month, index 0 = most recent month
	ActiveMonths       int       // months with >0 commits in last 12
	Trend              string    // "increasing", "stable", "declining"
	ContributorCount   int
	TotalCommits       int       // total commits in last 12 months
	HeadSHA            string    // HEAD commit SHA
	HasGit             bool
}

// Analyze runs git-based analysis on a repository root.
func Analyze(root string) *Result {
	r := &Result{}

	// Check for git directory
	if _, err := exec.LookPath("git"); err != nil {
		return r
	}

	// Check if this is a git repo
	out, err := gitCmd(root, "rev-parse", "--is-inside-work-tree")
	if err != nil || strings.TrimSpace(out) != "true" {
		return r
	}
	r.HasGit = true

	// HEAD SHA
	if sha, err := gitCmd(root, "rev-parse", "HEAD"); err == nil {
		r.HeadSHA = strings.TrimSpace(sha)
	}

	// Last commit date
	if dateStr, err := gitCmd(root, "log", "-1", "--format=%aI"); err == nil {
		dateStr = strings.TrimSpace(dateStr)
		if t, err := time.Parse(time.RFC3339, dateStr); err == nil {
			r.LastCommitDate = &t
			days := int(time.Since(t).Hours() / 24)
			if days < 0 {
				days = 0
			}
			r.DaysSinceLastCommit = days
		}
	}

	// Monthly commits for last 12 months
	now := time.Now()
	for i := 0; i < 12; i++ {
		monthStart := time.Date(now.Year(), now.Month()-time.Month(i), 1, 0, 0, 0, 0, time.UTC)
		monthEnd := time.Date(now.Year(), now.Month()-time.Month(i)+1, 1, 0, 0, 0, 0, time.UTC)

		after := fmt.Sprintf("--after=%s", monthStart.Format("2006-01-02"))
		before := fmt.Sprintf("--before=%s", monthEnd.Format("2006-01-02"))

		if countStr, err := gitCmd(root, "rev-list", "--count", "HEAD", after, before); err == nil {
			if n, err := strconv.Atoi(strings.TrimSpace(countStr)); err == nil {
				r.MonthlyCommits[i] = n
				r.TotalCommits += n
				if n > 0 {
					r.ActiveMonths++
				}
			}
		}
	}

	if r.ActiveMonths > 0 {
		r.CommitVelocity = float64(r.TotalCommits) / 12.0
	}

	// Trend: compare first half vs second half of the 12-month window
	r.Trend = computeTrend(r.MonthlyCommits)

	// Contributor count (unique authors)
	if authorStr, err := gitCmd(root, "log", "--format=%ae",
		fmt.Sprintf("--since=%s", now.AddDate(0, -12, 0).Format("2006-01-02"))); err == nil {
		authors := make(map[string]bool)
		for _, email := range strings.Split(strings.TrimSpace(authorStr), "\n") {
			email = strings.TrimSpace(email)
			if email != "" {
				authors[email] = true
			}
		}
		r.ContributorCount = len(authors)
	}

	return r
}

// computeTrend classifies the commit trend based on first half vs second half.
func computeTrend(monthly [12]int) string {
	// First half = months 0-5 (most recent), second half = months 6-11 (older)
	recentTotal := 0
	olderTotal := 0
	for i := 0; i < 6; i++ {
		recentTotal += monthly[i]
	}
	for i := 6; i < 12; i++ {
		olderTotal += monthly[i]
	}

	if recentTotal == 0 && olderTotal == 0 {
		return "stable"
	}

	// >25% increase → increasing, >25% decrease → declining, else stable
	if olderTotal == 0 {
		return "increasing"
	}

	ratio := float64(recentTotal) / float64(olderTotal)
	switch {
	case ratio > 1.25:
		return "increasing"
	case ratio < 0.75:
		return "declining"
	default:
		return "stable"
	}
}

func gitCmd(dir string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	out, err := cmd.Output()
	return string(out), err
}
