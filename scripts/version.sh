#!/bin/bash
# Version management script for Design Tokens Language Server
# Updates version across VSCode extension, Zed extension, and creates git tag
#
# Inspired by CEM's version management approach
#
# Usage: ./scripts/version.sh <version>
# Example: ./scripts/version.sh 0.0.30

set -e

if [ -z "$1" ]; then
  echo "Usage: $0 <version>"
  echo "Example: $0 0.0.30"
  exit 1
fi

VERSION="$1"
# Remove 'v' prefix if present
VERSION="${VERSION#v}"

echo "Updating version to: $VERSION"

# Update VSCode extension version
echo "Updating extensions/vscode/package.json..."
if command -v jq &> /dev/null; then
  # Use jq if available (preferred for correctness)
  jq ".version = \"$VERSION\"" extensions/vscode/package.json > extensions/vscode/package.json.tmp
  mv extensions/vscode/package.json.tmp extensions/vscode/package.json
elif command -v node &> /dev/null; then
  # Use Node.js if available (most portable on developer machines)
  node -e "
    const fs = require('fs');
    const pkg = JSON.parse(fs.readFileSync('extensions/vscode/package.json', 'utf8'));
    pkg.version = '$VERSION';
    fs.writeFileSync('extensions/vscode/package.json', JSON.stringify(pkg, null, 2) + '\n');
  "
else
  # Fallback to sed (fragile but works for basic cases)
  sed -i "s/\"version\": \".*\"/\"version\": \"$VERSION\"/" extensions/vscode/package.json
fi

# Update Zed extension version
echo "Updating extensions/zed/extension.toml..."
if [[ "$OSTYPE" == "darwin"* ]]; then
  # macOS requires '' after -i
  sed -i '' "s/^version = \".*\"/version = \"$VERSION\"/" extensions/zed/extension.toml
else
  sed -i "s/^version = \".*\"/version = \"$VERSION\"/" extensions/zed/extension.toml
fi

# Show changes
echo ""
echo "Version updated in:"
echo "  - extensions/vscode/package.json"
echo "  - extensions/zed/extension.toml"
echo ""
echo "Changes:"
git diff extensions/vscode/package.json extensions/zed/extension.toml

# Check if there are changes
if ! git diff --quiet extensions/vscode/package.json extensions/zed/extension.toml; then
  echo ""
  read -p "Commit version changes? (y/n) " -n 1 -r
  echo
  if [[ $REPLY =~ ^[Yy]$ ]]; then
    git add extensions/vscode/package.json extensions/zed/extension.toml
    git commit -m "chore: prepare version $VERSION"
    echo "✓ Version changes committed"
    echo ""
    echo "Next steps:"
    echo "  make release v$VERSION  (to tag, push, and create GitHub release)"
  else
    echo "Version changes rejected by user."
    # Discard changes
    git checkout -- extensions/vscode/package.json extensions/zed/extension.toml
    exit 1
  fi
else
  echo ""
  echo "No changes detected. Version might already be $VERSION"
fi
