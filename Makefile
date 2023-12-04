TOP_DIR      := $(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))
CURR_DATE    ?= $(shell date +'%y/%m/%d-%H:%M')

LOCALBIN      = $(TOP_DIR)/bin

# kernel-style V=1 build verbosity
ifeq ("$(origin V)", "command line")
  BUILD_VERBOSE = $(V)
endif

ifeq ($(BUILD_VERBOSE),1)
  Q =
else
  Q = @
endif

##############################
# General
##############################

PWO_EXE          = pwo
PWO_SRCS         = $(shell find . -name '*.go')
PWO_SHS          = $(shell find . -name '*.sh')
PWO_TPLS         = $(shell find . -name '*.yaml.tpl')
PWO_REPO        ?= $(shell grep "^module" go.mod | awk '{print $$2}')

##############################
# Git
##############################

GIT_CURRENT_TAG      ?= $(shell git tag --contains | egrep "^[0-9]*\\.[0-9]*\\.[0-9]*$$" 2>/dev/null)
GIT_LATEST_TAG       ?= $(shell git tag | egrep "^[0-9]*\\.[0-9]*\\.[0-9]*$$" 2>/dev/null)

ifeq ($(GIT_CURRENT_TAG),)
ifeq ($(GIT_VERSION),)
GIT_VERSION           = $(shell git describe --dirty --tags --always 2>/dev/null)
endif
else
GIT_VERSION           = $(GIT_CURRENT_TAG)
endif
export GIT_VERSION

ifeq ($(GIT_COMMIT),)
GIT_COMMIT            = $(shell git rev-parse HEAD 2>/dev/null)
endif
export GIT_COMMIT

GIT_COMMIT_URL        = https://$(PWO_REPO)/tree/$(GIT_COMMIT)
export GIT_COMMIT_URL

##############################
# Go
##############################

GO                   ?= go
GO_FLAGS              = -buildvcs=false
GO_BUILD_FLAGS       ?= -ldflags " \
                          -X '${PWO_REPO}/pkg/version.BuildDate="$(CURR_DATE)"' \
                          -X '${PWO_REPO}/pkg/version.GitVersion=$(GIT_VERSION)' \
                          -X '${PWO_REPO}/pkg/version.GitCommit=$(GIT_COMMIT)' \
                          -X '${PWO_REPO}/pkg/version.GitCommitURL="$(GIT_COMMIT_URL)"' \
                         "
GO_BUILD_FLAGS_EXTRA ?=

GO_TEST              ?= $(GO) test
GO_TEST_ARGS         ?= -v -timeout 5m
BUILD_ARCHs           = amd64 arm64
BUILD_OSs             = darwin linux windows

GOOS                 ?= $(shell $(GO) env GOOS)
GOARCH               ?= $(shell $(GO) env GOARCH)

# directory for executables
EXE_DIR              ?=
ifeq ($(EXE_DIR),)
GOBIN                ?= $(shell $(GO) env GOBIN)
ifeq ($(GOBIN),)
EXE_DIR               = /usr/local/bin
else
EXE_DIR               = $(GOBIN)
endif
endif

export CGO_ENABLED:=0
export GO111MODULE:=on
export GO15VENDOREXPERIMENT:=1

##############################
# Help                       
##############################

.DEFAULT_GOAL:=help

.PHONY: help
help: ## Show this help screen
	@echo 'Usage: make <OPTIONS> ... <TARGETS>'
	@echo ''
	@echo 'Available targets are:'
	@echo ''
	@awk 'BEGIN {FS = ":.*##"; printf "\nUsage:\n  make \033[36m<target>\033[0m\n"} /^[a-zA-Z0-9_-]+:.*?##/ { printf "  \033[36m%-15s\033[0m %s\n", $$1, $$2 } /^##@/ { printf "\n\033[1m%s\033[0m\n", substr($$0, 5) } ' $(MAKEFILE_LIST)

##############################
# Development
##############################

##@ Development

.PHONY: all install

all: format build test ## Build and test the project

# Code management.
.PHONY: format tidy clean cli-doc lint build

format-go: ## Format the Go source code
	@echo ">>> Formatting the Go source code..."
	$(Q)GOFLAGS="$(GO_FLAGS)" $(GO) fmt `$(GO) list ./...`

format-sh:  ## Format the Shell source code
	@echo ">>> Formatting the Shell source code..."
	$(Q)command -v shfmt >/dev/null && echo $(PWO_SHS) | xargs -r shfmt -w

format-all: format-go format-sh
format: format-all
fmt: format

tidy: ## Update dependencies
	$(Q)go mod tidy -v

.PHONY: vet
vet: ## Run go vet against code.
	$(Q)go vet ./...

clean: ## Clean up the build artifacts
	$(Q)rm -rf $(PWO_EXE) \
		build/* \
		test-*.log \

lint-sh:
	$(Q)echo ">>> Running shellcheck..."
	$(Q)shellcheck -s bash -x -S error $(PWO_SHS)

lint-go:
	$(Q)echo ">>> Running golang-ci..."
	$(Q)golangci-lint run

lint: lint-sh lint-go ## Run shellcheck and golangci-lint

##############################
##@ Build
##############################

build: $(PWO_EXE)## Build the executable

$(PWO_EXE): $(PWO_SRCS) $(PWO_TPLS)
	@go env -w GOPRIVATE=${GOPRIVATE}
	@echo ">>> Building $(PWO_EXE)_$(GOOS)_$(GOARCH) (GOARGS=$(GOARGS), GOOS=$(GOOS), GOARCH=$(GOARCH) FLAGS=$(GO_FLAGS))"
	$(Q)$(GOARGS) go build \
		-gcflags "all=-trimpath=${GOPATH}" \
		-asmflags "all=-trimpath=${GOPATH}" \
		$(GO_FLAGS) \
		$(GO_BUILD_FLAGS) \
		$(GO_BUILD_FLAGS_EXTRA) \
		-o "$@"

.PHONY: build-binaries
build-binaries:
	@echo ">>> Building binaries for all arches/OSes..."
	@mkdir -p $(TOP_DIR)/dist
	@for os in ${BUILD_OSs}; do \
		for arch in ${BUILD_ARCHs}; do \
			echo ">>> Building binaries for $$os/$$arch ..."; \
			GOOS=$$os GOARCH=$$arch make build; \
			if [ $$os = "windows" ]; then \
				mv $(PWO_EXE) $(TOP_DIR)/dist/$(PWO_EXE)-$$os-$$arch.exe; \
			else \
				mv $(PWO_EXE) $(TOP_DIR)/dist/$(PWO_EXE)-$$os-$$arch; \
    		fi \
		done \
	done

##############################
##@ Release
##############################
release:
	docker build -f release.Dockerfile \
		--build-arg BUILD_VERSION=${GIT_COMMIT} \
		--build-arg GIT_COMMIT=${GIT_COMMIT} \
		--build-arg GIT_VERSION=${GIT_VERSION} \
		--build-arg GIT_TOKEN=${GIT_TOKEN} \
		--build-arg GIT_HOST=${GIT_HOST} \
		--build-arg PRE_RELEASE="" .

release-show-latest-tag:
	@echo "Latest tag: $(GIT_LATEST_TAG)"

release-show-git-version:
	@echo "Git version: $(GIT_VERSION)"
