package infra

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/vettcode/scanner/internal/walker"
)

// Result holds the infrastructure detection results.
type Result struct {
	HasIaC        bool
	IaCTools      []string // e.g., "Docker", "Terraform", "Kubernetes"
	HasCICD       bool
	CICDProviders []string // e.g., "GitHub Actions", "GitLab CI"
	HasMonitoring bool
	MonitorTools  []string // e.g., "Datadog", "Sentry", "Prometheus"
}

// iacDirPatterns maps directory/file names to IaC tool names.
var iacFilePatterns = map[string]string{
	"Dockerfile":           "Docker",
	"dockerfile":           "Docker",
	"docker-compose.yml":   "Docker Compose",
	"docker-compose.yaml":  "Docker Compose",
	"compose.yml":          "Docker Compose",
	"compose.yaml":         "Docker Compose",
	"Pulumi.yaml":          "Pulumi",
	"Pulumi.yml":           "Pulumi",
	"serverless.yml":       "Serverless",
	"serverless.yaml":      "Serverless",
}

// iacExtPatterns maps file extensions to IaC tools.
var iacExtPatterns = map[string]string{
	".tf":      "Terraform",
	".tfvars":  "Terraform",
}

// cicdPathPatterns maps path patterns to CI/CD providers.
var cicdPathPatterns = []struct {
	pattern  string
	provider string
}{
	{".github/workflows", "GitHub Actions"},
	{".gitlab-ci.yml", "GitLab CI"},
	{".circleci", "CircleCI"},
	{"Jenkinsfile", "Jenkins"},
	{".travis.yml", "Travis CI"},
	{"bitbucket-pipelines.yml", "Bitbucket Pipelines"},
	{"azure-pipelines.yml", "Azure Pipelines"},
	{".buildkite", "Buildkite"},
}

// k8sFilePatterns are filenames that indicate Kubernetes manifests.
var k8sFilePatterns = []string{
	"deployment.yaml", "deployment.yml",
	"service.yaml", "service.yml",
	"ingress.yaml", "ingress.yml",
	"statefulset.yaml", "statefulset.yml",
	"daemonset.yaml", "daemonset.yml",
	"kustomization.yaml", "kustomization.yml",
	"Chart.yaml", // Helm
}

// cloudFormationPatterns indicate AWS CloudFormation templates.
var cloudFormationPatterns = []string{
	"template.yaml", "template.yml",
	"cloudformation.yaml", "cloudformation.yml",
}

// monitoringDeps maps dependency names to monitoring tools.
var monitoringDeps = map[string]string{
	"datadog":                    "Datadog",
	"dd-trace":                   "Datadog",
	"ddtrace":                    "Datadog",
	"@datadog/browser-rum":       "Datadog",
	"@sentry/node":               "Sentry",
	"@sentry/browser":            "Sentry",
	"@sentry/react":              "Sentry",
	"sentry-sdk":                 "Sentry",
	"sentry_sdk":                 "Sentry",
	"newrelic":                    "New Relic",
	"@newrelic/browser-agent":    "New Relic",
	"prometheus_client":          "Prometheus",
	"prom-client":                "Prometheus",
	"prometheus-client":          "Prometheus",
	"grafana":                    "Grafana",
	"@grafana/agent":             "Grafana",
	"opentelemetry":              "OpenTelemetry",
	"@opentelemetry/api":         "OpenTelemetry",
	"@opentelemetry/sdk-node":    "OpenTelemetry",
	"bugsnag":                    "Bugsnag",
	"@bugsnag/js":                "Bugsnag",
	"rollbar":                    "Rollbar",
	"honeybadger":                "Honeybadger",
	"elastic-apm-node":           "Elastic APM",
	"elasticapm":                 "Elastic APM",
}

// monitoringConfigFiles maps filenames to monitoring tools.
var monitoringConfigFiles = map[string]string{
	"prometheus.yml":       "Prometheus",
	"prometheus.yaml":      "Prometheus",
	"datadog.yaml":         "Datadog",
	".sentryclirc":         "Sentry",
	"sentry.properties":   "Sentry",
	"newrelic.yml":         "New Relic",
	"newrelic.js":          "New Relic",
}

// Analyze detects infrastructure tooling from walked files and dependencies.
func Analyze(root string, files []walker.FileInfo, deps []string) *Result {
	r := &Result{}
	iacSet := make(map[string]bool)
	cicdSet := make(map[string]bool)
	monitorSet := make(map[string]bool)

	for _, f := range files {
		base := filepath.Base(f.Path)
		relPath := f.RelPath

		// Check IaC file patterns
		if tool, ok := iacFilePatterns[base]; ok {
			iacSet[tool] = true
		}

		// Check IaC extension patterns
		ext := filepath.Ext(base)
		if tool, ok := iacExtPatterns[ext]; ok {
			iacSet[tool] = true
		}

		// Check K8s patterns
		lowerBase := strings.ToLower(base)
		for _, p := range k8sFilePatterns {
			if strings.EqualFold(base, p) {
				iacSet["Kubernetes"] = true
				break
			}
		}

		// Check CloudFormation
		for _, p := range cloudFormationPatterns {
			if lowerBase == p {
				iacSet["CloudFormation"] = true
				break
			}
		}

		// Check CI/CD path patterns
		for _, cp := range cicdPathPatterns {
			if strings.Contains(relPath, cp.pattern) || base == cp.pattern {
				cicdSet[cp.provider] = true
			}
		}

		// Check monitoring config files
		if tool, ok := monitoringConfigFiles[base]; ok {
			monitorSet[tool] = true
		}
	}

	// Also check root for CI/CD directories/files that may not be in walked files
	checkRootCICD(root, cicdSet)

	// Check dependencies for monitoring tools
	for _, dep := range deps {
		depLower := strings.ToLower(dep)
		if tool, ok := monitoringDeps[depLower]; ok {
			monitorSet[tool] = true
		}
	}

	r.IaCTools = setToSlice(iacSet)
	r.HasIaC = len(r.IaCTools) > 0

	r.CICDProviders = setToSlice(cicdSet)
	r.HasCICD = len(r.CICDProviders) > 0

	r.MonitorTools = setToSlice(monitorSet)
	r.HasMonitoring = len(r.MonitorTools) > 0

	return r
}

// checkRootCICD checks the root directory for CI/CD directories that may be
// excluded from the walker (e.g., .github is a dotdir).
func checkRootCICD(root string, cicdSet map[string]bool) {
	if _, err := os.Stat(filepath.Join(root, ".github", "workflows")); err == nil {
		cicdSet["GitHub Actions"] = true
	}
	if _, err := os.Stat(filepath.Join(root, ".gitlab-ci.yml")); err == nil {
		cicdSet["GitLab CI"] = true
	}
	if _, err := os.Stat(filepath.Join(root, ".circleci")); err == nil {
		cicdSet["CircleCI"] = true
	}
	if _, err := os.Stat(filepath.Join(root, ".buildkite")); err == nil {
		cicdSet["Buildkite"] = true
	}
	if _, err := os.Stat(filepath.Join(root, ".travis.yml")); err == nil {
		cicdSet["Travis CI"] = true
	}
}

func setToSlice(s map[string]bool) []string {
	result := make([]string, 0, len(s))
	for k := range s {
		result = append(result, k)
	}
	// Sort for deterministic output
	sortStrings(result)
	return result
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
