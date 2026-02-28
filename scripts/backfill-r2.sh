#!/usr/bin/env bash
set -euo pipefail

REPO="amikos-tech/chroma-go-local"
WORKFLOW="release.yml"
WORKFLOW_REF="${WORKFLOW_REF:-main}"
DEFAULT_TAGS=("v0.1.0" "v0.2.0" "v0.3.0")

usage() {
  cat <<EOF
Usage:
  $(basename "$0") [--workflow-ref <ref>] [--repo <owner/name>] [--workflow <name>] [tag...]

Dispatches release backfill runs for historical tags using the latest workflow definition.
Defaults:
  workflow-ref: ${WORKFLOW_REF}
  repo:         ${REPO}
  workflow:     ${WORKFLOW}
  tags:         ${DEFAULT_TAGS[*]}
EOF
}

if ! command -v gh >/dev/null 2>&1; then
  echo "gh CLI is required" >&2
  exit 1
fi

if ! gh auth status >/dev/null 2>&1; then
  echo "gh authentication is required" >&2
  exit 1
fi

tags=()
while [ "$#" -gt 0 ]; do
  case "$1" in
    --workflow-ref)
      if [ "$#" -lt 2 ]; then
        echo "Missing value for --workflow-ref" >&2
        usage
        exit 1
      fi
      WORKFLOW_REF="$2"
      shift 2
      ;;
    --repo)
      if [ "$#" -lt 2 ]; then
        echo "Missing value for --repo" >&2
        usage
        exit 1
      fi
      REPO="$2"
      shift 2
      ;;
    --workflow)
      if [ "$#" -lt 2 ]; then
        echo "Missing value for --workflow" >&2
        usage
        exit 1
      fi
      WORKFLOW="$2"
      shift 2
      ;;
    -h|--help)
      usage
      exit 0
      ;;
    --)
      shift
      while [ "$#" -gt 0 ]; do
        tags+=("$1")
        shift
      done
      ;;
    *)
      tags+=("$1")
      shift
      ;;
  esac
done

if [ "${#tags[@]}" -eq 0 ]; then
  tags=("${DEFAULT_TAGS[@]}")
fi

echo "Triggering ${WORKFLOW} on ${REPO} using workflow ref '${WORKFLOW_REF}' for tags: ${tags[*]}"
for tag in "${tags[@]}"; do
  echo "-> dispatch ${WORKFLOW} at workflow ref ${WORKFLOW_REF} for release_tag=${tag}"
  gh workflow run "${WORKFLOW}" \
    --repo "${REPO}" \
    --ref "${WORKFLOW_REF}" \
    -f release_tag="${tag}"
done

echo
echo "Backfill workflows were dispatched."
echo "Track runs with: gh run list --repo ${REPO} --workflow ${WORKFLOW} --limit 20"
