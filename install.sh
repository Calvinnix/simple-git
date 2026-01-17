#!/bin/bash
set -e

BINARY_NAME="simple-git"
INSTALL_DIR="${INSTALL_DIR:-$HOME/.local/bin}"

echo "Building $BINARY_NAME..."
go build -o "$BINARY_NAME" .

echo "Installing to $INSTALL_DIR..."
mkdir -p "$INSTALL_DIR"
mv "$BINARY_NAME" "$INSTALL_DIR/"

if [[ ":$PATH:" != *":$INSTALL_DIR:"* ]]; then
    echo ""
    echo "Note: $INSTALL_DIR is not in your PATH."
    echo "Add this to your shell config:"
    echo "  export PATH=\"\$PATH:$INSTALL_DIR\""
fi

echo "Done! Run '$BINARY_NAME' to use."
