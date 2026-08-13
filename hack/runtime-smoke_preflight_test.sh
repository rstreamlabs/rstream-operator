#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
SMOKE="${ROOT_DIR}/hack/runtime-smoke.sh"

if RSTREAM_PROJECT_ENDPOINT=project RSTREAM_TOKEN=token RSTREAM_CONTROL_PLANE_HEADER_NAME=x-test "${SMOKE}" >/dev/null 2>&1; then
  echo "a control-plane header without a value was accepted" >&2
  exit 1
fi
if RSTREAM_PROJECT_ENDPOINT=project RSTREAM_TOKEN=token RSTREAM_CONTROL_PLANE_HEADER_NAME='unsafe:value' RSTREAM_CONTROL_PLANE_HEADER_VALUE=value "${SMOKE}" >/dev/null 2>&1; then
  echo "an invalid control-plane header name was accepted" >&2
  exit 1
fi
