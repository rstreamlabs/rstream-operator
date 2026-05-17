#!make
# See LICENSE file in the project root for license information.

SHELL := /bin/bash

.DEFAULT_GOAL := all

IMAGE ?= rstream/rstream-operator:dev
AGENT_IMAGE ?= $(IMAGE)
KIND_CLUSTER ?= rstream-operator
DOCKER_BUILD_FLAGS ?=

LOCALBIN ?= $(shell pwd)/bin
$(LOCALBIN):
	mkdir -p $(LOCALBIN)

CONTROLLER_GEN ?= $(LOCALBIN)/controller-gen
KUSTOMIZE ?= $(LOCALBIN)/kustomize
ENVTEST ?= $(LOCALBIN)/setup-envtest

CONTROLLER_TOOLS_VERSION ?= v0.21.0
KUSTOMIZE_VERSION ?= v5.8.1
ENVTEST_VERSION ?= v0.24.1
ENVTEST_K8S_VERSION ?= 1.36

.PHONY: all
all: generate manifests helm-crds test

.PHONY: controller-gen
controller-gen: $(CONTROLLER_GEN)
$(CONTROLLER_GEN): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-tools/cmd/controller-gen@$(CONTROLLER_TOOLS_VERSION)

.PHONY: kustomize
kustomize: $(KUSTOMIZE)
$(KUSTOMIZE): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/kustomize/kustomize/v5@$(KUSTOMIZE_VERSION)

.PHONY: envtest
envtest: $(ENVTEST)
$(ENVTEST): $(LOCALBIN)
	GOBIN=$(LOCALBIN) go install sigs.k8s.io/controller-runtime/tools/setup-envtest@$(ENVTEST_VERSION)

.PHONY: generate
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

.PHONY: manifests
manifests: controller-gen
	$(CONTROLLER_GEN) rbac:roleName=manager-role crd webhook paths="./..." output:crd:artifacts:config=config/crd/bases output:rbac:artifacts:config=config/rbac

.PHONY: helm-crds
helm-crds: manifests
	mkdir -p charts/rstream-operator/crds
	cp config/crd/bases/*.yaml charts/rstream-operator/crds/

.PHONY: fmt
fmt:
	go fmt ./...

.PHONY: vet
vet:
	go vet ./...

.PHONY: test
test: manifests helm-crds generate fmt vet envtest
	KUBEBUILDER_ASSETS="$$( $(ENVTEST) use $(ENVTEST_K8S_VERSION) --bin-dir $(LOCALBIN) -p path )" go test ./...

.PHONY: build
build: generate fmt vet
	CGO_ENABLED=0 go build -buildvcs=false -o bin/manager ./cmd/manager
	CGO_ENABLED=0 go build -buildvcs=false -o bin/rstream-agent ./cmd/rstream-agent

.PHONY: run
run: manifests generate fmt vet
	go run ./cmd/manager --agent-image=$(AGENT_IMAGE)

.PHONY: docker-build
docker-build:
	docker build $(DOCKER_BUILD_FLAGS) -t $(IMAGE) .

.PHONY: docker-push
docker-push:
	docker push $(IMAGE)

.PHONY: install
install: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl apply -f -

.PHONY: uninstall
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd | kubectl delete --ignore-not-found=true -f -

.PHONY: deploy
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=$(IMAGE)
	$(KUSTOMIZE) build config/default | kubectl apply -f -
	kubectl -n rstream-system set env deployment/rstream-operator-controller-manager RSTREAM_AGENT_IMAGE=$(AGENT_IMAGE)

.PHONY: undeploy
undeploy: kustomize
	$(KUSTOMIZE) build config/default | kubectl delete --ignore-not-found=true -f -

.PHONY: helm-lint
helm-lint:
	helm lint charts/rstream-operator
