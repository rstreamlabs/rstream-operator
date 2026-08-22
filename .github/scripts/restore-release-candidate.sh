#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <version> <archive>" >&2
  exit 1
fi

version=$1
archive=$2
candidate_parent=out/release-candidate
candidate_root="${candidate_parent}/${version}"

if [[ ! -f "$archive" || ! -f "${archive}.sha256" ]]; then
  echo "release candidate archive or checksum is missing" >&2
  exit 1
fi

archive_directory=$(cd "$(dirname "$archive")" && pwd)
archive_basename=$(basename "$archive")
(
  cd "$archive_directory"
  shasum -a 256 --check "${archive_basename}.sha256"
)
if ! tar -tzf "$archive" | awk -v prefix="${version}/" '
  $0 != prefix && index($0, prefix) != 1 { invalid = 1 }
  /(^|\/)\.\.($|\/)/ { invalid = 1 }
  END { exit invalid }
'; then
  echo "release candidate contains an invalid path" >&2
  exit 1
fi
if [[ -e "$candidate_root" ]]; then
  echo "release candidate already exists: ${candidate_root}" >&2
  exit 1
fi

mkdir -p "$candidate_parent"
tar -xzf "$archive" -C "$candidate_parent"
(
  cd "$candidate_root"
  shasum -a 256 --check release-manifest.sha256
)
