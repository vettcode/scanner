package language

import "strings"

// Tier represents the analysis depth for a language.
type Tier int

const (
	Tier1 Tier = 1 // Full analysis: AST complexity + dependency parsing + all metrics
	Tier2 Tier = 2 // Detection + LOC only: no complexity or dependency analysis
)

// Info holds metadata about a detected language.
type Info struct {
	Name       string
	Tier       Tier
	Extensions []string
}

// Tier1Languages are fully analyzed languages.
var Tier1Languages = map[string]Info{
	"JavaScript": {Name: "JavaScript", Tier: Tier1, Extensions: []string{".js", ".jsx", ".mjs", ".cjs"}},
	"TypeScript": {Name: "TypeScript", Tier: Tier1, Extensions: []string{".ts", ".tsx", ".mts", ".cts"}},
	"Python":     {Name: "Python", Tier: Tier1, Extensions: []string{".py", ".pyw"}},
	"Go":         {Name: "Go", Tier: Tier1, Extensions: []string{".go"}},
	"PHP":        {Name: "PHP", Tier: Tier1, Extensions: []string{".php", ".phtml"}},
	"Ruby":       {Name: "Ruby", Tier: Tier1, Extensions: []string{".rb", ".rake", ".gemspec"}},
	"Java":       {Name: "Java", Tier: Tier1, Extensions: []string{".java"}},
}

// Tier2Languages are detection + LOC only.
var Tier2Languages = map[string]Info{
	"HTML":       {Name: "HTML", Tier: Tier2, Extensions: []string{".html", ".htm"}},
	"CSS":        {Name: "CSS", Tier: Tier2, Extensions: []string{".css", ".scss", ".sass", ".less"}},
	"SQL":        {Name: "SQL", Tier: Tier2, Extensions: []string{".sql"}},
	"Shell":      {Name: "Shell", Tier: Tier2, Extensions: []string{".sh", ".bash", ".zsh"}},
	"Markdown":   {Name: "Markdown", Tier: Tier2, Extensions: []string{".md", ".markdown"}},
	"YAML":       {Name: "YAML", Tier: Tier2, Extensions: []string{".yml", ".yaml"}},
	"XML":        {Name: "XML", Tier: Tier2, Extensions: []string{".xml"}},
	"Dockerfile": {Name: "Dockerfile", Tier: Tier2, Extensions: []string{}}, // detected by filename
	"Terraform":  {Name: "Terraform", Tier: Tier2, Extensions: []string{".tf", ".tfvars"}},
	"JSON":       {Name: "JSON", Tier: Tier2, Extensions: []string{".json"}},
	"TOML":       {Name: "TOML", Tier: Tier2, Extensions: []string{".toml"}},
}

// extToLanguage is a reverse lookup: extension -> language name.
var extToLanguage map[string]string

func init() {
	extToLanguage = make(map[string]string)
	for _, info := range Tier1Languages {
		for _, ext := range info.Extensions {
			extToLanguage[ext] = info.Name
		}
	}
	for _, info := range Tier2Languages {
		for _, ext := range info.Extensions {
			extToLanguage[ext] = info.Name
		}
	}
}

// filenameToLanguage maps special filenames to languages.
var filenameToLanguage = map[string]string{
	"Dockerfile":     "Dockerfile",
	"Makefile":       "Shell",
	"Rakefile":       "Ruby",
	"Gemfile":        "Ruby",
	"Vagrantfile":    "Ruby",
	"Jenkinsfile":    "Shell",
	"docker-compose.yml":  "YAML",
	"docker-compose.yaml": "YAML",
}

// ManifestFiles maps manifest filenames to the language they indicate.
var ManifestFiles = map[string]string{
	"package.json":      "JavaScript",
	"package-lock.json": "JavaScript",
	"yarn.lock":         "JavaScript",
	"pnpm-lock.yaml":   "JavaScript",
	"tsconfig.json":     "TypeScript",
	"requirements.txt":  "Python",
	"setup.py":          "Python",
	"pyproject.toml":    "Python",
	"Pipfile":           "Python",
	"Pipfile.lock":      "Python",
	"poetry.lock":       "Python",
	"go.mod":            "Go",
	"go.sum":            "Go",
	"composer.json":     "PHP",
	"composer.lock":     "PHP",
	"Gemfile":           "Ruby",
	"Gemfile.lock":      "Ruby",
	"pom.xml":           "Java",
	"build.gradle":      "Java",
	"build.gradle.kts":  "Java",
}

// DetectByExtension returns the language name for a file extension, or empty string.
// Extensions are matched case-insensitively.
func DetectByExtension(ext string) string {
	return extToLanguage[strings.ToLower(ext)]
}

// DetectByFilename returns the language name for a special filename, or empty string.
func DetectByFilename(filename string) string {
	return filenameToLanguage[filename]
}

// IsTier1 returns true if the language name is a Tier 1 language.
func IsTier1(langName string) bool {
	_, ok := Tier1Languages[langName]
	return ok
}

// GetTier returns the tier for a language name, or 0 if unknown.
func GetTier(langName string) Tier {
	if info, ok := Tier1Languages[langName]; ok {
		return info.Tier
	}
	if info, ok := Tier2Languages[langName]; ok {
		return info.Tier
	}
	return 0
}
