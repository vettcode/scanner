package language

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDetectByExtension(t *testing.T) {
	tests := []struct {
		ext      string
		expected string
	}{
		{".go", "Go"},
		{".js", "JavaScript"},
		{".jsx", "JavaScript"},
		{".ts", "TypeScript"},
		{".tsx", "TypeScript"},
		{".py", "Python"},
		{".php", "PHP"},
		{".rb", "Ruby"},
		{".java", "Java"},
		{".html", "HTML"},
		{".css", "CSS"},
		{".yml", "YAML"},
		{".yaml", "YAML"},
		{".md", "Markdown"},
		{".sh", "Shell"},
		{".sql", "SQL"},
		{".tf", "Terraform"},
		{".xml", "XML"},
		{".unknown", ""},
		// Case-insensitive extensions
		{".GO", "Go"},
		{".Js", "JavaScript"},
		{".PY", "Python"},
	}
	for _, tt := range tests {
		t.Run(tt.ext, func(t *testing.T) {
			assert.Equal(t, tt.expected, DetectByExtension(tt.ext))
		})
	}
}

func TestDetectByFilename(t *testing.T) {
	assert.Equal(t, "Dockerfile", DetectByFilename("Dockerfile"))
	assert.Equal(t, "Shell", DetectByFilename("Makefile"))
	assert.Equal(t, "Ruby", DetectByFilename("Rakefile"))
	assert.Equal(t, "", DetectByFilename("README.md"))
}

func TestIsTier1(t *testing.T) {
	tier1 := []string{"JavaScript", "TypeScript", "Python", "Go", "PHP", "Ruby", "Java"}
	for _, lang := range tier1 {
		assert.True(t, IsTier1(lang), "%s should be Tier 1", lang)
	}

	tier2 := []string{"HTML", "CSS", "YAML", "Shell", "Markdown"}
	for _, lang := range tier2 {
		assert.False(t, IsTier1(lang), "%s should not be Tier 1", lang)
	}
}

func TestGetTier(t *testing.T) {
	assert.Equal(t, Tier1, GetTier("Go"))
	assert.Equal(t, Tier1, GetTier("Python"))
	assert.Equal(t, Tier2, GetTier("HTML"))
	assert.Equal(t, Tier2, GetTier("YAML"))
	assert.Equal(t, Tier(0), GetTier("Unknown"))
}

func TestClassifyFile(t *testing.T) {
	tests := []struct {
		path     string
		lang     string
		tier     Tier
		isTest   bool
		isNil    bool
	}{
		{"src/main.go", "Go", Tier1, false, false},
		{"src/main_test.go", "Go", Tier1, true, false},
		{"src/app.js", "JavaScript", Tier1, false, false},
		{"src/app.test.js", "JavaScript", Tier1, true, false},
		{"src/__tests__/app.js", "JavaScript", Tier1, true, false},
		{"test/app.js", "JavaScript", Tier1, true, false},
		{"tests/integration/helpers.js", "JavaScript", Tier1, true, false},
		{"src/app.spec.ts", "TypeScript", Tier1, true, false},
		{"src/main.py", "Python", Tier1, false, false},
		{"tests/test_main.py", "Python", Tier1, true, false},
		{"src/main_test.py", "Python", Tier1, true, false},
		{"src/Main.java", "Java", Tier1, false, false},
		{"src/test/MainTest.java", "Java", Tier1, true, false},
		{"spec/app_spec.rb", "Ruby", Tier1, true, false},
		{"tests/AppTest.php", "PHP", Tier1, true, false},
		{"index.html", "HTML", Tier2, false, false},
		{"style.css", "CSS", Tier2, false, false},
		{"Dockerfile", "Dockerfile", Tier2, false, false},
		{"unknown.xyz", "", 0, false, true},
		// .d.ts declaration files should be detected as TypeScript
		{"src/types.d.ts", "TypeScript", Tier1, false, false},
		// Makefile detection
		{"build/Makefile", "Shell", Tier2, false, false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := ClassifyFile(tt.path)
			if tt.isNil {
				assert.Nil(t, result)
				return
			}
			assert.NotNil(t, result)
			assert.Equal(t, tt.lang, result.Language)
			assert.Equal(t, tt.tier, result.Tier)
			assert.Equal(t, tt.isTest, result.IsTest)
		})
	}
}

func TestManifestFiles(t *testing.T) {
	expected := map[string]string{
		"package.json":     "JavaScript",
		"tsconfig.json":    "TypeScript",
		"requirements.txt": "Python",
		"go.mod":           "Go",
		"composer.json":    "PHP",
		"Gemfile":          "Ruby",
		"pom.xml":          "Java",
	}
	for file, lang := range expected {
		t.Run(file, func(t *testing.T) {
			assert.Equal(t, lang, ManifestFiles[file])
		})
	}
}

func TestAggregateResults(t *testing.T) {
	files := map[string]int{
		"TypeScript": 30000,
		"Python":     10000,
		"HTML":       5000,
	}

	result := AggregateResults(files)
	assert.True(t, result.HasTier1)
	assert.Equal(t, 45000, result.TotalLOC)
	assert.InDelta(t, 66.67, result.Percentages["TypeScript"], 0.01)
	assert.InDelta(t, 22.22, result.Percentages["Python"], 0.01)
	assert.InDelta(t, 11.11, result.Percentages["HTML"], 0.01)
	assert.Equal(t, []string{"Python", "TypeScript"}, result.Tier1Languages)
	assert.Equal(t, []string{"HTML", "Python", "TypeScript"}, result.DetectedLanguages)
}

func TestAggregateResults_NoTier1(t *testing.T) {
	files := map[string]int{
		"HTML": 5000,
		"CSS":  3000,
	}
	result := AggregateResults(files)
	assert.False(t, result.HasTier1)
	assert.Empty(t, result.Tier1Languages)
}

func TestAggregateResults_Empty(t *testing.T) {
	result := AggregateResults(map[string]int{})
	assert.False(t, result.HasTier1)
	assert.Equal(t, 0, result.TotalLOC)
}
