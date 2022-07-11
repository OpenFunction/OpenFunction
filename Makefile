#
# Copyright 2022 The OpenFunction Authors.
#
# Licensed under the Apache License, Version 2.0 (the "License");
# you may not use this file except in compliance with the License.
# You may obtain a copy of the License at
#
#     http://www.apache.org/licenses/LICENSE-2.0
#
# Unless required by applicable law or agreed to in writing, software
# distributed under the License is distributed on an "AS IS" BASIS,
# WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
# See the License for the specific language governing permissions and
# limitations under the License.
#

VERSION?=$(shell cat VERSION | tr -d " \t\n\r")
# Image URL to use all building/pushing image targets
IMG ?= openfunction/openfunction:$(VERSION)
IMG_DEV ?= openfunctiondev/openfunction:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
#CRD_OPTIONS ?= "crd:trivialVersions=true,preserveUnknownFields=false"
CRD_OPTIONS ?= "crd:preserveUnknownFields=false,maxDescLen=0"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

# Setting SHELL to bash allows bash commands to be executed by recipes.
# This is a requirement for 'setup-envtest.sh' in the test target.
# Options are set to exit when a recipe line exits non-zero or a piped command fails.
SHELL = /usr/bin/env bash -o pipefail
.SHELLFLAGS = -ec

all: build test

##@ General

# The help target prints out all targets with their descriptions organized
# beneath their categories. The categories are represented by '##@' and the
# target descriptions by '##'. The awk commands is responsible for reading the
# entire set of makefiles included in this invocation, looking for lines of the
# file as xyz: ## something, and then pretty-format the target and help. Then,
# if there's a line with ##@ something, that gets pretty-printed as a category.
# More info on the usage of ANSI control characters for terminal formatting:
# https://en.wikipedia.org/wiki/ANSI_escape_code#SGR_parameters
# More info on the awk command:
# http://linuxcommand.org/lc3_adv_awk.php

help: ## Display this help.
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z_0-9-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##@ Development

manifests: generate fmt vet controller-gen ## Generate WebhookConfiguration, ClusterRole and CustomResourceDefinition objects.
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role paths="./..." output:crd:artifacts:config=config/crd/bases
	kubectl kustomize config/default | sed -e '/creationTimestamp: null/d' | sed -e 's/openfunction-system/openfunction/g' | sed -e 's/openfunction\:latest/openfunction\:$(VERSION)/g' | sed -e 's/app.kubernetes.io\/version\: latest/app.kubernetes.io\/version\: $(VERSION)/g' > config/bundle.yaml
	cat config/configmap/openfunction-config.yaml >> config/bundle.yaml

generate: controller-gen ## Generate code containing DeepCopy, DeepCopyInto, and DeepCopyObject method implementations.
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

fmt: goimports ## Run go fmt && goimports against code.
	go fmt ./...
	$(GOIMPORTS) -w -local github.com/openfunction main.go controllers/ pkg/ docs/ hack/ apis/

GOIMPORTS=$(shell pwd)/bin/goimports
goimports:
	$(call go-get-tool,$(GOIMPORTS),golang.org/x/tools/cmd/goimports@v0.1.7)

vet: ## Run go vet against code.
	go vet ./...

ENVTEST_ASSETS_DIR=$(shell pwd)/testbin
test: manifests ## Run tests.
	mkdir -p ${ENVTEST_ASSETS_DIR}
	test -f ${ENVTEST_ASSETS_DIR}/setup-envtest.sh || curl -sSLo ${ENVTEST_ASSETS_DIR}/setup-envtest.sh https://raw.githubusercontent.com/kubernetes-sigs/controller-runtime/v0.8.3/hack/setup-envtest.sh
	source ${ENVTEST_ASSETS_DIR}/setup-envtest.sh; fetch_envtest_tools $(ENVTEST_ASSETS_DIR); setup_envtest_env $(ENVTEST_ASSETS_DIR); go test ./pkg/... ./controllers/... -coverprofile cover.out

verify: verify-crds

verify-crds: manifests
	@if !(git diff --quiet HEAD config/crd); then \
		echo "generated files are out of date, run make generate"; exit 1; \
	fi

##@ Build

binary: ## Build openfunction binary without test.
	go build -o bin/openfunction main.go

build: generate fmt vet ## Build openfunction binary.
	go build -o bin/openfunction main.go

run: manifests ## Run a controller from your host.
	go run ./main.go

docker-build: all ## Build docker image with the openfunction.
	docker build -t ${IMG} .

docker-push: ## Push docker image with the openfunction.
	docker push ${IMG}

docker-build-dev: all ## Build dev docker image with the openfunction.
	docker build -t ${IMG_DEV} .

docker-push-dev: ## Push dev docker image with the openfunction.
	docker push ${IMG_DEV}

##@ Deployment

install: manifests kustomize ## Install CRDs into the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl create -f -

uninstall: manifests kustomize ## Uninstall CRDs from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/crd | kubectl delete -f -

deploy: manifests kustomize ## Deploy controller to the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl apply -f -

undeploy: ## Undeploy controller from the K8s cluster specified in ~/.kube/config.
	$(KUSTOMIZE) build config/default | kubectl delete -f -

dev:
	kubectl kustomize config/default | sed -e '/creationTimestamp/d' | sed -e 's/openfunction-system/openfunction/g' | sed -e 's/openfunction\/openfunction/openfunctiondev\/openfunction/g' > config/bundle.yaml
	kubectl kustomize config/samples/ | sed -e 's/openfunction\/sample-go-func/openfunctiondev\/sample-go-func/g' > config/samples/function-sample-dev.yaml

clean:
	git checkout config/bundle.yaml
	rm -rf config/samples/function-sample-dev.yaml
	docker rmi `docker image ls | sed -e 's/[ ][ ]*/\t/g' | cut -f 2,3 | grep none | cut -f 2 | tr "\n" " "`  2>/dev/null

##@ E2E test

e2e: skywalking-e2e yq
	$(E2E) run -c test/e2e.yaml

e2e-knative: skywalking-e2e yq
	$(E2E) run -c test/knative-runtime/e2e.yaml

e2e-async: skywalking-e2e yq
	$(E2E) run -c test/async-runtime/e2e.yaml

e2e-plugin: skywalking-e2e yq
	$(E2E) run -c test/plugin/e2e.yaml

e2e-events: skywalking-e2e yq
	$(E2E) run -c test/events/e2e.yaml

CONTROLLER_GEN = $(shell pwd)/bin/controller-gen
controller-gen: ## Download controller-gen locally if necessary.
	$(call go-get-tool,$(CONTROLLER_GEN),sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1)

KUSTOMIZE = $(shell pwd)/bin/kustomize
kustomize: ## Download kustomize locally if necessary.
	$(call go-get-tool,$(KUSTOMIZE),sigs.k8s.io/kustomize/kustomize/v4@v4.5.2)

E2E = $(shell pwd)/bin/e2e
skywalking-e2e: ## Download skywalking-e2e locally if necessary.
	$(call go-get-tool,$(E2E),github.com/apache/skywalking-infra-e2e/cmd/e2e@2a33478)

YQ = $(shell pwd)/bin/yq
yq: ## Download yq locally if necessary.
	$(call go-get-tool,$(YQ),github.com/mikefarah/yq/v4@latest)

# go-get-tool will 'go get' any package $2 and install it to $1.
PROJECT_DIR := $(shell dirname $(abspath $(lastword $(MAKEFILE_LIST))))
define go-get-tool
@[ -f $(1) ] || { \
set -e ;\
TMP_DIR=$$(mktemp -d) ;\
cd $$TMP_DIR ;\
go mod init tmp ;\
echo "Downloading $(2)" ;\
GOBIN=$(PROJECT_DIR)/bin go install -v $(2) ;\
rm -rf $$TMP_DIR ;\
}
endef
