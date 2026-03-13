package output

import "fmt"

// ANSI color codes
const (
	ansiReset  = "\033[0m"
	ansiBold   = "\033[1m"
	ansiRed    = "\033[31m"
	ansiGreen  = "\033[32m"
	ansiYellow = "\033[33m"
	ansiCyan   = "\033[36m"
	ansiGray   = "\033[90m"
)

// ColorConfig controls whether ANSI color codes are emitted.
type ColorConfig struct {
	Enabled bool
}

func (c *ColorConfig) bold(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiBold + s + ansiReset
}

func (c *ColorConfig) red(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiRed + s + ansiReset
}

func (c *ColorConfig) green(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiGreen + s + ansiReset
}

func (c *ColorConfig) yellow(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiYellow + s + ansiReset
}

func (c *ColorConfig) cyan(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiCyan + s + ansiReset
}

func (c *ColorConfig) gray(s string) string {
	if !c.Enabled {
		return s
	}
	return ansiGray + s + ansiReset
}

// gradeColor returns the grade string colored based on its value.
func (c *ColorConfig) gradeColor(grade string) string {
	if !c.Enabled {
		return grade
	}
	switch {
	case len(grade) > 0 && grade[0] == 'A':
		return ansiGreen + ansiBold + grade + ansiReset
	case len(grade) > 0 && grade[0] == 'B':
		return ansiGreen + grade + ansiReset
	case len(grade) > 0 && grade[0] == 'C':
		return ansiYellow + grade + ansiReset
	case len(grade) > 0 && grade[0] == 'D':
		return ansiYellow + grade + ansiReset
	case len(grade) > 0 && grade[0] == 'F':
		return ansiRed + ansiBold + grade + ansiReset
	default:
		return grade
	}
}

// severityColor colors a severity string.
func (c *ColorConfig) severityColor(severity string) string {
	if !c.Enabled {
		return severity
	}
	switch severity {
	case "critical":
		return ansiRed + ansiBold + severity + ansiReset
	case "high":
		return ansiRed + severity + ansiReset
	case "medium":
		return ansiYellow + severity + ansiReset
	default:
		return severity
	}
}

// yesNo formats a boolean detection with optional detail.
func (c *ColorConfig) yesNo(detected bool, detail string) string {
	if detected {
		if detail != "" {
			return c.green(fmt.Sprintf("Yes (%s)", detail))
		}
		return c.green("Yes")
	}
	return "No"
}
