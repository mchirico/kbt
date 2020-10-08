
# Image URL to use all building/pushing image targets
IMG ?= controller:latest
# Produce CRDs that work back to Kubernetes 1.11 (no version conversion)
# CRD_OPTIONS ?= "crd:trivialVersions=true"
CRD_OPTIONS = "crd:crdVersions=v1"


# Get the currently used golang install path (in GOPATH/bin, unless GOBIN is set)
ifeq (,$(shell go env GOBIN))
GOBIN=$(shell go env GOPATH)/bin
else
GOBIN=$(shell go env GOBIN)
endif


# sed is different on Linux/Mac
OS_SED_FILE  :=
ifeq ($(OS),Windows_NT)
        OS_SED_FILE += sed
else
        UNAME_S := $(shell uname -s)
        ifeq ($(UNAME_S),Linux)
                OS_SED_FILE += sed -i -re
        endif
        ifeq ($(UNAME_S),Darwin)
                OS_SED_FILE += sed -ir -E
        endif
endif

OS_SED  :=
ifeq ($(OS),Windows_NT)
        OS_SED_FILE += sed
else
        UNAME_S := $(shell uname -s)
        ifeq ($(UNAME_S),Linux)
                OS_SED += sed -re
        endif
        ifeq ($(UNAME_S),Darwin)
                OS_SED += sed -E
        endif
endif



all: manager

# Run tests
test: generate fmt vet manifests
	go test ./... -coverprofile cover.out

# Build manager binary
manager: generate fmt vet
	go build -o bin/manager main.go

# Run against the configured Kubernetes cluster in ~/.kube/config
run: generate fmt vet manifests
	go run ./main.go

# Install CRDs into a cluster
install: manifests kustomize
	$(KUSTOMIZE) build config/crd |$(OS_SED) 's/( +)  UDP, TCP, or SCTP. Defaults to "TCP"./\1default: TCP/'|kubectl apply -f -


# Uninstall CRDs from a cluster
uninstall: manifests kustomize
	$(KUSTOMIZE) build config/crd |$(OS_SED) 's/( +)  UDP, TCP, or SCTP. Defaults to "TCP"./\1default: TCP/'|kubectl delete -f -



# Deploy controller in the configured Kubernetes cluster in ~/.kube/config
deploy: manifests kustomize
	cd config/manager && $(KUSTOMIZE) edit set image controller=${IMG}
	$(KUSTOMIZE) build config/default |$(OS_SED) 's/( +)  UDP, TCP, or SCTP. Defaults to "TCP"./\1default: TCP/'|kubectl apply -f -


# Generate manifests e.g. CRD, RBAC etc.
manifests: controller-gen
	$(CONTROLLER_GEN) $(CRD_OPTIONS) rbac:roleName=manager-role webhook paths="./..." output:crd:artifacts:config=config/crd/bases
	$(OS_SED_FILE) 's/( +)  UDP, TCP, or SCTP. Defaults to "TCP"./\1default: TCP/' config/crd/bases/*.yaml



# fix by making sure default: TCP
.PHONY: fix
fix:
	@echo 'checking files'
	$(OS_SED_FILE) 's/( +)  UDP, TCP, or SCTP. Defaults to "TCP"./\1default: TCP/' config/crd/bases/*.yaml
	grep 'default: TCP' config/crd/bases/*.yaml





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
docker-build: test
	docker build . -t ${IMG}

# Push the docker image
docker-push:
	docker push ${IMG}

# find or download controller-gen
# download controller-gen if necessary
controller-gen:
ifeq (, $(shell which controller-gen))
	@{ \
	set -e ;\
	CONTROLLER_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$CONTROLLER_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/controller-tools/cmd/controller-gen@v0.3.0 ;\
	rm -rf $$CONTROLLER_GEN_TMP_DIR ;\
	}
CONTROLLER_GEN=$(GOBIN)/controller-gen
else
CONTROLLER_GEN=$(shell which controller-gen)
endif

kustomize:
ifeq (, $(shell which kustomize))
	@{ \
	set -e ;\
	KUSTOMIZE_GEN_TMP_DIR=$$(mktemp -d) ;\
	cd $$KUSTOMIZE_GEN_TMP_DIR ;\
	go mod init tmp ;\
	go get sigs.k8s.io/kustomize/kustomize/v3@v3.5.4 ;\
	rm -rf $$KUSTOMIZE_GEN_TMP_DIR ;\
	}
KUSTOMIZE=$(GOBIN)/kustomize
else
KUSTOMIZE=$(shell which kustomize)
endif
