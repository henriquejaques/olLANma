#!/bin/sh
set -e

# olLANma 1-Line Installer Script
# This script downloads and installs the latest olLANma binary for Linux.

REPO="henriquejaques/olLANma"
INSTALL_DIR="/usr/local/bin"
BINARY_NAME="olLANma"

echo "=== olLANma Installer ==="

# Check for curl or wget
if command -v curl >/dev/null 2>&1; then
    DOWNLOADER="curl -fsSL"
elif command -v wget >/dev/null 2>&1; then
    DOWNLOADER="wget -qO-"
else
    echo "Error: curl or wget is required to download the binary."
    exit 1
fi

# Detect system architecture
ARCH="$(uname -m)"
case "$ARCH" in
    x86_64)  ARCH_SUFFIX="amd64" ;;
    aarch64) ARCH_SUFFIX="arm64" ;;
    armv7l)  ARCH_SUFFIX="armv7" ;;
    *)
        echo "Error: Unsupported architecture: $ARCH"
        echo "Supported: x86_64 (amd64), aarch64 (arm64), armv7l (armv7)"
        exit 1
        ;;
esac

echo "Detected architecture: $ARCH ($ARCH_SUFFIX)"

DOWNLOAD_URL="https://github.com/${REPO}/releases/latest/download/olLANma-linux-${ARCH_SUFFIX}"
CHECKSUM_URL="https://github.com/${REPO}/releases/latest/download/checksums.txt"

# Create secure temporary files (M-004: prevents symlink attacks)
tmp_file="$(mktemp /tmp/ollanma_download.XXXXXXXXXX)"
tmp_checksums="$(mktemp /tmp/ollanma_checksums.XXXXXXXXXX)"

# Clean up temp files on exit (success or failure)
trap 'rm -f "$tmp_file" "$tmp_checksums"' EXIT

echo "Downloading the latest version of olLANma..."
$DOWNLOADER "$DOWNLOAD_URL" > "$tmp_file"

# Verify integrity via SHA256 checksum (fail-closed: M-003)
echo "Verifying download integrity..."
if ! $DOWNLOADER "$CHECKSUM_URL" > "$tmp_checksums" 2>/dev/null; then
    echo "Error: Could not download checksums.txt for integrity verification."
    echo "Aborting installation. To install without verification, download manually."
    exit 1
fi

if [ ! -s "$tmp_checksums" ]; then
    echo "Error: Downloaded checksums.txt is empty."
    echo "Aborting installation."
    exit 1
fi

EXPECTED_HASH=$(grep "olLANma-linux-${ARCH_SUFFIX}" "$tmp_checksums" | awk '{print $1}')
if [ -z "$EXPECTED_HASH" ]; then
    echo "Error: Could not find checksum for olLANma-linux-${ARCH_SUFFIX} in checksums.txt."
    echo "Aborting installation."
    exit 1
fi

ACTUAL_HASH=$(sha256sum "$tmp_file" | awk '{print $1}')
if [ "$EXPECTED_HASH" != "$ACTUAL_HASH" ]; then
    echo "Error: Checksum verification FAILED."
    echo "  Expected: $EXPECTED_HASH"
    echo "  Got:      $ACTUAL_HASH"
    echo "The downloaded binary may have been tampered with. Aborting."
    exit 1
fi
echo "Checksum verified successfully."

echo "Making binary executable..."
chmod +x "$tmp_file"

echo "Installing to $INSTALL_DIR (requires sudo)..."
if [ "$(id -u)" != "0" ]; then
    sudo mv "$tmp_file" "$INSTALL_DIR/$BINARY_NAME"
else
    mv "$tmp_file" "$INSTALL_DIR/$BINARY_NAME"
fi

echo "=== Installation Complete! ==="
echo "You can now run 'olLANma config' to set up your LAN instance."
