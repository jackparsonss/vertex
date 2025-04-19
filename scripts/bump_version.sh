#!/bin/bash

# Usage: ./scripts/bump_version.sh [major|minor|patch]

set -e
latest_tag=$(git describe --tags --abbrev=0)
version="${latest_tag#v}"

IFS='.' read -r major minor patch <<<"$version"

part=$1

if [[ -z "$part" ]]; then
  echo "Usage: $0 [major|minor|patch]"
  exit 1
fi

case "$part" in
major)
  ((major++))
  minor=0
  patch=0
  ;;
minor)
  ((minor++))
  patch=0
  ;;
patch)
  ((patch++))
  ;;
*)
  echo "Invalid part to increment. Use major, minor, or patch."
  exit 1
  ;;
esac

new_tag="v$major.$minor.$patch"

git tag "$new_tag"
git push origin "$new_tag"

echo "Tagged and pushed: $new_tag"
