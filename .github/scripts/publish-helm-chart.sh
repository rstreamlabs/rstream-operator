#!/usr/bin/env bash

set -euo pipefail

if [[ $# -ne 3 ]]; then
  echo "usage: $0 <chart-package> <oci-repository> <version>" >&2
  exit 1
fi

chart=$1
repository=$2
version=$3
work_directory=$(mktemp -d)
trap 'rm -rf "$work_directory"' EXIT

verify_chart() {
  rm -f "${work_directory}/rstream-operator-${version}.tgz"
  helm pull "${repository}/rstream-operator" --version "$version" --destination "$work_directory"
  if ! cmp --silent "$chart" "${work_directory}/rstream-operator-${version}.tgz"; then
    echo "published Helm chart differs from candidate version ${version}" >&2
    exit 1
  fi
}

if helm pull "${repository}/rstream-operator" --version "$version" --destination "$work_directory" 2>/dev/null; then
  if ! cmp --silent "$chart" "${work_directory}/rstream-operator-${version}.tgz"; then
    echo "immutable Helm chart conflict for version ${version}" >&2
    exit 1
  fi
  exit 0
fi

helm push "$chart" "$repository"
verify_chart
