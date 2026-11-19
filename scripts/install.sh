#!/bin/sh
set -e

REPO="DazFather/Wednesday"

# Detect OS and ARCH
OS="$(uname | tr '[:upper:]' '[:lower:]')"
ARCH="$(uname -m)"

# Map ARCH to Go's naming
case "$ARCH" in
  x86_64 | amd64) ARCH="amd64" ;;
  aarch64 | arm64) ARCH="arm64" ;;
  *) echo "Unsupported architecture: $ARCH" >&2; exit 1 ;;
esac

# Use GitHub API to find latest release tag
VERSION=$(curl -sSfL "https://api.github.com/repos/$REPO/releases/latest" | sed -n 's/.*"tag_name"[[:space:]]*:[[:space:]]*"\([^"]*\)".*/\1/p' | head -n 1)
if [ -z "$VERSION" ]; then
  echo "Unable to determine latest release version" >&2
  exit 1
fi

# Build target archive name
FILENAME="wed-${OS}-${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

# Temp dir
TMP_DIR=$(mktemp -d)
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

# Download the archive
echo "Downloading $FILENAME (version $VERSION)..."
curl -sSfL "$URL" -o "$TMP_DIR/$FILENAME"

# Verify the downloaded archive against the release checksums
SUMS_URL="https://github.com/$REPO/releases/download/$VERSION/SHA256SUMS"
curl -sSfL "$SUMS_URL" -o "$TMP_DIR/SHA256SUMS"
SUM_LINE=$(grep "  \./${FILENAME}\$" "$TMP_DIR/SHA256SUMS" || true)
if [ -z "$SUM_LINE" ]; then
  echo "Checksum for $FILENAME not found" >&2
  exit 1
fi
if command -v sha256sum >/dev/null 2>&1; then
  printf '%s\n' "$SUM_LINE" | (cd "$TMP_DIR" && sha256sum -c -)
else
  printf '%s\n' "$SUM_LINE" | (cd "$TMP_DIR" && shasum -a 256 -c -)
fi

# Extract and run included install script
tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"
cd "$TMP_DIR"
if [ -f ./install.sh ]; then
  echo "Running installer..."
  chmod +x ./install.sh
  # Old release archives may carry a #!/bin/bash shebang even when the
  # container has no bash, so invoke the script with an available shell.
  if command -v bash >/dev/null 2>&1; then
    bash ./install.sh
  else
    sh ./install.sh
  fi
else
  echo "Install script not found in archive!" >&2
  exit 1
fi
echo "✓ Installed successfully"

