#!/bin/sh
set -e

OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)
case "$ARCH" in
  x86_64) ARCH="amd64" ;;
  aarch64|arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH"; exit 1 ;;
esac

VERSION="${TICK_VERSION:-latest}"
PROJECT_PATH="${TICK_PROJECT_PATH:-halo-reach/tick}"
GITHUB_URL="${TICK_GITHUB_URL:-https://github.com}"
API_URL="${GITHUB_URL}/api/v3/repos/${PROJECT_PATH}"
BINARY="tick-${VERSION}-${OS}-${ARCH}"

case "$OS" in
  mingw*|msys*|cygwin*|windows*) BINARY="${BINARY}.exe" ;;
esac
BINARY_SHA="${BINARY}.sha256"

INSTALL_DIR="${TICK_INSTALL_DIR:-/usr/local/bin}"
TMP_DIR="${TMPDIR:-/tmp}"
TMP_BIN="${TMP_DIR}/tick-new"

AUTH_HEADER=""
if [ -n "${TICK_INSTALL_TOKEN}" ]; then
  AUTH_HEADER="--header Authorization: Basic $(printf '%s' "${TICK_INSTALL_TOKEN}" | base64)"
fi

echo "Fetching release info for ${VERSION}..."
RELEASE_JSON="$(curl -fsSL ${AUTH_HEADER} "${API_URL}/releases/${VERSION}")"

if command -v jq >/dev/null 2>&1; then
  ASSET_URL="$(echo "${RELEASE_JSON}" | jq -r --arg n "${BINARY}" '.assets.links[]? | select(.name==$n) | .direct_asset_url // .url' | head -1)"
  SHA_URL="$(echo "${RELEASE_JSON}" | jq -r --arg n "${BINARY_SHA}" '.assets.links[]? | select(.name==$n) | .direct_asset_url // .url' | head -1)"
else
  ASSET_URL="$(echo "${RELEASE_JSON}" | tr ',' '\n' | grep -A1 "\"name\":\"${BINARY}\"" | grep '"url"' | head -1 | sed -E 's/.*"url":"([^"]+)".*/\1/')"
  SHA_URL="$(echo "${RELEASE_JSON}" | tr ',' '\n' | grep -A1 "\"name\":\"${BINARY_SHA}\"" | grep '"url"' | head -1 | sed -E 's/.*"url":"([^"]+)".*/\1/')"
fi

if [ -z "${ASSET_URL}" ] || [ "${ASSET_URL}" = "null" ]; then
  ASSET_URL="${GITHUB_URL}/${PROJECT_PATH}/releases/download/${VERSION}/${BINARY}"
  SHA_URL="${GITHUB_URL}/${PROJECT_PATH}/releases/download/${VERSION}/${BINARY_SHA}"
  echo "warning: API did not return asset, falling back to direct link" >&2
fi

echo "Downloading tick ${VERSION} for ${OS}/${ARCH}..."
curl -fsSL ${AUTH_HEADER} "${ASSET_URL}" -o "${TMP_BIN}"
chmod +x "${TMP_BIN}"

echo "Verifying SHA256..."
if [ -n "${SHA_URL}" ] && [ "${SHA_URL}" != "null" ]; then
  EXPECTED_SHA="$(curl -fsSL ${AUTH_HEADER} "${SHA_URL}" 2>/dev/null | awk '{print $1}')"
fi
if [ -n "${EXPECTED_SHA}" ]; then
  if command -v sha256sum >/dev/null 2>&1; then
    ACTUAL_SHA="$(sha256sum "${TMP_BIN}" | awk '{print $1}')"
  else
    ACTUAL_SHA="$(shasum -a 256 "${TMP_BIN}" | awk '{print $1}')"
  fi
  if [ "${EXPECTED_SHA}" != "${ACTUAL_SHA}" ]; then
    echo "SHA256 mismatch: expected=${EXPECTED_SHA} got=${ACTUAL_SHA}" >&2
    echo "Installation aborted, binary discarded." >&2
    rm -f "${TMP_BIN}"
    exit 1
  fi
  echo "SHA256 verified (${ACTUAL_SHA})"
else
  echo "warning: could not download .sha256 sidecar, skipping verification" >&2
fi

if [ "${OS}" = "darwin" ]; then
  xattr -dr com.apple.quarantine "${TMP_BIN}" 2>/dev/null || true
fi

SAME_VOL=1
if [ "${OS}" = "darwin" ] || [ "${OS}" = "linux" ]; then
  TMP_DEV="$(df -P "${TMP_DIR}" 2>/dev/null | tail -1 | awk '{print $1}')"
  INST_DEV="$(df -P "${INSTALL_DIR}" 2>/dev/null | tail -1 | awk '{print $1}')"
  if [ -n "${TMP_DEV}" ] && [ -n "${INST_DEV}" ] && [ "${TMP_DEV}" != "${INST_DEV}" ]; then
    SAME_VOL=0
  fi
fi

echo "Installing to ${INSTALL_DIR}/tick..."
if [ -w "${INSTALL_DIR}" ]; then
  if [ "${SAME_VOL}" = "1" ]; then
    mv "${TMP_BIN}" "${INSTALL_DIR}/tick"
  else
    cp "${TMP_BIN}" "${INSTALL_DIR}/tick" && rm -f "${TMP_BIN}"
  fi
  chmod 0755 "${INSTALL_DIR}/tick"
else
  echo "sudo required to write to ${INSTALL_DIR}..."
  if [ "${SAME_VOL}" = "1" ]; then
    sudo mv "${TMP_BIN}" "${INSTALL_DIR}/tick"
  else
    sudo cp "${TMP_BIN}" "${INSTALL_DIR}/tick" && sudo rm -f "${TMP_BIN}"
  fi
  sudo chmod 0755 "${INSTALL_DIR}/tick"
fi

echo "tick installed to ${INSTALL_DIR}/tick"
tick --version || true
