#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
RENDERER="${ROOT_DIR}/hack/render-runtime-smoke-sample.sh"
OUTPUT="$(${RENDERER} release-gate)"

if [[ "$(grep -c 'release-gate' <<<"${OUTPUT}")" -ne 3 ]]; then
  echo "custom namespace was not applied to every sample resource" >&2
  exit 1
fi
if grep -q 'rstream-demo' <<<"${OUTPUT}"; then
  echo "default namespace leaked into the rendered sample" >&2
  exit 1
fi
if "${RENDERER}" 'unsafe/value' >/dev/null 2>&1; then
  echo "unsafe namespace was accepted" >&2
  exit 1
fi
