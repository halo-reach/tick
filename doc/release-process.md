# Tick CLI Release Process

## Pre-release Checklist

1. All planned changes merged to `main` and tested (PR merged, CI green)
2. Team members notified about the upcoming release to avoid tag conflicts

## Release Steps

```bash
# 1. Switch to main and pull latest
git checkout main
git pull

# 2. Create tag (semver: vMAJOR.MINOR.PATCH)
git tag v0.7.0
git push origin v0.7.0

# 3. Wait for GitHub Actions CI (~5-10 minutes)
#    https://github.com/halo-reach/tick/actions

# 4. Verify release assets (5 binaries + 5 .sha256 sidecars)
curl -fsI https://github.com/halo-reach/tick/releases/download/v0.7.0/tick-v0.7.0-darwin-arm64
# Expected: HTTP 200
```

## User Installation

### One-line Install (Recommended)

```bash
# Install latest release (requires public repo)
curl -fsSL https://raw.githubusercontent.com/halo-reach/tick/main/code/install.sh | sh

# Specify version
curl -fsSL https://raw.githubusercontent.com/halo-reach/tick/main/code/install.sh | \
  TICK_VERSION=v0.7.0 sh
```

### Private Repo Installation

When repo is Private, asset downloads require authentication. Use `TICK_INSTALL_TOKEN` (GitHub Token).

**Token Setup** (one-time):
1. GitHub Settings → Developer settings → Personal access tokens → Generate new token
2. Select `repo` scope
3. Save the token (only shown once)

**Install Command**:

```bash
# Recommended: put token in environment variable to avoid shell history leak
export TICK_INSTALL_TOKEN="<your-github-token>"
curl -fsSL https://raw.githubusercontent.com/halo-reach/tick/main/code/install.sh | \
  TICK_VERSION=v0.7.0 sh

# Or one-time
curl -fsSL https://raw.githubusercontent.com/halo-reach/tick/main/code/install.sh | \
  TICK_VERSION=v0.7.0 TICK_INSTALL_TOKEN="<your-github-token>" sh
```

**Security Tips**:
- Token is read-only and repo-scoped, minimal permissions
- Never commit token to git or write it in scripts
- Use `TICK_INSTALL_TOKEN` variable in CI/CD (mark as masked)
- If token is leaked, revoke immediately on GitHub

### Manual Download

```bash
# 1. Go to release page
#    https://github.com/halo-reach/tick/releases/v0.7.0
# 2. Download and place in PATH
sudo install -m 0755 tick-v0.7.0-darwin-arm64 /usr/local/bin/tick
tick --version   # v0.7.0
```

## Asset Naming Convention

| Filename | Purpose |
|----------|---------|
| `tick-v{VER}-{OS}-{ARCH}` | Binary (Windows: `.exe` suffix) |
| `tick-v{VER}-{OS}-{ARCH}.sha256` | SHA256 sidecar for binary |
| `SHA256SUMS` | All assets SHA256 summary (optional) |

## Rollback

```bash
# 1. Delete release on GitHub UI (or mark as deprecated)
# 2. Delete local + remote tag
git tag -d v0.7.0
git push origin :refs/tags/v0.7.0
```

## RC / Pre-release

Tags like `v0.7.0-rc1`, `v0.7.0-beta1` also trigger CI release workflow.

Install RC version:

```bash
TICK_VERSION=v0.7.0-rc1 curl -fsSL https://raw.githubusercontent.com/halo-reach/tick/main/code/install.sh | sh
```

## CI Flow

```
git push v0.7.0
    ↓
GitHub Actions detects tag matching /^v\d+\.\d+\.\d+/
    ↓
test job: golang:1.25 image, go mod verify + go test
    ↓ (test passes)
release job: make build-cross → 5 platforms + sha256 + SHA256SUMS
    ↓
upload to GitHub Release
    ↓
users download from release page / install via install.sh
```
