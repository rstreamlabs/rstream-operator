#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 2 ]]; then
  echo "usage: $0 <version> <archive>" >&2
  exit 1
fi

version=$1
archive=$2
candidate_root="out/release-candidate/${version}"
manifest="${candidate_root}/release-manifest.sha256"

if [[ ! "$version" =~ ^[0-9]+(\.[0-9]+){2}$ ]]; then
  echo "invalid release version: ${version}" >&2
  exit 1
fi
for required in \
  "${candidate_root}/rstream-operator-${version}.oci.tar" \
  "${candidate_root}/rstream-operator-${version}.tgz"; do
  if [[ ! -f "$required" ]]; then
    echo "release candidate file is missing: ${required}" >&2
    exit 1
  fi
done

(
  cd "$candidate_root"
  find . -type f ! -name release-manifest.sha256 -print0 \
    | LC_ALL=C sort -z \
    | xargs -0 shasum -a 256
) > "$manifest"

tar -czf "$archive" -C "$(dirname "$candidate_root")" "$version"
shasum -a 256 "$archive" > "${archive}.sha256"
