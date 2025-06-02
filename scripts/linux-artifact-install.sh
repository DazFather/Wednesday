#!/bin/bash
set -euo pipefail

# Determine binary installation directory
if [ -n "${GOBIN:-}" ]; then
    BIN_DIR="$GOBIN"
elif [ -n "${GOPATH:-}" ]; then
    BIN_DIR="${GOPATH}/bin"
elif [ -d "${HOME}/go/bin" ]; then
    BIN_DIR="${HOME}/go/bin"
else
    BIN_DIR="${HOME}/.local/bin"
fi

# Create bin directory if it doesn't exist
mkdir -p "${BIN_DIR}"

# Install binary
if cp wed "${BIN_DIR}/wed"; then
    chmod 755 "${BIN_DIR}/wed"
    echo "Installed wed to ${BIN_DIR}"
    echo "Ensure it's in your PATH:"
    echo "  export PATH=\"${BIN_DIR}:\$PATH\""
else
    echo "Failed to install wed to ${BIN_DIR}" >&2
    exit 1
fi

# Install man pages if available and man is installed
if [ -d "manuals" ] && command -v man >/dev/null 2>&1; then

    # Determine man page destination
    find_man_dir() {
        # 1. Check XDG-compliant user location first
        local user_man_dir="${XDG_DATA_HOME:-$HOME/.local/share}/man"
        if [ -d "$user_man_dir" ]; then
            echo "$user_man_dir"
            return
        fi

        # 2. Check traditional user location
        local traditional_man_dir="$HOME/.local/man"
        if [ -d "$traditional_man_dir" ]; then
            echo "$traditional_man_dir"
            return
        fi

        # 3. Check if manpath exists
        if command -v manpath >/dev/null 2>&1; then
            local manpath_output
            manpath_output=$(manpath 2>/dev/null)
            for path in ${manpath_output//:/ }; do
                if [[ "$path" == "$HOME"/* ]]; then
                    echo "$path"
                    return
                fi
            done
        fi

        # 4. Fallback to creating XDG-compliant directory
        mkdir -p "${user_man_dir}/man1"
        echo "$user_man_dir"
    }

    MAN_DIR=$(find_man_dir)
    echo "man detected, installing pages to '${MAN_DIR}'"

    for manpage in manuals/*.gz; do
        [ -f "$manpage" ] || continue

        # Extract base name (wed.1.gz > wed)
        page_name=$(basename "$manpage" | cut -d. -f1)
        printf "  Adding page '%s'..." "$page_name"

        # Extract section (wed.1.gz > 1)
        section=$(basename "$manpage" | cut -d. -f2)

        # Validate section is a digit
        if [[ ! "$section" =~ ^[0-9]+$ ]]; then
            echo " [ERROR: Invalid section number]" >&2
            continue
        fi

        dest_dir="${MAN_DIR}/man${section}"
        mkdir -p "$dest_dir"
        
        if cp "$manpage" "$dest_dir/"; then
            echo " [OK]"
        else
            echo " [FAILED]" >&2
        fi
    done
    
    # Update man database if mandb exists
    if command -v mandb >/dev/null 2>&1; then
        echo "Updating man database..."
        mandb -q "$MAN_DIR" >/dev/null 2>&1 || true
    fi
fi

echo "Installation complete"
