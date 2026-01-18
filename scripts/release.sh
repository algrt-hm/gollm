#!/bin/bash

# Release script: increments the version tag and pushes to trigger a GitHub release

set -e

# Get the latest tag
LATEST_TAG=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")

echo "Latest tag: $LATEST_TAG"

# Parse version components (strip leading 'v')
VERSION="${LATEST_TAG#v}"
IFS='.' read -r MAJOR MINOR PATCH <<< "$VERSION"

# Default to 0 if parsing fails
MAJOR=${MAJOR:-0}
MINOR=${MINOR:-0}
PATCH=${PATCH:-0}

# Increment patch version by default
NEW_PATCH=$((PATCH + 1))
NEW_TAG="v${MAJOR}.${MINOR}.${NEW_PATCH}"

echo "New tag: $NEW_TAG"
echo ""

# Confirm with user
read -p "Create and push tag $NEW_TAG? (y/n) " -n 1 -r
echo ""

if [[ $REPLY =~ ^[Yy]$ ]]; then
    git tag "$NEW_TAG"
    echo "Created tag $NEW_TAG"

    git push origin "$NEW_TAG"
    echo "Pushed tag $NEW_TAG to origin"
    echo ""
    echo "GitHub Actions will now build and create a release."
else
    echo "Aborted."
    exit 1
fi
