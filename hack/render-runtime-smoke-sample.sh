#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
NAMESPACE="${1:-}"

if [[ -z "${NAMESPACE}" || ${#NAMESPACE} -gt 63 || ! "${NAMESPACE}" =~ ^[a-z0-9]([-a-z0-9]*[a-z0-9])?$ ]]; then
  echo "invalid Kubernetes namespace: ${NAMESPACE}" >&2
  exit 2
fi

sed "s/rstream-demo/${NAMESPACE}/g" "${ROOT_DIR}/config/samples/http_server.yaml"
