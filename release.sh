#!/bin/bash
set -e

BINARY_NAME="simple-git"
DIST_DIR="dist"
FORMULA_PATH="$HOME/dev/homebrew-tap/Formula/simple-git.rb"
REPO="Calvinnix/simple-git"

# Get current version from main.go
CURRENT_VERSION=$(grep 'const version' main.go | sed 's/.*"\(.*\)".*/\1/')

# Prompt for version
echo "Current version: $CURRENT_VERSION"
read -p "Enter version to release (press Enter for $CURRENT_VERSION): " INPUT_VERSION
VERSION="${INPUT_VERSION:-$CURRENT_VERSION}"

# Update main.go if version changed
if [ "$VERSION" != "$CURRENT_VERSION" ]; then
    echo "Updating version in main.go from $CURRENT_VERSION to $VERSION..."
    sed -i "s/const version = \"$CURRENT_VERSION\"/const version = \"$VERSION\"/" main.go
fi

echo
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

# Create git tag and push if it doesn't exist
if ! git rev-parse "v$VERSION" >/dev/null 2>&1; then
    echo "Creating and pushing tag v$VERSION..."
    git add main.go
    git commit -m "Release v$VERSION" --allow-empty
    git tag "v$VERSION"
    git push origin master
    git push origin "v$VERSION"
else
    echo "Tag v$VERSION already exists"
fi

# Create GitHub release and upload artifacts
echo
if gh release view "v$VERSION" --repo "$REPO" >/dev/null 2>&1; then
    echo "Release v$VERSION already exists, uploading assets..."
    gh release upload "v$VERSION" \
        --repo "$REPO" \
        --clobber \
        "$DIST_DIR"/*.tar.gz
else
    echo "Creating GitHub release v$VERSION..."
    gh release create "v$VERSION" \
        --repo "$REPO" \
        --title "v$VERSION" \
        --generate-notes \
        "$DIST_DIR"/*.tar.gz
fi

echo
echo "Release v$VERSION complete!"
echo "View at: https://github.com/$REPO/releases/tag/v$VERSION"

# Update homebrew-tap
echo
echo "Updating homebrew-tap..."
cd ~/dev/homebrew-tap
git add -A
git commit -m "Update simple-git to v$VERSION"
git push

echo
echo "Done! Homebrew formula updated."
