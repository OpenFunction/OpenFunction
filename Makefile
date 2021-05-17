
VERSION?=$(shell cat VERSION | tr -d " \t\n\r")
# Image URL to use all building/pushing image targets
IMG ?= openfunction/openfunction:$(VERSION)
IMG_DEV ?= openfunctiondev/openfunction:$(VERSION)
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
CRD_OPTIONS ?= "crd:trivialVersions=true"

# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif

all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build openfunction binary
manager: generate fmt vet
	go build -o bin/openfunction cmd/main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests
	kubectl kustomize config/crd | kubectl apply -f -

# Uninstall CRDs from a cluster
uninstall: manifests
	kubectl kustomize config/crd | kubectl delete -f -

# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests
	cd config/manager && kustomize edit set image controller=${IMG}
	kubectl kustomize config/default | sed -e '/creationTimestamp/d' | sed -e 's/openfunction-system/openfunction/g' | kubectl apply -f -

# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	kubectl kustomize config/default | sed -e '/creationTimestamp/d' | sed -e 's/openfunction-system/openfunction/g' | sed -e 's/\:latest/\:$(VERSION)/g' > config/bundle.yaml


# Run go fmt against code
fmt:
	go fmt ./...

# Run go vet against code
vet:
	go vet ./...

# Generate code
generate: controller-gen
	$(CONTROLLER_GEN) object:headerFile="hack/boilerplate.go.txt" paths="./..."

# Build the docker image
build: test
	docker build -f cmd/Dockerfile . -t ${IMG}

# Push the docker image
push:
	docker push ${IMG}

# Build the dev docker image
build-dev: test
	docker build -f cmd/Dockerfile . -t ${IMG_DEV}

# Push the dev docker image
push-dev:
	docker push ${IMG_DEV}

dev:
	kubectl kustomize config/default | sed -e '/creationTimestamp/d' | sed -e 's/openfunction-system/openfunction/g' | sed -e 's/openfunction\/openfunction/openfunctiondev\/openfunction/g' > config/bundle.yaml
	kubectl kustomize config/samples/ | sed -e 's/openfunction\/sample-go-func/openfunctiondev\/sample-go-func/g' > config/samples/function-sample-dev.yaml

clean:
	git checkout config/bundle.yaml
	rm -rf config/samples/function-sample-dev.yaml
	docker rmi `docker image ls | sed -e 's/[ ][ ]*/\t/g' | cut -f 2,3 | grep none | cut -f 2 | tr "\n" " "`  2>/dev/null

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.4.1 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif
