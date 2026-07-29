SHELL = bash
UNAME := $(shell uname)

PROJECT_ROOT := $(patsubst %/,%,$(dir $(abspath $(lastword $(MAKEFILE_LIST)))))
APP_NAME = natscope
APP_PROJECT =
APP_ENV_PREFIX ?=
APP_VERSION ?= v$(shell git describe --tags --abbrev=0 2>/dev/null | sed 's/^v//' || echo "0.0.0")
APP_VERSION_COMMIT ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo "unknown")

GOOS ?= $(shell uname -s | tr '[:upper:]' '[:lower:]')
GOARCH ?= $(shell uname -m)
GO_RACE_DETECTOR_ENABLE ?= 0
BUILD_TAGS ?= cse
COMPILER_FLAGS ?=

# Pure-Go bbolt + keychain/file-vault secrets: no CGO needed. The -race blocks
# below still flip this on when the race detector is requested.
CGO_ENABLE = 0

BIN_OUTPUT_DIR ?= ${PROJECT_ROOT}/build/bin/${APP_VERSION}/${GOOS}-${GOARCH}
LDFLAGS ?= -w -s

ifeq (Darwin,$(UNAME))
LDFLAGS := $(LDFLAGS) -extldflags=-Wl,-ld_classic
endif

DOCKER_IMAGE ?= "natscope"
DOCKER_IMAGE_LATEST_TAG = "latest"

ifeq ($(findstring alpha,$(APP_VERSION)), alpha)
	DOCKER_IMAGE_LATEST_TAG = "dev"
else ifeq ($(findstring beta,$(APP_VERSION)), beta)
	DOCKER_IMAGE_LATEST_TAG = "beta"
else ifeq ($(findstring rc,$(APP_VERSION)), rc)
	DOCKER_IMAGE_LATEST_TAG = "rc"
endif

ifeq ($(findstring -race,$(COMPILER_FLAGS)), -race)
	CGO_ENABLE = 1
else ifeq ($(GO_RACE_DETECTOR_ENABLE), 1)
	CGO_ENABLE = 1
	COMPILER_FLAGS := $(COMPILER_FLAGS) -race
endif

.PHONY: all
all: help

.PHONY: help
help: ## Show this help
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n\033[36m\033[0m"} /^[$$()% a-zA-Z_-]+:.*?##/ { printf "  \033[36m%-30s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

.PHONY: clean
clean: ## Remove temporary files
	@rm -rf ${BIN_OUTPUT_DIR}
	@rm -rf ${PROJECT_ROOT}/build
	@rm -rf ${PROJECT_ROOT}/internal/transports/grpc/dist

.PHONY: deps
deps: install-tools ## Install all dependencies
	@go mod download
	@npm install
	@cd web && npm install
	@$(MAKE) proto-generate

.PHONY: install-tools
install-tools: ## Install pinned dev tools from devtools/
	@cd ${PROJECT_ROOT}/devtools && go install \
		github.com/daixiang0/gci \
		github.com/golangci/golangci-lint/v2/cmd/golangci-lint
	@go install github.com/bufbuild/buf/cmd/buf@v1.65.0
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@v1.36.11
	@go install connectrpc.com/connect/cmd/protoc-gen-connect-go@v1.19.2
	@go install github.com/planetscale/vtprotobuf/cmd/protoc-gen-go-vtproto@v0.6.1-0.20240319094008-0393e58bdf10

.PHONY: fmt
fmt: tidy ## Run go fmt on all go files
	@gci write \
		-s standard \
		-s default \
		-s "prefix(google.golang.org)" \
		-s "prefix(golang.org)" \
		-s "prefix(github.com/altessa-s/go-atlas)" \
		-s "prefix(github.com/dmit-4884/natscope)" \
		-s blank -s alias \
	 $$(go list -f {{.Dir}} ./... | grep -v /proto/gen/)

.PHONY: lint
lint: tidy fmt ## Run linter
	golangci-lint run ./...

.PHONY: tidy
tidy: ## Run go mod tidy
	@go mod tidy

.PHONY: generate
generate: ## Generate code (go generate)
	@go generate ./...

##@ Proto

.PHONY: proto-generate
proto-generate: ## Generate Go and TypeScript code from proto files
	@test -x ${PROJECT_ROOT}/node_modules/.bin/protoc-gen-es || { \
		echo "error: root node_modules missing (protoc-gen-es). Run 'make deps' or 'npm install' first."; \
		exit 1; \
	}
	cd ${PROJECT_ROOT}/proto && buf generate
	cd ${PROJECT_ROOT}/proto && buf generate --template buf.gen.deps-ts.yaml

.PHONY: proto-lint
proto-lint: ## Lint proto files
	cd ${PROJECT_ROOT}/proto && buf lint

.PHONY: proto-breaking
proto-breaking: ## Check for breaking changes in proto files
	cd ${PROJECT_ROOT}/proto && buf breaking --against '.git#subdir=proto'

##@ Build

.PHONY: build
build: clean proto-generate tidy build-frontend ## Build project with embedded frontend
	@echo "-------------------------------------------"
	@echo "Build options:"
	@echo PROJECT_ROOT="${PROJECT_ROOT}"
	@echo GOOS="${GOOS}"
	@echo GOARCH="${GOARCH}"
	@echo CGO_ENABLED="${CGO_ENABLE}"
	@echo VERSION="${APP_VERSION}"
	@echo VERSION_COMMIT="${APP_VERSION_COMMIT}"
	@echo LDFLAGS="${LDFLAGS}"
	@echo BIN_OUTPUT_DIR="${BIN_OUTPUT_DIR}"
	@echo "-------------------------------------------"

	CGO_ENABLED=${CGO_ENABLE} GOOS=${GOOS} GOARCH=${GOARCH} go build -trimpath \
		-tags "${BUILD_TAGS}" \
		${COMPILER_FLAGS} \
		-ldflags "${LDFLAGS} \
			-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Name=${APP_NAME} \
			-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Project=${APP_PROJECT} \
			-X github.com/altessa-s/go-atlas/core/runtime/appinfo.EnvPrefix=${APP_ENV_PREFIX} \
			-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Version=${APP_VERSION} \
			-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Commit=${APP_VERSION_COMMIT}" \
		-o ${BIN_OUTPUT_DIR}/${APP_NAME} ${PROJECT_ROOT}/cmd/${APP_NAME}

	@echo "Binary created at: ${BIN_OUTPUT_DIR}/${APP_NAME}"

CROSS_PLATFORMS ?= linux/amd64 linux/arm64 darwin/amd64 darwin/arm64 windows/amd64

.PHONY: build-all
build-all: clean proto-generate tidy build-frontend ## Build for all platforms (linux, darwin, windows)
	@for platform in $(CROSS_PLATFORMS); do \
		os=$${platform%%/*}; \
		arch=$${platform##*/}; \
		ext=""; \
		if [ "$$os" = "windows" ]; then ext=".exe"; fi; \
		outdir="${PROJECT_ROOT}/build/bin/${APP_VERSION}/$${os}-$${arch}"; \
		echo "Building $${os}/$${arch}..."; \
		CGO_ENABLED=0 GOOS=$$os GOARCH=$$arch go build -trimpath \
			-tags "${BUILD_TAGS}" \
			-ldflags "-w -s \
				-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Name=${APP_NAME} \
				-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Project=${APP_PROJECT} \
				-X github.com/altessa-s/go-atlas/core/runtime/appinfo.EnvPrefix=${APP_ENV_PREFIX} \
				-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Version=${APP_VERSION} \
				-X github.com/altessa-s/go-atlas/core/runtime/appinfo.Commit=${APP_VERSION_COMMIT}" \
			-o "$${outdir}/${APP_NAME}$${ext}" ${PROJECT_ROOT}/cmd/${APP_NAME} || exit 1; \
	done
	@echo "-------------------------------------------"
	@echo "All binaries built in: ${PROJECT_ROOT}/build/bin/${APP_VERSION}/"
	@ls -la ${PROJECT_ROOT}/build/bin/${APP_VERSION}/*/

.PHONY: build-backend
build-backend: proto-generate stub-dist tidy ## Build backend only (without frontend)
	CGO_ENABLED=${CGO_ENABLE} go build -trimpath \
		-tags "${BUILD_TAGS}" \
		-ldflags "${LDFLAGS}" \
		-o ${BIN_OUTPUT_DIR}/${APP_NAME} ${PROJECT_ROOT}/cmd/${APP_NAME}

.PHONY: build-frontend
build-frontend: proto-generate ## Build frontend
	@echo "Building frontend..."
	@cd ${PROJECT_ROOT}/web && npm install && npm run build

.PHONY: build-docker-image
build-docker-image: ## Build docker container
ifndef DOCKER_IMAGE
	$(error DOCKER_IMAGE is not set)
endif

	@docker buildx build \
		--platform=linux/amd64 \
		--build-arg APP_NAME="${APP_NAME}" \
		--build-arg APP_VERSION="${APP_VERSION}" \
		--build-arg APP_VERSION_COMMIT="${APP_VERSION_COMMIT}" \
		--build-arg APP_ENV_PREFIX="${APP_ENV_PREFIX}" \
		--build-arg LDFLAGS="${LDFLAGS}" \
		--build-arg COMPILER_FLAGS="${COMPILER_FLAGS}" \
		--build-arg CGO_ENABLE="${CGO_ENABLE}" \
		--tag "${DOCKER_IMAGE}:${APP_VERSION}" \
		--tag "${DOCKER_IMAGE}:${DOCKER_IMAGE_LATEST_TAG}" \
		--label=org.opencontainers.image.title="${APP_NAME}" \
		--label=org.opencontainers.image.revision="${APP_VERSION_COMMIT}" \
		--label=org.opencontainers.image.version="${APP_VERSION}" \
		--label=org.opencontainers.image.created=$(shell date -u +"%Y-%m-%dT%H:%M:%SZ") \
		-f ./Dockerfile .

##@ Development

DIST_STUB := ${PROJECT_ROOT}/internal/transports/grpc/dist/index.html

.PHONY: stub-dist
stub-dist: ## Create a minimal frontend stub so go:embed compiles without a full build
	@mkdir -p ${PROJECT_ROOT}/internal/transports/grpc/dist
	@if [ ! -f "${DIST_STUB}" ]; then \
		printf '<!doctype html><html lang="en"><head><meta charset="UTF-8"/><title>Natscope</title></head><body><p>Run <code>make dev-frontend</code> or <code>make build-frontend</code> to build the UI.</p></body></html>' \
		> "${DIST_STUB}"; \
	fi

.PHONY: dev-backend
dev-backend: stub-dist ## Run backend in development mode (pair with make dev-frontend for the UI)
	@go run ./cmd/${APP_NAME} server run

.PHONY: dev-frontend
dev-frontend: ## Run frontend in development mode
	@cd ${PROJECT_ROOT}/web && npm run dev

##@ Testing

.PHONY: test
test: test-backend test-frontend ## Run all tests

.PHONY: test-backend
test-backend: ## Run backend tests
	@go test -v ./...

.PHONY: test-frontend
test-frontend: ## Run frontend tests
	@cd ${PROJECT_ROOT}/web && npm test

.PHONY: test-coverage
test-coverage: ## Run backend tests with coverage
	@go test -cover ./...
	@go test -coverprofile=coverage.out ./...
	@go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"

.PHONY: web-verify
web-verify: proto-generate ## Run full frontend verification (typecheck, lint, tests, build, bundle size)
	@cd ${PROJECT_ROOT}/web && npm run typecheck
	@cd ${PROJECT_ROOT}/web && npm run lint
	@cd ${PROJECT_ROOT}/web && npm test
	@cd ${PROJECT_ROOT}/web && npm run build
	@${PROJECT_ROOT}/scripts/check-bundle-size.sh

##@ CI/CD

.PHONY: precommit-install
precommit-install: ## Install pre-commit hooks
	@pip3 install pre-commit
	@pre-commit install

.PHONY: precommit-run
precommit-run: ## Run pre-commit hooks
	@pre-commit run --all-files
