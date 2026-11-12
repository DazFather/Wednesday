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

# Build target ZIP name
FILENAME="wed-${OS}-${ARCH}.tar.gz"
URL="https://github.com/$REPO/releases/download/$VERSION/$FILENAME"

# Temp dir
TMP_DIR=$(mktemp -d)
cleanup() { rm -rf "$TMP_DIR"; }
trap cleanup EXIT

# Download the ZIP
echo "Downloading $FILENAME (version $VERSION)..."
curl -sSL "$URL" -o "$TMP_DIR/$FILENAME"

# Extract and run included install script
tar -xzf "$TMP_DIR/$FILENAME" -C "$TMP_DIR"
cd "$TMP_DIR"
if [ -f ./install.sh ]; then
  echo "Running installer..."
  chmod +x ./install.sh
  ./install.sh
else
  echo "Install script not found in archive!" >&2
  exit 1
fi
echo "✓ Installed successfully"

