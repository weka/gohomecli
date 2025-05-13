BUILD_PATH=$(CURDIR)
BIN_PATH=$(BUILD_PATH)/bin
PKG_PATH=$(BUILD_PATH)/pkg

TOOLS_PATH=$(shell pwd)/.tools
GOLANGCI_LINT_VERSION=v2.1.6
GOLANGCI_LINT=$(TOOLS_PATH)/golangci-lint-$(GOLANGCI_LINT_VERSION)

MAIN_FALLBACK_BRANCH = master
BASE_BRANCH = $(shell \
  git branch -r | grep -o 'origin/release/v[0-9.]*' | sed 's@origin/@@' | \
  sort -V | tail -n1 | awk 'NF' || echo $(MAIN_FALLBACK_BRANCH) )

LS_LINT_VERSION=v2.3.0
LS_LINT=$(TOOLS_PATH)/ls-lint-$(LS_LINT_VERSION)

# export GO11MODULE=yes
GO=$(shell which go)
GOGET=$(GO) get

# PLATFORMS := darwin/386 darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64 freebsd/386
PLATFORMS := darwin/amd64 linux/amd64 windows/386 windows/amd64
PLATFORM = $(subst /, ,$@)
OS = $(word 1, $(PLATFORM))
ARCH = $(word 2, $(PLATFORM))

CURRENT_OS = $(shell uname -s | tr '[:upper:]' '[:lower:]')
CURRENT_ARCH = $(shell uname -m)

BINARY_NAME=homecli
CMD_SOURCES = $(wildcard cmd/homecli/*.go)
BUILD_TIME = $(shell date +'%Y-%m-%d_%T')
VERSION = $(shell git describe)
LD_FLAGS = -X main.BuildVersion=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
GO_BUILD = $(GO) build -ldflags "$(LD_FLAGS)"

# .PHONY: makedir build test clean prepare default all $(PLATFORMS)
.DEFAULT_GOAL := default

$(TOOLS_PATH):
	@mkdir -p $(TOOLS_PATH)

$(GOLANGCI_LINT): $(TOOLS_PATH)
	@echo "Installing golangci-lint"
	@curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $(TOOLS_PATH) $(GOLANGCI_LINT_VERSION)
	@mv $(TOOLS_PATH)/golangci-lint $(GOLANGCI_LINT)

$(LS_LINT): $(TOOLS_PATH)
	@if [ ! -x "$(LS_LINT)" ]; then \
		echo "Installing ls-lint for $(CURRENT_OS)/$(CURRENT_ARCH)"; \
		curl -fsSL --show-error -o $(LS_LINT) \
		  https://github.com/loeffel-io/ls-lint/releases/download/$(LS_LINT_VERSION)/ls-lint-$(CURRENT_OS)-$(CURRENT_ARCH); \
		chmod +x $(LS_LINT); \
	else \
		echo "ls-lint already installed at $(LS_LINT)"; \
	fi

.PHONY: makedir
makedir:
	@echo "Creating directories"
	@if [ ! -d $(BIN_PATH) ] ; then mkdir -p $(BIN_PATH) ; fi
	@if [ ! -d $(PKG_PATH) ] ; then mkdir -p $(PKG_PATH) ; fi
	@echo ok

.PHONY: build
build:
	@echo "Starting build"
	@$(GO_BUILD) -o $(BIN_PATH)/$(BINARY_NAME) $(CMD_SOURCES)
	@echo ok

.PHONY: test
test:
	@echo "Validating with go fmt"
	@go fmt $$(go list ./... | grep -v /vendor/)
	@echo ok
	@echo "Validating with go vet"
	@go vet $$(go list ./... | grep -v /vendor/)
	@echo ok

vet:
	@echo "Validating with go vet"
	@go vet ./...

.PHONY: tools
tools: $(GOLANGCI_LINT) $(LS_LINT)

.PHONY: tidy
tidy:
	@echo "Tidying"
	@go mod tidy

.PHONY: ls-lint
ls-lint: $(LS_LINT)
	@echo "Validating with ls-lint"
	@$(LS_LINT) -config .ls-lint.yml

.PHONY: ls-lint-files
ls-lint-files: $(LS_LINT)
	@if [ -z "$(FILES)" ]; then \
		echo "No files provided"; \
		exit 1; \
	fi
	@echo "Running ls-lint for files: $(FILES)"
	@$(LS_LINT) -config .ls-lint.yml $(FILES)

.PHONY: lint
lint: $(GOLANGCI_LINT)
	@echo "Validating with golangci-lint"
	@$(GOLANGCI_LINT) run

.PHONY: lint-fix
lint-fix: $(GOLANGCI_LINT) fmt
	@echo "Validating with golangci-lint"
	@$(GOLANGCI_LINT) run --fix

.PHONY: fmt
fmt: $(GOLANGCI_LINT)
	@echo "Running golangci-lint fmt"
	@$(GOLANGCI_LINT) fmt

.PHONY: lintfix-changes
lintfix-changes: $(GOLANGCI_LINT)
	@echo "Running golangci-lint lintfix-changes"
	@$(GOLANGCI_LINT) run --fix --new-from-merge-base=$(BASE_BRANCH) --whole-files

.PHONY: fmt-files
fmt-files:
	@if [ -z "$(FILES)" ]; then \
		echo "No files provided"; \
		exit 1; \
	fi
	@echo "Running golangci-lint fmt for files: $(FILES)"
	@$(GOLANGCI_LINT) fmt $(FILES)

.PHONY: clean
clean:
	@echo "Cleaning directories"
	@rm -rf $(BIN_PATH)
	@rm -rf $(PKG_PATH)
	@rm -rf $(BUILD_PATH)/src
	@echo ok

.PHONY: prepare
prepare: test makedir

.PHONY: default
default: prepare build

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	@echo "Building $(OS)/$(ARCH)"
	$(eval EXT := $(shell if [ "$(OS)" = "windows" ]; then echo .exe; fi))
	@GOOS=$(OS) GOARCH=$(ARCH) $(GO_BUILD) -o $(BIN_PATH)/$(BINARY_NAME)_$(OS)_$(ARCH)$(EXT) $(CMD_SOURCES)
	@echo ok

.PHONY: all
all: default $(PLATFORMS)
