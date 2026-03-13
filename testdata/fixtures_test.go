package testdata

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestAllFixturesExist(t *testing.T) {
	for _, name := range AllFixtures() {
		t.Run(name, func(t *testing.T) {
			assert.True(t, FixtureExists(name), "fixture %s should exist", name)
		})
	}
}

func TestFixturesDir(t *testing.T) {
	dir := FixturesDir()
	info, err := os.Stat(dir)
	require.NoError(t, err)
	assert.True(t, info.IsDir())
}

func TestHealthySaas_HasExpectedStructure(t *testing.T) {
	base := FixturePath(HealthySaas)
	expectedFiles := []string{
		"README.md",
		".env.example",
		".github/workflows/ci.yml",
		"frontend/package.json",
		"frontend/package-lock.json",
		"frontend/jest.config.js",
		"frontend/src/services/payment.ts",
		"frontend/src/services/user.ts",
		"frontend/src/components/Dashboard.tsx",
		"frontend/tests/payment.test.ts",
		"backend/requirements.txt",
		"backend/app/routes/users.py",
		"backend/app/routes/billing.py",
		"backend/tests/test_users.py",
		"backend/tests/test_billing.py",
		"backend/Dockerfile",
	}
	for _, f := range expectedFiles {
		path := filepath.Join(base, f)
		assert.FileExists(t, path, "expected file %s", f)
	}
}

func TestNeglectedProject_HasExpectedStructure(t *testing.T) {
	base := FixturePath(NeglectedProject)
	// Should have PHP source and deps, but NO tests, NO CI/CD, NO README
	expectedFiles := []string{
		"composer.json",
		"composer.lock",
		"src/controllers/UserController.php",
		"src/controllers/OrderController.php",
		"src/models/User.php",
		"src/utils/Helper.php",
	}
	for _, f := range expectedFiles {
		assert.FileExists(t, filepath.Join(base, f))
	}

	// Should NOT have test files, CI/CD, or README
	noTestDir := filepath.Join(base, "tests")
	_, err := os.Stat(noTestDir)
	assert.True(t, os.IsNotExist(err), "neglected project should have no tests/ directory")

	noCIDir := filepath.Join(base, ".github")
	_, err = os.Stat(noCIDir)
	assert.True(t, os.IsNotExist(err), "neglected project should have no .github/ directory")

	noReadme := filepath.Join(base, "README.md")
	_, err = os.Stat(noReadme)
	assert.True(t, os.IsNotExist(err), "neglected project should have no README.md")
}

func TestSecurityNightmare_HasExpectedStructure(t *testing.T) {
	base := FixturePath(SecurityNightmare)
	expectedFiles := []string{
		"Gemfile",
		"Gemfile.lock",
		"app/controllers/api_controller.rb",
		"app/controllers/auth_controller.rb",
		"app/config/secrets.yml",
		"app/config/database.yml",
		".env",
		"spec/api_controller_spec.rb",
		"README.md",
	}
	for _, f := range expectedFiles {
		assert.FileExists(t, filepath.Join(base, f))
	}
}

func TestJavaEnterprise_HasExpectedStructure(t *testing.T) {
	base := FixturePath(JavaEnterprise)
	expectedFiles := []string{
		"api/pom.xml",
		"api/src/main/java/com/example/controllers/UserController.java",
		"api/src/main/java/com/example/services/PaymentService.java",
		"api/src/test/java/com/example/UserControllerTest.java",
		"worker/go.mod.fixture",
		"worker/go.sum.fixture",
		"worker/cmd/main.go",
		"worker/internal/processor/processor.go",
		"worker/Dockerfile",
		".github/workflows/ci.yml",
		"docker-compose.yml",
		"README.md",
		".env.example",
	}
	for _, f := range expectedFiles {
		assert.FileExists(t, filepath.Join(base, f))
	}
}

func TestTier2Only_HasExpectedStructure(t *testing.T) {
	base := FixturePath(Tier2Only)
	expectedFiles := []string{
		"public/index.html",
		"public/styles.css",
		"config/nginx.conf",
		"config/docker-compose.yml",
		"infrastructure/terraform/main.tf",
		"infrastructure/k8s/deployment.yaml",
		"README.md",
	}
	for _, f := range expectedFiles {
		assert.FileExists(t, filepath.Join(base, f))
	}

	// Should NOT have any Tier 1 language source files (recursive check)
	tier1Extensions := map[string]bool{
		".go": true, ".py": true, ".js": true, ".ts": true,
		".tsx": true, ".php": true, ".rb": true, ".java": true,
	}
	err := filepath.WalkDir(base, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := filepath.Ext(path)
		assert.False(t, tier1Extensions[ext],
			"tier2-only should not have Tier 1 file: %s", path)
		return nil
	})
	assert.NoError(t, err)
}
