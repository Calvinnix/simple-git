#!/bin/bash
set -e

VERSION=$(grep 'const version' main.go | sed 's/.*"\(.*\)".*/\1/')
BINARY_NAME="simple-git"
DIST_DIR="dist"
FORMULA_PATH="$HOME/dev/homebrew-tap/Formula/simple-git.rb"

echo "Building $BINARY_NAME v$VERSION"
echo

rm -rf "$DIST_DIR"
mkdir -p "$DIST_DIR"

declare -A checksums

platforms=(
    "darwin/arm64"
    "darwin/amd64"
    "linux/arm64"
    "linux/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS="${platform%/*}"
    GOARCH="${platform#*/}"
    output="$BINARY_NAME-$GOOS-$GOARCH"

    echo "Building $output..."
    GOOS=$GOOS GOARCH=$GOARCH go build -o "$DIST_DIR/$BINARY_NAME"
    tar -czvf "$DIST_DIR/$output.tar.gz" -C "$DIST_DIR" "$BINARY_NAME" > /dev/null
    rm "$DIST_DIR/$BINARY_NAME"

    checksums[$output]=$(sha256sum "$DIST_DIR/$output.tar.gz" | cut -d' ' -f1)
done

echo
echo "SHA256 checksums:"
for key in "${!checksums[@]}"; do
    echo "  $key: ${checksums[$key]}"
done

cat > "$FORMULA_PATH" << EOF
class SimpleGit < Formula
  desc "Lightweight Git TUI"
  homepage "https://github.com/Calvinnix/simple-git"
  version "$VERSION"
  license "MIT"

  on_macos do
    on_arm do
      url "https://github.com/Calvinnix/simple-git/releases/download/v$VERSION/$BINARY_NAME-darwin-arm64.tar.gz"
      sha256 "${checksums[simple-git-darwin-arm64]}"
    end
    on_intel do
      url "https://github.com/Calvinnix/simple-git/releases/download/v$VERSION/$BINARY_NAME-darwin-amd64.tar.gz"
      sha256 "${checksums[simple-git-darwin-amd64]}"
    end
  end

  on_linux do
    on_arm do
      url "https://github.com/Calvinnix/simple-git/releases/download/v$VERSION/$BINARY_NAME-linux-arm64.tar.gz"
      sha256 "${checksums[simple-git-linux-arm64]}"
    end
    on_intel do
      url "https://github.com/Calvinnix/simple-git/releases/download/v$VERSION/$BINARY_NAME-linux-amd64.tar.gz"
      sha256 "${checksums[simple-git-linux-amd64]}"
    end
  end

  def install
    bin.install "simple-git"
  end

  test do
    assert_match "simple-git version", shell_output("#{bin}/simple-git --version")
  end
end
EOF

echo
echo "Updated $FORMULA_PATH"
echo
echo "Next steps:"
echo "  1. Upload dist/*.tar.gz to GitHub release v$VERSION"
echo "  2. cd ~/dev/homebrew-tap && git add -A && git commit -m 'Update simple-git to v$VERSION' && git push"
