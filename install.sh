#!/bin/bash

OS=$(uname -s)
if [ "$OS" != "Linux" ]; then
    echo "This script only supports Linux/Android platforms."
    exit 1
fi

ARCH=$(uname -m)
case "$ARCH" in
    aarch64|arm64) ARCH="arm64" ;;
    armv7*|armv8*) ARCH="arm" ;;
    x86_64)        ARCH="amd64" ;;
    i386|i686)     ARCH="386" ;;
    *)             echo "Unsupported architecture: $ARCH" && exit 1 ;;
esac

rm -f BPB-Warp-Scanner.tar.gz BPB-Warp-Scanner-linux-*

BINARY="BPB-Warp-Scanner-linux-${ARCH}"
ARCHIVE="${BINARY}.tar.gz"

echo "Downloading ${ARCHIVE}..."
curl -L -# -o "$ARCHIVE" "https://github.com/bia-pain-bache/BPB-Warp-Scanner/releases/latest/download/${ARCHIVE}" && \
tar xzf "$ARCHIVE" && \
chmod +x "./${BINARY}" && \
chmod +x "./core/xray" && \
exec ./"${BINARY}"