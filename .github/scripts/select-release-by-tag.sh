#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

if (($# != 1)); then
  printf 'usage: %s <release-tag>\n' "$0" >&2
  exit 2
fi

tag=$1
jq --exit-status --slurp --arg tag "$tag" '
  map(.[]) |
  map(select(.tag_name == $tag)) |
  if length == 1 then
    .[0]
  else
    error("expected exactly one release for tag " + $tag)
  end
'
