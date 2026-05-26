#!/usr/bin/env bash
# See LICENSE file in the project root for license information.

set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

KIND_CLUSTER="${KIND_CLUSTER:-rstream-operator}"
IMAGE_REPOSITORY="${IMAGE_REPOSITORY:-rstream-operator}"
IMAGE_TAG="${IMAGE_TAG:-runtime}"
IMAGE="${IMAGE_REPOSITORY}:${IMAGE_TAG}"
NAMESPACE="${NAMESPACE:-rstream-demo}"
TIMEOUT="${TIMEOUT:-180s}"
SMOKE_TMP_DIR=""

if [[ -z "${RSTREAM_ENGINE:-}" && -z "${RSTREAM_PROJECT_ENDPOINT:-}" && -z "${RSTREAM_PROJECT_ID:-}" ]]; then
  echo "Set RSTREAM_PROJECT_ENDPOINT, RSTREAM_PROJECT_ID, or RSTREAM_ENGINE." >&2
  exit 2
fi

if [[ -z "${RSTREAM_TOKEN:-}" ]]; then
  echo "RSTREAM_TOKEN is required. Export a project token before running the smoke test." >&2
  exit 2
fi

cd "${ROOT_DIR}"

if ! kind get clusters | grep -qx "${KIND_CLUSTER}"; then
  kind create cluster --name "${KIND_CLUSTER}"
fi

build_image() {
  local sdk_dir="${RSTREAM_GO_REPO:-${ROOT_DIR}/../rstream-go}"
  if [[ -f "${sdk_dir}/go.mod" ]]; then
    SMOKE_TMP_DIR="$(mktemp -d)"
    trap 'rm -rf "${SMOKE_TMP_DIR}"' EXIT
    mkdir -p "${SMOKE_TMP_DIR}/rstream-operator" "${SMOKE_TMP_DIR}/rstream-go"
    rsync -a --delete \
      --exclude .git \
      --exclude bin \
      --exclude dist \
      --exclude out \
      --exclude coverage.out \
      --exclude go.work \
      --exclude go.work.sum \
      "${ROOT_DIR}/" "${SMOKE_TMP_DIR}/rstream-operator/"
    rsync -a --delete \
      --exclude .git \
      --exclude out \
      "${sdk_dir}/" "${SMOKE_TMP_DIR}/rstream-go/"
    cat > "${SMOKE_TMP_DIR}/Dockerfile" <<'DOCKERFILE'
FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH

WORKDIR /workspace/rstream-operator

COPY rstream-go /workspace/rstream-go
COPY rstream-operator .

RUN go mod edit -replace github.com/rstreamlabs/rstream-go=/workspace/rstream-go && go mod download
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -buildvcs=false -trimpath -ldflags="-s -w" -o /manager ./cmd/manager
RUN CGO_ENABLED=0 GOOS=${TARGETOS:-linux} GOARCH=${TARGETARCH} go build -buildvcs=false -trimpath -ldflags="-s -w" -o /rstream-agent ./cmd/rstream-agent

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 65532 rstream && \
    adduser -S -D -H -u 65532 -G rstream -h /home/rstream rstream

COPY --from=builder /manager /manager
COPY --from=builder /rstream-agent /rstream-agent

USER 65532:65532
WORKDIR /home/rstream

ENTRYPOINT ["/manager"]
DOCKERFILE
    docker build -t "${IMAGE}" -f "${SMOKE_TMP_DIR}/Dockerfile" "${SMOKE_TMP_DIR}"
    return
  fi
  docker build -t "${IMAGE}" .
}

build_image
kind load docker-image "${IMAGE}" --name "${KIND_CLUSTER}"

helm upgrade --install rstream-operator ./charts/rstream-operator \
  --namespace rstream-system \
  --create-namespace \
  --set image.repository="${IMAGE_REPOSITORY}" \
  --set image.tag="${IMAGE_TAG}" \
  --set image.pullPolicy=IfNotPresent \
  --set agent.image.repository="${IMAGE_REPOSITORY}" \
  --set agent.image.tag="${IMAGE_TAG}"

kubectl -n rstream-system rollout restart deployment/rstream-operator
kubectl -n rstream-system rollout status deployment/rstream-operator --timeout="${TIMEOUT}"

kubectl apply -f config/samples/http_server.yaml
kubectl -n "${NAMESPACE}" create secret generic rstream-credentials \
  --from-literal=token="${RSTREAM_TOKEN}" \
  --dry-run=client -o yaml | kubectl apply -f -

connection_selector=""
if [[ -n "${RSTREAM_PROJECT_ENDPOINT:-}" ]]; then
  connection_selector="  projectEndpoint: ${RSTREAM_PROJECT_ENDPOINT}"
  if [[ -n "${RSTREAM_API_URL:-}" ]]; then
    connection_selector="${connection_selector}
  apiURL: ${RSTREAM_API_URL}"
  fi
elif [[ -n "${RSTREAM_PROJECT_ID:-}" ]]; then
  connection_selector="  projectID: ${RSTREAM_PROJECT_ID}"
  if [[ -n "${RSTREAM_API_URL:-}" ]]; then
    connection_selector="${connection_selector}
  apiURL: ${RSTREAM_API_URL}"
  fi
else
  connection_selector="  engine: ${RSTREAM_ENGINE}"
fi

cat <<EOF | kubectl apply -f -
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamConnection
metadata:
  name: default
  namespace: ${NAMESPACE}
spec:
${connection_selector}
  tokenSecretRef:
    name: rstream-credentials
    key: token
---
apiVersion: tunnels.rstream.io/v1alpha1
kind: RstreamTunnel
metadata:
  name: web
  namespace: ${NAMESPACE}
spec:
  target:
    service:
      name: http-server
      port: http
  publish: true
  protocol: http
  http:
    version: http/1.1
EOF

kubectl -n "${NAMESPACE}" wait rstreamtunnel/web --for=condition=Ready --timeout="${TIMEOUT}"

forwarding_address="$(kubectl -n "${NAMESPACE}" get rstreamtunnel web -o jsonpath='{.status.forwardingAddress}')"
if [[ -z "${forwarding_address}" ]]; then
  echo "RstreamTunnel is Ready but status.forwardingAddress is empty" >&2
  exit 1
fi

curl --fail --silent --show-error --retry 5 --retry-delay 2 "${forwarding_address}" >/dev/null

echo "Runtime smoke test passed: ${forwarding_address}"
