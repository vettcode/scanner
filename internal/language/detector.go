package language

import (
	"path/filepath"
	"sort"
	"strings"
)

// FileClassification holds the language detection result for a single file.
type FileClassification struct {
	Language string
	Tier     Tier
	IsTest   bool
	LOC      int
}

// DetectionResult holds the aggregate language detection for a repository.
type DetectionResult struct {
	// Languages maps language name to LOC count.
	Languages map[string]int
	// Percentages maps language name to percentage of total LOC.
	Percentages map[string]float64
	// HasTier1 is true if any Tier 1 language was detected.
	HasTier1 bool
	// DetectedLanguages is a sorted list of all detected language names.
	DetectedLanguages []string
	// Tier1Languages is a sorted list of detected Tier 1 language names.
	Tier1Languages []string
	// TotalLOC is the total lines of code across all languages.
	TotalLOC int
}

// ClassifyFile determines the language and test status of a file.
func ClassifyFile(path string) *FileClassification {
	base := filepath.Base(path)
	ext := filepath.Ext(path)

	// Try special filenames first
	if lang := DetectByFilename(base); lang != "" {
		return &FileClassification{
			Language: lang,
			Tier:     GetTier(lang),
			IsTest:   false,
		}
	}

	// Try extension
	if lang := DetectByExtension(ext); lang != "" {
		isTest := isTestFile(path, lang)
		return &FileClassification{
			Language: lang,
			Tier:     GetTier(lang),
			IsTest:   isTest,
		}
	}

	return nil // unrecognized language
}

// AggregateResults computes language percentages from file classifications.
func AggregateResults(files map[string]int) *DetectionResult {
	result := &DetectionResult{
		Languages:   make(map[string]int),
		Percentages: make(map[string]float64),
	}

	totalLOC := 0
	for lang, loc := range files {
		result.Languages[lang] = loc
		totalLOC += loc
	}
	result.TotalLOC = totalLOC

	if totalLOC == 0 {
		return result
	}

	tier1Set := make(map[string]bool)
	allLangs := make(map[string]bool)

	for lang, loc := range files {
		pct := float64(loc) / float64(totalLOC) * 100.0
		result.Percentages[lang] = pct
		allLangs[lang] = true
		if IsTier1(lang) {
			tier1Set[lang] = true
			result.HasTier1 = true
		}
	}

	for lang := range allLangs {
		result.DetectedLanguages = append(result.DetectedLanguages, lang)
	}
	sort.Strings(result.DetectedLanguages)

	for lang := range tier1Set {
		result.Tier1Languages = append(result.Tier1Languages, lang)
	}
	sort.Strings(result.Tier1Languages)

	return result
}

// isTestFile detects if a file is a test file based on language-specific patterns.
func isTestFile(path string, lang string) bool {
	base := filepath.Base(path)
	dir := filepath.Dir(path)
	parts := strings.Split(filepath.ToSlash(dir), "/")

	switch lang {
	case "Go":
		return strings.HasSuffix(base, "_test.go")
	case "JavaScript", "TypeScript":
		if strings.HasSuffix(base, ".test.js") || strings.HasSuffix(base, ".test.ts") ||
			strings.HasSuffix(base, ".test.jsx") || strings.HasSuffix(base, ".test.tsx") ||
			strings.HasSuffix(base, ".spec.js") || strings.HasSuffix(base, ".spec.ts") ||
			strings.HasSuffix(base, ".spec.jsx") || strings.HasSuffix(base, ".spec.tsx") {
			return true
		}
		for _, p := range parts {
			if p == "__tests__" || p == "test" || p == "tests" {
				return true
			}
		}
	case "Python":
		if strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py") {
			return true
		}
		for _, p := range parts {
			if p == "tests" {
				return true
			}
		}
	case "Java":
		if strings.HasSuffix(base, "Test.java") || strings.HasSuffix(base, "Tests.java") {
			return true
		}
		for _, p := range parts {
			if p == "test" {
				// Check if under src/test/
				for i, pp := range parts {
					if pp == "src" && i+1 < len(parts) && parts[i+1] == "test" {
						return true
					}
				}
			}
		}
	case "Ruby":
		if strings.HasSuffix(base, "_spec.rb") {
			return true
		}
		for _, p := range parts {
			if p == "spec" {
				return true
			}
		}
	case "PHP":
		if strings.HasSuffix(base, "Test.php") {
			return true
		}
		for _, p := range parts {
			if p == "tests" {
				return true
			}
		}
	}
	return false
}
