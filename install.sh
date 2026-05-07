#!/bin/sh
# VettCode Scanner installer
# Usage: curl -sSfL https://get.vettcode.com | sh
#
# Environment variables:
#   VETTCODE_INSTALL_DIR  - installation directory (default: /usr/local/bin)
#   VETTCODE_VERSION      - specific version to install (default: latest)

set -e

GITHUB_REPO="vettcode/scanner"
BINARY_NAME="vettcode"
DEFAULT_INSTALL_DIR="/usr/local/bin"

main() {
    echo ""
    echo "  VettCode Scanner Installer"
    echo ""

    install_dir="${VETTCODE_INSTALL_DIR:-$DEFAULT_INSTALL_DIR}"

    os="$(detect_os)"
    arch="$(detect_arch)"

    if [ -z "$os" ] || [ -z "$arch" ]; then
        echo "Error: unsupported platform: $(uname -s)/$(uname -m)"
        echo "Download manually from https://github.com/${GITHUB_REPO}/releases"
        exit 1
    fi

    # Validate OS/arch combination has a published build
    validate_platform "$os" "$arch"
    echo "Detected platform: ${os}/${arch}"

    version="${VETTCODE_VERSION:-$(fetch_latest_version)}"
    if [ -z "$version" ]; then
        echo "Error: could not determine latest version."
        echo "If rate-limited by GitHub, set VETTCODE_VERSION=v1.x.x manually."
        exit 1
    fi

    # Strip leading 'v' for archive name
    version_num="${version#v}"

    if [ "$os" = "windows" ]; then
        archive="${BINARY_NAME}_${version_num}_${os}_${arch}.zip"
        bin_name="${BINARY_NAME}.exe"
    else
        archive="${BINARY_NAME}_${version_num}_${os}_${arch}.tar.gz"
        bin_name="${BINARY_NAME}"
    fi

    download_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/${archive}"
    checksum_url="https://github.com/${GITHUB_REPO}/releases/download/${version}/checksums.txt"

    tmpdir="$(mktemp -d)"
    trap 'rm -rf "$tmpdir"' EXIT

    echo "Downloading ${BINARY_NAME} ${version}..."
    download "$download_url" "$tmpdir/$archive"
    download "$checksum_url" "$tmpdir/checksums.txt"

    echo "Verifying checksum..."
    verify_checksum "$tmpdir/$archive" "$tmpdir/checksums.txt" "$archive"

    echo "Extracting..."
    extract "$tmpdir/$archive" "$tmpdir" "$os"

    echo "Installing to ${install_dir}..."
    install_binary "$tmpdir/$bin_name" "$install_dir" "$bin_name"

    echo ""
    echo "${BINARY_NAME} ${version} installed to ${install_dir}/${bin_name}"

    # Post-install verification
    if command -v "$BINARY_NAME" >/dev/null 2>&1; then
        echo ""
        "$BINARY_NAME" version
        echo ""
        echo "Ready! Run 'vettcode scan ./your-repo' to get started."
    else
        echo ""
        echo "Note: ${install_dir} may not be in your PATH."
        echo "Add it with:  export PATH=\"${install_dir}:\$PATH\""
    fi
}

detect_os() {
    case "$(uname -s)" in
        Darwin)  echo "darwin"  ;;
        Linux)   echo "linux"   ;;
        MINGW*|MSYS*|CYGWIN*) echo "windows" ;;
        *)       echo ""        ;;
    esac
}

detect_arch() {
    case "$(uname -m)" in
        x86_64|amd64)  echo "amd64" ;;
        arm64|aarch64) echo "arm64" ;;
        *)             echo ""      ;;
    esac
}

validate_platform() {
    os="$1"
    arch="$2"
    # Only these OS/arch combos have published builds
    case "${os}/${arch}" in
        darwin/amd64|darwin/arm64|linux/amd64|windows/amd64)
            return 0 ;;
        *)
            echo "Error: no pre-built binary for ${os}/${arch}"
            echo "Available platforms: darwin/amd64, darwin/arm64, linux/amd64, windows/amd64"
            echo "Download manually from https://github.com/${GITHUB_REPO}/releases"
            exit 1 ;;
    esac
}

fetch_latest_version() {
    # Try redirect-based detection first (no API rate limit)
    if command -v curl >/dev/null 2>&1; then
        tag="$(curl -sSfL -o /dev/null -w '%{redirect_url}' \
            "https://github.com/${GITHUB_REPO}/releases/latest" 2>/dev/null \
            | grep -oE 'v[0-9]+\.[0-9]+\.[0-9]+[^ ]*$' || true)"
        if [ -n "$tag" ]; then
            echo "$tag"
            return
        fi
    fi

    # Fallback: GitHub API (may be rate-limited for unauthenticated requests)
    api_url="https://api.github.com/repos/${GITHUB_REPO}/releases/latest"
    if command -v curl >/dev/null 2>&1; then
        curl -sSfL "$api_url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    elif command -v wget >/dev/null 2>&1; then
        wget -qO- "$api_url" 2>/dev/null | grep '"tag_name"' | sed -E 's/.*"tag_name": *"([^"]+)".*/\1/'
    else
        echo ""
    fi
}

download() {
    url="$1"
    dest="$2"
    if command -v curl >/dev/null 2>&1; then
        curl -sSfL -o "$dest" "$url"
    elif command -v wget >/dev/null 2>&1; then
        wget -qO "$dest" "$url"
    else
        echo "Error: curl or wget required"
        exit 1
    fi
}

verify_checksum() {
    archive="$1"
    checksums_file="$2"
    archive_name="$3"

    # Anchor match to end of line to avoid substring matches
    expected="$(grep " ${archive_name}$" "$checksums_file" | awk '{print $1}')"
    if [ -z "$expected" ]; then
        echo "Error: checksum not found for $archive_name"
        exit 1
    fi

    if command -v sha256sum >/dev/null 2>&1; then
        actual="$(sha256sum "$archive" | awk '{print $1}')"
    elif command -v shasum >/dev/null 2>&1; then
        actual="$(shasum -a 256 "$archive" | awk '{print $1}')"
    else
        echo "Warning: no sha256 tool found, skipping checksum verification"
        return 0
    fi

    if [ "$expected" != "$actual" ]; then
        echo "Error: checksum mismatch"
        echo "  expected: $expected"
        echo "  actual:   $actual"
        exit 1
    fi
}

extract() {
    archive="$1"
    dest="$2"
    os="$3"

    if [ "$os" = "windows" ]; then
        unzip -q "$archive" -d "$dest"
    else
        tar -xzf "$archive" -C "$dest"
    fi
}

install_binary() {
    src="$1"
    dest_dir="$2"
    bin_name="$3"

    if [ ! -d "$dest_dir" ]; then
        mkdir -p "$dest_dir"
    fi

    if [ -w "$dest_dir" ]; then
        cp "$src" "$dest_dir/$bin_name"
        chmod +x "$dest_dir/$bin_name"
    else
        if command -v sudo >/dev/null 2>&1; then
            echo "Elevated permissions required to install to $dest_dir"
            sudo cp "$src" "$dest_dir/$bin_name"
            sudo chmod +x "$dest_dir/$bin_name"
        else
            echo "Error: cannot write to $dest_dir and sudo is not available."
            echo "Set VETTCODE_INSTALL_DIR to a writable directory, e.g.:"
            echo "  VETTCODE_INSTALL_DIR=\$HOME/.local/bin curl -sSfL https://get.vettcode.com | sh"
            exit 1
        fi
    fi
}

main
