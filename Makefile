# SPDX-FileCopyrightText: (C) 2025 Intel Corporation
# SPDX-License-Identifier: Apache-2.0

PROJECT_NAME                       := sre-exporter

## Labels to add Docker/Helm/Service CI meta-data.
LABEL_REVISION                     = $(shell git rev-parse HEAD)
LABEL_CREATED                      ?= $(shell date -u "+%Y-%m-%dT%H:%M:%SZ")

VERSION                            ?= $(shell cat VERSION | tr -d '[:space:]')
BUILD_DIR                          ?= ./build

## CHART_NAME is specified in Chart.yaml
CHART_NAME                         ?= $(PROJECT_NAME)
## CHART_VERSION is specified in Chart.yaml
CHART_VERSION                      ?= $(shell grep "version:" ./deployments/$(PROJECT_NAME)/Chart.yaml  | cut -d ':' -f 2 | tr -d '[:space:]')
## CHART_APP_VERSION is modified on every commit
CHART_APP_VERSION                  ?= $(VERSION)
## CHART_BUILD_DIR is given based on repo structure
CHART_BUILD_DIR                    ?= $(BUILD_DIR)/chart/
## CHART_PATH is given based on repo structure
CHART_PATH                         ?= "./deployments/$(CHART_NAME)"
## CHART_NAMESPACE can be modified here
CHART_NAMESPACE                    ?= orch-sre
## CHART_RELEASE can be modified here
CHART_RELEASE                      ?= $(PROJECT_NAME)
## CHART_TEST is specified in test-connection.yaml
CHART_TEST                         ?= test-connection

REGISTRY                           ?= 080137407410.dkr.ecr.us-west-2.amazonaws.com
REGISTRY_NO_AUTH                   ?= edge-orch
REPOSITORY                         ?= o11y
REPOSITORY_NO_AUTH                 := $(REGISTRY)/$(REGISTRY_NO_AUTH)/$(REPOSITORY)
DOCKER_METRICS_EXPORTER_IMAGE_NAME ?= sre-metrics-exporter
DOCKER_CONFIG_RELOADER_IMAGE_NAME  ?= sre-config-reloader
DOCKER_IMAGE_TAG                   ?= $(VERSION)
DOCKER_REGISTRY_READ_PATH          = registry-rs.edgeorchestration.intel.com/edge-orch/o11y

TEST_JOB_NAME 				       ?= sre-exporter-test-connection
DOCKER_FILES_TO_LINT               := $(shell find . -type f -name 'Dockerfile*' -print )

GOCMD         := CGO_ENABLED=0 GOARCH=amd64 GOOS=linux go
GOCMD_TEST    := CGO_ENABLED=1 GOARCH=amd64 GOOS=linux go
GOEXTRAFLAGS  :=-trimpath -mod=readonly -gcflags="all=-spectre=all -N -l" -asmflags="all=-spectre=all" -ldflags="all=-s -w -X main.version=$(shell cat ./VERSION)"

.DEFAULT_GOAL := help
.PHONY: build

## CI Mandatory Targets start
dependency-check:
	@# Help: Unsupported target
	@echo '"make $@" is unsupported'

build: build-metrics-exporter build-config-reloader
	@# Help: Builds metrics exporter and config reloader

lint: lint-go lint-markdown lint-yaml lint-proto lint-json lint-shell lint-helm lint-docker
	@# Help: Runs all linters

test:
	@# Help: Runs tests and creates a coverage report
	@echo "---MAKEFILE TEST---"
	$(GOCMD_TEST) test ./... --race -coverprofile $(BUILD_DIR)/coverage.out -covermode atomic
	gocover-cobertura < $(BUILD_DIR)/coverage.out > $(BUILD_DIR)/coverage.xml
	@echo "---END MAKEFILE TEST---"

docker-build: docker-build-metrics-exporter docker-build-config-reloader
	@# Help: Builds all docker images

helm-build: helm-clean
	@# Help: Builds the helm chart
	@echo "---MAKEFILE HELM-BUILD---"
	yq eval -i '.version = "$(VERSION)"' $(CHART_PATH)/Chart.yaml
	yq eval -i '.appVersion = "$(VERSION)"' $(CHART_PATH)/Chart.yaml
	yq eval -i '.annotations.revision = "$(LABEL_REVISION)"' $(CHART_PATH)/Chart.yaml
	yq eval -i '.annotations.created = "$(LABEL_CREATED)"' $(CHART_PATH)/Chart.yaml
	helm package \
		--app-version=$(CHART_APP_VERSION) \
		--debug \
		--dependency-update \
		--destination $(CHART_BUILD_DIR) \
		$(CHART_PATH)

	@echo "---END MAKEFILE HELM-BUILD---"

docker-push: docker-push-metrics-exporter docker-push-config-reloader
	@# Help: Pushes all docker images

helm-push:
	@# Help: Pushes the helm chart
	@echo "---MAKEFILE HELM-PUSH---"
	aws ecr create-repository --region us-west-2 --repository-name $(REGISTRY_NO_AUTH)/$(REPOSITORY)/charts/$(CHART_NAME) || true
	helm push $(CHART_BUILD_DIR)$(CHART_NAME)*.tgz oci://$(REPOSITORY_NO_AUTH)/charts
	@echo "---END MAKEFILE HELM-PUSH---"
## CI Mandatory Targets end

## Helper Targets start
all: clean build lint test
	@# Help: Runs clean, build, lint, test targets

clean:
	@# Help: Deletes directories created by build targets
	@echo "---MAKEFILE CLEAN---"
	rm -rf $(BUILD_DIR)
	@echo "---END MAKEFILE CLEAN---"

helm-clean:
	@# Help: Cleans the build directory of the helm chart
	@echo "---MAKEFILE HELM-CLEAN---"
	rm -rf $(CHART_BUILD_DIR)
	@echo "---END MAKEFILE HELM-CLEAN---"

build-metrics-exporter:
	@# Help: Builds metrics exporter
	@echo "---MAKEFILE BUILD-METRICS-EXPORTER---"
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BUILD_DIR)/metrics-exporter ./cmd/metrics-exporter
	@echo "---END MAKEFILE BUILD-METRICS-EXPORTER---"

build-config-reloader:
	@# Help: Builds SRE config reloader
	@echo "---MAKEFILE BUILD-CONFIG-RELOADER---"
	$(GOCMD) build $(GOEXTRAFLAGS) -o $(BUILD_DIR)/config-reloader ./cmd/config-reloader/config_reloader.go
	@echo "---END MAKEFILE BUILD-CONFIG-RELOADER---"

lint-go:
	@# Help: Runs linters for golang source code files including ./tests
	@echo "---MAKEFILE LINT-GO---"
	golangci-lint -v run --build-tags smoke,mage
	@echo "---END MAKEFILE LINT-GO---"

lint-markdown:
	@# Help: Runs linter for markdown files
	@echo "---MAKEFILE LINT-MARKDOWN---"
	markdownlint-cli2 '**/*.md' "!.github" "!**/ci/*"
	@echo "---END MAKEFILE LINT-MARKDOWN---"

lint-yaml:
	@# Help: Runs linter for for yaml files
	@echo "---MAKEFILE LINT-YAML---"
	yamllint -v
	yamllint -f parsable -c yamllint-conf.yaml .
	@echo "---END MAKEFILE LINT-YAML---"

lint-proto:
	@# Help: Runs linter for for proto files
	@echo "---MAKEFILE LINT-PROTO---"
	protolint version
	protolint lint -reporter unix api/
	@echo "---END MAKEFILE LINT-PROTO---"

lint-json:
	@# Help: Runs linter for json files
	@echo "---MAKEFILE LINT-JSON---"
	./scripts/lintJsons.sh
	@echo "---END MAKEFILE LINT-JSON---"

lint-shell:
	@# Help: Runs linter for shell scripts
	@echo "---MAKEFILE LINT-SHELL---"
	shellcheck --version
	shellcheck ***/*.sh
	@echo "---END MAKEFILE LINT-SHELL---"

lint-helm:
	@# Help: Runs linter for helm chart
	@echo "---MAKEFILE LINT-HELM---"
	helm version
	helm lint --strict $(CHART_PATH) --values $(CHART_PATH)/values.yaml
	@echo "---END MAKEFILE LINT-HELM---"

lint-docker:
	@# Help: Runs linter for docker files
	@echo "---MAKEFILE LINT-DOCKER---"
	hadolint --version
	hadolint $(DOCKER_FILES_TO_LINT)
	@echo "---END MAKEFILE LINT-DOCKER---"

lint-license:
	@# Help: Runs license check
	@echo "---MAKEFILE LINT-LICENSE---"
	reuse --version
	reuse --root . lint
	@echo "---END MAKEFILE LINT-LICENSE---"

docker-build-metrics-exporter:
	@# Help: Builds metrics-exporter docker image
	@echo "---MAKEFILE DOCKER-BUILD-METRICS-EXPORTER---"
	docker rmi $(REPOSITORY_NO_AUTH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) --force
	docker build -f Dockerfile \
		-t $(REPOSITORY_NO_AUTH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) \
		--build-arg http_proxy="$(http_proxy)" --build-arg https_proxy="$(https_proxy)" --build-arg no_proxy="$(no_proxy)" \
		--platform linux/amd64 --no-cache .
	@echo "---END MAKEFILE DOCKER-BUILD-METRICS-EXPORTER---"

docker-build-config-reloader:
	@# Help: Builds SRE config reloader docker image
	@echo "---MAKEFILE DOCKER-CONFIG-RELOADER---"
	docker rmi $(REPOSITORY_NO_AUTH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) --force
	docker build -f Dockerfile.config-reloader \
		-t $(REPOSITORY_NO_AUTH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) \
		--build-arg http_proxy="$(http_proxy)" --build-arg https_proxy="$(https_proxy)" --build-arg no_proxy="$(no_proxy)" \
		--platform linux/amd64 --no-cache .
	@echo "---END MAKEFILE DOCKER-CONFIG-RELOADER---"

docker-push-metrics-exporter:
	@# Help: Pushes metrics-exporter docker image
	@echo "---MAKEFILE DOCKER-PUSH-METRICS-EXPORTER---"
	aws ecr create-repository --region us-west-2 --repository-name ${REGISTRY_NO_AUTH}/$(REPOSITORY)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME) || true
	docker push $(REPOSITORY_NO_AUTH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	@echo "---END MAKEFILE DOCKER-PUSH-METRICS-EXPORTER---"

docker-push-config-reloader:
	@# Help: Pushes SRE config reloader docker image
	@echo "---MAKEFILE DOCKER-PUSH-CONFIG-RELOADER---"
	aws ecr create-repository --region us-west-2 --repository-name ${REGISTRY_NO_AUTH}/$(REPOSITORY)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME) || true
	docker push $(REPOSITORY_NO_AUTH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	@echo "---END MAKEFILE DOCKER-PUSH-CONFIG-RELOADER---"

kind-all: helm-clean helm-build docker-build kind-load
	@# Help: Builds all images, loads them into the kind cluster and builds the helm chart

kind-load: kind-load-metrics-exporter kind-load-config-reloader
	@# Help: Loads all docker images into the kind cluster

kind-load-metrics-exporter:
	@# Help: Loads metrics-exporter docker image into the kind cluster
	@echo "---MAKEFILE KIND-LOAD-METRICS-EXPORTER---"
	docker tag $(REPOSITORY_NO_AUTH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) $(DOCKER_REGISTRY_READ_PATH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	kind load docker-image $(DOCKER_REGISTRY_READ_PATH)/$(DOCKER_METRICS_EXPORTER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	@echo "---END MAKEFILE KIND-LOAD-METRICS-EXPORTER---"

kind-load-config-reloader:
	@# Help: Loads SRE config reloader docker image into the kind cluster
	@echo "---MAKEFILE KIND-LOAD-CONFIG-RELOADER---"
	docker tag $(REPOSITORY_NO_AUTH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG) $(DOCKER_REGISTRY_READ_PATH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	kind load docker-image $(DOCKER_REGISTRY_READ_PATH)/$(DOCKER_CONFIG_RELOADER_IMAGE_NAME):$(DOCKER_IMAGE_TAG)
	@echo "---END MAKEFILE KIND-LOAD-CONFIG-RELOADER---"

kind-load-victoria-metrics:
	@# Help: Load VictoriaMetrics docker image into the kind cluster
	@echo "---MAKEFILE KIND-LOAD-VICTORIA-METRICS---"
	docker pull victoriametrics/victoria-metrics:latest
	docker tag victoriametrics/victoria-metrics:latest docker.io/victoriametrics/victoriametrics:latest
	kind load docker-image docker.io/victoriametrics/victoriametrics:latest
	@echo "---END MAKEFILE KIND-LOAD-VICTORIA-METRICS---"

test-mage:
	@# Help: Runs tests in magefiles folder
	@echo "---MAKEFILE TEST-MAGE---"
	$(GOCMD) test ./magefiles/... --tags=mage --count=1
	@echo "---END MAKEFILE TEST-MAGE---"

test-generate-ui-traffic:
	@# Help: Runs curl request to UI
	@echo "---MAKEFILE TEST-GENERATE-UI-TRAFFIC---"
	@# ignore error to not block dependent targets if /etc/hosts wasn't updated
	@-curl https://web-ui.kind.internal --noproxy '*'
	@echo "---END MAKEFILE TEST-GENERATE-UI-TRAFFIC---"

test-victoria-metrics-port-forward:
	@# Help: Forwards port of VictoriaMetrics testing instance from kind cluster to host
	@echo "---MAKEFILE TEST-VICTORIA-METRICS-PORT-FORWARD---"
	kubectl port-forward -n $(CHART_NAMESPACE)  svc/sre-exporter-destination 8428:8428 --address='0.0.0.0'
	@echo "---END MAKEFILE TEST-VICTORIA-METRICS-PORT-FORWARD---"

proto:
	@# Help: Regenerates proto-based code
	@echo "---MAKEFILE PROTO---"
    # Requires installed: protoc, protoc-gen-go and protoc-gen-go-grpc
    # See: https://grpc.io/docs/languages/go/quickstart/
	protoc api/config-reloader/*.proto --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative --proto_path=.
	@echo "---END-MAKEFILE PROTO---"

install-tools:
	@# Help: Installs tools required for the project
	# Requires installed: asdf
	@echo "---MAKEFILE INSTALL-TOOLS---"
	./scripts/installTools.sh .tool-versions
	@echo "---END MAKEFILE INSTALL-TOOLS---"
## Helper Targets end

list: help
	@# Help: Displays make targets

help:
	@# Help: Displays make targets
	@printf "%-35s %s\n" "Target" "Description"
	@printf "%-35s %s\n" "------" "-----------"
	@grep -E '^[a-zA-Z0-9_%-]+:|^[[:space:]]+@# Help:' Makefile | \
	awk '\
		/^[a-zA-Z0-9_%-]+:/ { \
			target = $$1; \
			sub(":", "", target); \
		} \
		/^[[:space:]]+@# Help:/ { \
			if (target != "") { \
				help_line = $$0; \
				sub("^[[:space:]]+@# Help: ", "", help_line); \
				printf "%-35s %s\n", target, help_line; \
				target = ""; \
			} \
		}' | sort -k1,1
