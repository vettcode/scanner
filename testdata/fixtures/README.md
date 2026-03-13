# Test Fixtures

Five fixture repositories for testing the VettCode scanner across all analysis dimensions.

## Fixtures

### healthy-saas (JS/TS + Python)
Well-maintained SaaS app. Expect: maintainability B+/A-, security A-, good test coverage.
- Frontend: TypeScript (React/Next.js) with tests
- Backend: Python (FastAPI) with tests
- CI/CD: GitHub Actions
- No secrets, no CVEs, healthy dependencies

### neglected-project (PHP)
Stale, unmaintained project. Expect: red flags for no tests, no CI/CD, high complexity.
- PHP (Laravel 5.8) with outdated dependencies
- No tests, no CI/CD, no README
- High cyclomatic complexity (avg ~12)
- Dependencies 5+ years old

### security-nightmare (Ruby)
Security issues galore. Expect: red flags for secrets, CVEs.
- Ruby on Rails 5.2 with vulnerable gems
- Planted secrets: AWS keys, API tokens, DB passwords, private keys
- 8+ detectable secrets across .env, config files, source code
- Some tests (low coverage)

### java-enterprise (Java + Go)
Multi-language enterprise app. Expect: correct multi-lang analysis.
- Java: Spring Boot API with Maven dependencies
- Go: Worker service with go.mod
- Docker + CI/CD
- Tests in both languages

### tier2-only (HTML + CSS + YAML)
No Tier 1 languages. Expect: LOC/tech stack reported, complexity N/A.
- Static HTML/CSS site
- Infrastructure: Terraform, Kubernetes, Docker, Nginx
- No dependency manifests
- No programming language source code
