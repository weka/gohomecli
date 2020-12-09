BUILDPATH=$(CURDIR)
BINPATH=$(BUILDPATH)/bin
PKGPATH=$(BUILDPATH)/pkg

# export GO11MODULE=yes
GO=$(shell which go)
GOGET=$(GO) get

# PLATFORMS := darwin/386 darwin/amd64 linux/386 linux/amd64 windows/386 windows/amd64 freebsd/386
PLATFORMS := darwin/amd64 linux/amd64 windows/386 windows/amd64
PLATFORM = $(subst /, ,$@)
OS = $(word 1, $(PLATFORM))
ARCH = $(word 2, $(PLATFORM))

EXENAME=homecli
CMDSOURCES = $(wildcard cmd/homecli/*.go)
GOBUILD=$(GO) build

# .PHONY: makedir build test clean prepare default all $(PLATFORMS)
.DEFAULT_GOAL := default

.PHONY: makedir
makedir:
	@echo "Creating directories"
	@if [ ! -d $(BINPATH) ] ; then mkdir -p $(BINPATH) ; fi
	@if [ ! -d $(PKGPATH) ] ; then mkdir -p $(PKGPATH) ; fi
	@echo ok

.PHONY: build
build:
	@echo "Starting build"
	@$(GOBUILD) -o $(BINPATH)/$(EXENAME) $(CMDSOURCES)
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
	@rm -rf $(BINPATH)
	@rm -rf $(PKGPATH)
	@rm -rf $(BUILDPATH)/src
	@echo ok

.PHONY: prepare
prepare: test makedir

.PHONY: default
default: prepare build

.PHONY: $(PLATFORMS)
$(PLATFORMS):
	@echo "Building $(OS)/$(ARCH)"
	$(eval EXT := $(shell if [ "$(OS)" = "windows" ]; then echo .exe; fi))
	@GOOS=$(OS) GOARCH=$(ARCH) $(GOBUILD) -o $(BINPATH)/$(EXENAME)_$(OS)_$(ARCH)$(EXT) $(CMDSOURCES)
	@echo ok

.PHONY: all
all: default $(PLATFORMS)