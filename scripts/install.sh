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
VERSION=$(curl -sSf "https://api.github.com/repos/$REPO/releases/latest" | grep -Po '"tag_name": "\K.*?(?=")')

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
INSTALL_SCRIPT="$TMP_DIR/install.sh"
if [ -x $INSTALL_SCRIPT ]; then
  echo "Running installer..."
  chmod +x $INSTALL_SCRIPT
  $INSTALL_SCRIPT
else
  echo "Install script not found in archive!" >&2
  exit 1
fi

echo "✓ Installed successfully"

