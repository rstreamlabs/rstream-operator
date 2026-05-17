# syntax=docker/dockerfile:1.7
# See LICENSE file in the project root for license information.

FROM --platform=$BUILDPLATFORM golang:1.26.3-alpine AS builder

ARG TARGETOS
ARG TARGETARCH
ARG TARGETVARIANT

WORKDIR /workspace

RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN set -eux; \
    goos="${TARGETOS:-linux}"; \
    goarch="${TARGETARCH:-$(go env GOARCH)}"; \
    if [ "${goarch}" = "arm" ] && [ -n "${TARGETVARIANT:-}" ]; then export GOARM="${TARGETVARIANT#v}"; fi; \
    if [ "${goarch}" = "amd64" ] && [ -n "${TARGETVARIANT:-}" ]; then export GOAMD64="${TARGETVARIANT}"; fi; \
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build -mod=mod -buildvcs=false -trimpath -ldflags="-s -w" -o /manager ./cmd/manager; \
    CGO_ENABLED=0 GOOS="${goos}" GOARCH="${goarch}" go build -mod=mod -buildvcs=false -trimpath -ldflags="-s -w" -o /rstream-agent ./cmd/rstream-agent

FROM alpine:3.22

RUN apk add --no-cache ca-certificates && \
    addgroup -S -g 65532 rstream && \
    adduser -S -D -H -u 65532 -G rstream -h /home/rstream rstream

COPY --from=builder /manager /manager
COPY --from=builder /rstream-agent /rstream-agent

USER 65532:65532
WORKDIR /home/rstream

ENTRYPOINT ["/manager"]
