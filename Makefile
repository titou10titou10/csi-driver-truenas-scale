# Copyright (C) 2025 Denis Forveille titou10.titou10@gmail.com
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

# versions
IMAGE_TAG ?= dev
CSI_VERSION = 1.11.0

# code
MAIN_GO_FILE = cmd/tnsplugin/main.go
OUTPUT_BIN = bin/tnsplugin
PKG = github.com/titou10/csi-driver-truenas-scale

# podman
IMAGE_NAME = tnsplugin
REGISTRY ?= ghcr.io/titou10titou10
CONTAINERFILE_PATH = Containerfile
CONTAINER_BUILD_CONTEXT = .

# go
GIT_COMMIT = $(shell git rev-parse HEAD)
BUILD_DATE = $(shell date -u +"%Y-%m-%dT%H:%M:%SZ")
LDFLAGS1 = -X ${PKG}/pkg/csi.driverVersion=${IMAGE_TAG} -X ${PKG}/pkg/csi.gitCommit=${GIT_COMMIT} -X ${PKG}/pkg/csi.buildDate=${BUILD_DATE}
LDFLAGS2 = -s -w -extldflags "-static"

# exec
GO_CMD = go
PODMAN_CMD = podman

# Paths

# Build the Go application
.PHONY: build
build:
#	$(GO_CMD) build -o bin/$(IMAGE_NAME) $(MAIN_GO_FILE)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 $(GO_CMD) build -o $(OUTPUT_BIN) -trimpath -ldflags="${LDFLAGS1} ${LDFLAGS2}" $(MAIN_GO_FILE)

# Build the container image using Podman
.PHONY: podman-build
podman-build: build
	$(PODMAN_CMD) build \
	              -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) \
	              -f $(CONTAINERFILE_PATH) \
				  --build-arg DRIVER_VERSION="${IMAGE_TAG}" \
				  --build-arg BUILD_DATE="${BUILD_DATE}" \
				  --build-arg GIT_COMMIT="${GIT_COMMIT}" \
				  --build-arg CSI_VERSION="${CSI_VERSION}" \
				  $(CONTAINER_BUILD_CONTEXT)

# Tag the container image
.PHONY: podman-tag
podman-tag:
	$(PODMAN_CMD) tag $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)-$(shell date +%Y%m%d%H%M%S)

# Push the container image to the registry using Podman
.PHONY: podman-push
podman-push: podman-build
	$(PODMAN_CMD) push $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG)

# Clean build artifacts
.PHONY: clean
clean:
	rm -rf bin/*

# Full process (build and push)
.PHONY: build-and-push
build-and-push: podman-push clean

# -------------------------
# Github Container Registry
# -------------------------
#REGISTRY_GHCR = ghcr.io/titou10titou10
#
# Build + push ghcr
#.PHONY: push-ghcr
#push-ghcr: podman-push
#	$(PODMAN_CMD) tag $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) $(REGISTRY_GHCR)/$(IMAGE_NAME):$(IMAGE_TAG)
#	$(PODMAN_CMD) push $(REGISTRY_GHCR)/$(IMAGE_NAME):$(IMAGE_TAG)

# --------
# Helm
# --------
HELM_VERSION = ${IMAGE_TAG}
HELM_REGISTRY ?= ghcr.io/titou10titou10
HELM_BASE_DIR =${IMAGE_TAG}
ifeq ($(IMAGE_TAG),dev)
   HELM_VERSION = v0.0.0-dev
endif
ifeq ($(IMAGE_TAG),ci)
   HELM_VERSION = v0.0.0-ci
   HELM_BASE_DIR = dev
endif

.PHONY: helm-package
helm-package:
	helm package charts/${HELM_BASE_DIR}/tns-csi-driver -d charts/${HELM_BASE_DIR}/ --version ${HELM_VERSION}
	helm repo index charts --url ${HELM_REGISTRY}

.PHONY: helm-push
helm-push: helm-package
	helm push charts/${HELM_BASE_DIR}/tns-csi-driver-${HELM_VERSION}.tgz oci://${HELM_REGISTRY}

# --------
# Tests
# --------
.PHONY: test-go
test-go:
	go test -v pkg/csi/*

.PHONY: go-vet
go-vet:
	pwd
	go list ./... | grep -v vendor
	go vet $(shell go list ./... | grep -v vendor)