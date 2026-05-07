# VettCode Scanner — Release Guide

## First-Time Setup (One-Time)

### 1. GitHub Repositories

Create the following repos under the `vettcode` GitHub org:

- **`vettcode/scanner`** — the scanner source code (this repo)
- **`vettcode/homebrew-tap`** — Homebrew formula (auto-updated by GoReleaser)

### 2. GitHub Secrets

Configure these secrets on `vettcode/scanner`:

| Secret | Purpose |
|---|---|
| `DOCKERHUB_USERNAME` | Docker Hub org username |
| `DOCKERHUB_TOKEN` | Docker Hub access token (read/write) |
| `HOMEBREW_TAP_GITHUB_TOKEN` | PAT with `repo` scope on `vettcode/homebrew-tap` |

`GITHUB_TOKEN` is provided automatically by GitHub Actions.

### 3. Docker Hub

- Create the `vettcode` organization on Docker Hub
- Create the `vettcode/scanner` repository

### 4. GitHub Container Registry

- GHCR uses `GITHUB_TOKEN` automatically — no extra setup needed
- Images publish to `ghcr.io/vettcode/scanner`

### 5. Ed25519 Signing Key

Generate the production signing key:

```bash
# Generate a 32-byte random seed
openssl rand 32 > scanner-key-2026-03.seed

# Base64-encode it for the environment variable
base64 < scanner-key-2026-03.seed
# → set this as VETTCODE_SIGNING_KEY in your build environment

# Derive the public key (for the platform's key registry)
# Use the Go tool or openssl to derive the Ed25519 public key from the seed.
```

Key injection options (choose one):
- **CI secret**: Set `VETTCODE_SIGNING_KEY` as a GitHub Actions secret
- **File-based**: Set `VETTCODE_SIGNING_KEY_FILE` pointing to a mounted secret

**Key ID:** Update `ScannerKeyID` in `internal/output/signer.go` if the key ID changes.

Store the seed securely (e.g., 1Password vault, GCP Secret Manager). Back it up — losing this key means you can't sign compatible scan results until you rotate.

### 6. DNS — Install Script

Set up `get.vettcode.com` to serve `scripts/install.sh`:

**Option A: GitHub Pages**
- Create a `vettcode/get.vettcode.com` repo
- Copy `scripts/install.sh` as `index.html` with `Content-Type: text/plain`
- Configure custom domain in GitHub Pages settings

**Option B: Vercel**
- Deploy a static site that serves the install script at `/`
- Configure the `get.vettcode.com` domain in Vercel

**DNS record:** `get.vettcode.com` → CNAME to your host (GitHub Pages or Vercel)

### 7. Grammar Hosting (GCS)

> Can be deferred if GCP is not yet available — the scanner downloads grammars
> on first run, so this is needed for production but not for a private beta.

- Create a GCS multi-region bucket: `vettcode-grammars`
- Upload WASM grammar files to `gs://vettcode-grammars/0.1.0/`
- Set bucket to public read (grammars are not sensitive)
- Update `GrammarManifest` SHA-256 checksums in `internal/grammar/manager.go`

---

## Release Checklist (Every Release)

### Pre-Release

- [ ] All tests pass: `go test -race ./...`
- [ ] Lint clean: `golangci-lint run`
- [ ] `CHANGELOG.md` updated with new version section
- [ ] Version bumped in `internal/cli/version.go`
- [ ] OSV database snapshot updated (run `go run ./cmd/osv-snapshot`)
- [ ] Grammar checksums in `internal/grammar/manager.go` match published grammars
- [ ] Signing key is configured (check `VETTCODE_SIGNING_KEY` or `VETTCODE_SIGNING_KEY_FILE`)
- [ ] If major release: new signing key generated, public key registered on platform

### Tag and Release

```bash
# Ensure you're on main with a clean tree
git checkout main
git pull origin main

# Create a signed tag
git tag -s v1.0.0 -m "v1.0.0: Initial release"

# Push the tag — triggers the release pipeline
git push origin v1.0.0
```

### Post-Release Verification

- [ ] GitHub Release page has binaries for: darwin/arm64, darwin/amd64, linux/amd64, windows/amd64
- [ ] `checksums.txt` is present in the release
- [ ] Docker images published:
  - `docker pull vettcode/scanner:v1.0.0`
  - `docker pull ghcr.io/vettcode/scanner:v1.0.0`
- [ ] Homebrew tap updated: `brew install vettcode/tap/vettcode && vettcode version`
- [ ] Install script works: `curl -sSfL https://get.vettcode.com | sh && vettcode version`
- [ ] Smoke test on each platform:
  ```bash
  vettcode scan /path/to/test-repo --offline
  ```
- [ ] Scan result JSON is valid and signed
- [ ] Docker smoke test:
  ```bash
  docker run -v $(pwd):/scan vettcode/scanner scan /scan --offline
  ```

### Rollback

If a critical issue is found post-release:

1. Delete the GitHub Release (hides binaries from download)
2. Delete the Docker tags: `v1.0.0`, `latest`
3. Revert the Homebrew tap formula
4. Push a new patch tag (`v1.0.1`) with the fix
