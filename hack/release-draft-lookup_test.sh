#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

ROOT_DIR=$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)
SELECTOR="${ROOT_DIR}/.github/scripts/select-release-by-tag.sh"
PAGES='[{"id":11,"tag_name":"v0.5.2","draft":false}]
[{"id":12,"tag_name":"v0.6.0","draft":true}]'
SELECTED=$(printf '%s\n' "$PAGES" | "$SELECTOR" v0.6.0)
jq --exit-status '.id == 12 and .draft == true' <<<"$SELECTED" >/dev/null

if printf '%s\n' "$PAGES" | "$SELECTOR" v9.9.9 >/dev/null 2>&1; then
  echo "A missing release tag must fail closed." >&2
  exit 1
fi
DUPLICATES='[{"id":12,"tag_name":"v0.6.0","draft":true}]
[{"id":13,"tag_name":"v0.6.0","draft":true}]'
if printf '%s\n' "$DUPLICATES" | "$SELECTOR" v0.6.0 >/dev/null 2>&1; then
  echo "Duplicate release tags must fail closed." >&2
  exit 1
fi
if printf '%s\n' 'not JSON' | "$SELECTOR" v0.6.0 >/dev/null 2>&1; then
  echo "Malformed release metadata must fail closed." >&2
  exit 1
fi

WORKFLOW="${ROOT_DIR}/.github/workflows/promote-release.yml"
LOOKUP=$(sed -n '/name: Find matching release candidate/,/name: Restore release candidate/p' "$WORKFLOW")
grep -F 'releases?per_page=100' <<<"$LOOKUP" >/dev/null
# These are literal workflow expressions, not shell expansions in this test.
# shellcheck disable=SC2016
grep -F 'ref: ${{ github.event.repository.default_branch }}' "$WORKFLOW" >/dev/null
# shellcheck disable=SC2016
grep -F './.promotion-policy/.github/scripts/select-release-by-tag.sh "$RELEASE_TAG"' <<<"$LOOKUP" >/dev/null
if grep -F 'releases/tags/' <<<"$LOOKUP" >/dev/null; then
  echo "Draft lookup must not use the published-only tag endpoint." >&2
  exit 1
fi
