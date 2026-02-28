#!/usr/bin/env bash
set -euo pipefail

REPO="amikos-tech/chroma-go-local"
WORKFLOW="release.yml"
DEFAULT_TAGS=("v0.1.0" "v0.2.0" "v0.3.0")

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "gh authentication is required" >&2
  exit 1
fi

if [ "$#" -eq 0 ]; then
  tags=("${DEFAULT_TAGS[@]}")
else
  tags=("$@")
fi

echo "Triggering ${WORKFLOW} for tags: ${tags[*]}"
for tag in "${tags[@]}"; do
  echo "-> dispatch ${WORKFLOW} at ref ${tag}"
  gh workflow run "${WORKFLOW}" --repo "${REPO}" --ref "${tag}"
done

echo
echo "Backfill workflows were dispatched."
echo "Track runs with: gh run list --repo ${REPO} --workflow ${WORKFLOW} --limit 20"
