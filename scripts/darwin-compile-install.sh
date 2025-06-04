#!/bin/sh
set -euo pipefail

# Version detection (tag or commit)
wed_version="$(git describe --always --tags 2>/dev/null || git rev-parse --short HEAD)"

# Binary directory (macOS Homebrew-friendly)
if [ -n "${GOBIN:-}" ]; then
    BIN_DIR="$GOBIN"
elif [ -n "${GOPATH:-}" ]; then
    BIN_DIR="${GOPATH}/bin"
elif [ -d "${HOME}/go/bin" ]; then
    BIN_DIR="${HOME}/go/bin"
else
    BIN_DIR="/usr/local/bin" 
fi


# Build and install
echo "Compiling wed@${wed_version}..."
go build -ldflags="-s -w -X main.Version=$wed_version" -o "$BIN_DIR/wed" ./cmd/wed
chmod 755 "$BIN_DIR/wed"

# Man pages (Homebrew-compatible)
if [ -d "manuals" ] && command -v man >/dev/null 2>&1; then
    MAN_DIR="/usr/local/share/man"
    
    for manpage in man/*.[0-9]; do
        [ -f "$manpage" ] || continue
        
        # Get base name (wed.1) and section (1)
        page_name=$(basename "$manpage")
        section="${page_name##*.}"
        
        # Create target directory and compress
        mkdir -p "$MAN_DIR/man$section"
        gzip -9 -c "$manpage" > "$MAN_DIR/man$section/$page_name.gz" && {
            echo "  ✓ Installed: $page_name.gz"
        } || {
            echo "  ✗ Failed to install: $page_name" >&2
        }
    done
fi

echo "Installation complete"
echo "Binary installed to: $BIN_DIR/wed"
