BUILD_PATH=$(CURDIR)
BIN_PATH=$(BUILD_PATH)/bin
PKG_PATH=$(BUILD_PATH)/pkg

# export GO11MODULE=yes
GO=$(shell which go)
GOGET=$(GO) get

# PLATFORMS := darwin/386 darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64 freebsd/386
PLATFORMS := darwin/amd64 linux/amd64 windows/386 windows/amd64
PLATFORM = $(subst /, ,$@)
OS = $(word 1, $(PLATFORM))
ARCH = $(word 2, $(PLATFORM))

BINARY_NAME=homecli
CMD_SOURCES = $(wildcard cmd/homecli/*.go)
BUILD_TIME = $(shell date +'%Y-%m-%d_%T')
VERSION = $(shell git describe)
LD_FLAGS = -X main.BuildVersion=$(VERSION) -X main.BuildTime=$(BUILD_TIME)
GO_BUILD = $(GO) build -ldflags "$(LD_FLAGS)"

# .PHONY: makedir build test clean prepare default all $(PLATFORMS)
.DEFAULT_GOAL := default

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