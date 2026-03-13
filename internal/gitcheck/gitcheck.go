package gitcheck

import (
	"fmt"
	"os/exec"
	"regexp"
	"strconv"
	"strings"
)

// MinMajor is the minimum required Git major version.
const MinMajor = 2

// MinMinor is the minimum required Git minor version.
const MinMinor = 20

// Result holds the outcome of a Git version check.
type Result struct {
	Available bool
	Version   string
	Supported bool
	Message   string // warning message if not supported
}

// Check validates that Git is installed and meets the minimum version requirement (2.20+).
// Returns a Result indicating availability and whether the version is supported.
func Check() Result {
	_, err := exec.LookPath("git")
	if err != nil {
		return Result{
			Available: false,
			Message:   "WARN: Git not found\n  → Skipping development activity analysis.\n    Install Git 2.20+ to enable commit history metrics.",
		}
	}

	out, err := exec.Command("git", "--version").Output()
	if err != nil {
		return Result{
			Available: false,
			Message:   "WARN: Git not found or version < 2.20\n  → Skipping development activity analysis.\n    Install Git 2.20+ to enable commit history metrics.",
		}
	}

	versionStr := strings.TrimSpace(string(out))
	major, minor, err := parseGitVersion(versionStr)
	if err != nil {
		return Result{
			Available: true,
			Version:   versionStr,
			Supported: false,
			Message:   fmt.Sprintf("WARN: Could not parse Git version %q\n  → Skipping development activity analysis.", versionStr),
		}
	}

	ver := fmt.Sprintf("%d.%d", major, minor)
	if major < MinMajor || (major == MinMajor && minor < MinMinor) {
		return Result{
			Available: true,
			Version:   ver,
			Supported: false,
			Message:   fmt.Sprintf("WARN: Git version %s found, but 2.20+ is required\n  → Skipping development activity analysis.\n    Install Git 2.20+ to enable commit history metrics.", ver),
		}
	}

	return Result{
		Available: true,
		Version:   ver,
		Supported: true,
	}
}

var gitVersionRe = regexp.MustCompile(`(\d+)\.(\d+)`)

func parseGitVersion(versionOutput string) (major, minor int, err error) {
	matches := gitVersionRe.FindStringSubmatch(versionOutput)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("cannot parse version from %q", versionOutput)
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}
	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}
	return major, minor, nil
}
