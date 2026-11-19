#!/bin/sh
# Darwin (macOS) installation script (user-local only)
set -eu

# Determine binary installation directory
if [ -n "${GOBIN:-}" ]; then
    BIN_DIR="$GOBIN"
elif [ -n "${GOPATH:-}" ]; then
    BIN_DIR="${GOPATH}/bin"
elif [ -d "${HOME}/go/bin" ]; then
    BIN_DIR="${HOME}/go/bin"
else
    BIN_DIR="/usr/local/bin"  # Default for macOS homebrew-style installs
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

    # macOS-specific man path detection
    find_man_dir() {
        # 1. Check homebrew location
        brew_man_dir="/usr/local/share/man"
        if [ -d "$brew_man_dir" ]; then
            echo "$brew_man_dir"
            return
        fi

        # 2. Check XDG location
        xdg_man_dir="${HOME}/.local/share/man"
        if [ -d "$xdg_man_dir" ]; then
            echo "$xdg_man_dir"
            return
        fi

        # 3. Check manpath output
        if command -v manpath >/dev/null 2>&1; then
            manpath_output=$(manpath 2>/dev/null)
            old_ifs=$IFS
            IFS=:
            for path in $manpath_output; do
                case "$path" in
                    "$HOME"/* | /usr/local*)
                        echo "$path"
                        IFS=$old_ifs
                        return
                        ;;
                esac
            done
            IFS=$old_ifs
        fi

        # 4. Fallback to homebrew-style location
        mkdir -p "${brew_man_dir}/man1"
        echo "$brew_man_dir"
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
        case "$section" in
            '' | *[!0-9]*)
                echo " [ERROR: Invalid section number]" >&2
                continue
                ;;
        esac

        dest_dir="${MAN_DIR}/man${section}"
        mkdir -p "$dest_dir"
        
        if cp "$manpage" "$dest_dir/"; then
            echo " [OK]"
        else
            echo " [FAILED]" >&2
        fi
    done
    
    # Update man database if available (not typically needed on macOS)
    if command -v mandb >/dev/null 2>&1; then
        echo "Updating man database..."
        mandb -q "$MAN_DIR" >/dev/null 2>&1 || true
    fi
fi

echo "Installation complete"
