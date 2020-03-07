PROJECT     := github.com/cfhamlet/os-rq-pod
BINNAME     ?= rq-pod
BINDIR      := $(CURDIR)/bin
BUILDTIME   := $(shell date +'%Y-%m-%d %H:%M:%S')

GOPATH      := $(shell go env GOPATH)
GOVERSION   := $(shell go version)
GOIMPORTS   := $(GOPATH)/bin/goimports
GOLINT      := $(GOPATH)/bin/golangci-lint
INSTALLPATH := $(GOPATH)/bin

PKG         := ./...
TESTS       := .
LDFLAGS     :=
GOFLAGS     := -tags=jsoniter
TESTFLAGS   :=
SRC         := $(shell find . -type f -name '*.go' -print)

GIT_COMMIT  := $(shell git rev-parse HEAD)
GIT_SHA     := $(shell git rev-parse --short HEAD)
GIT_TAG     := $(shell git describe --tags --abbrev=0 --exact-match 2>/dev/null)
GIT_STATUS  := $(shell test -n "`git status --porcelain`" && echo "dirty" || echo "clean")

VERSION     ?= $(GIT_TAG)
VERSIONMOD  := $(PROJECT)/internal/version

ifneq ($(VERSION),)
	LDFLAGS +=  -X "$(VERSIONMOD).version=$(VERSION)"
endif

LDFLAGS +=  -X "$(VERSIONMOD).goVersion=$(GOVERSION)"
LDFLAGS +=  -X "$(VERSIONMOD).buildTime=$(BUILDTIME)"
LDFLAGS +=  -X "$(VERSIONMOD).gitCommit=$(GIT_COMMIT)"
LDFLAGS +=  -X "$(VERSIONMOD).gitTag=$(GIT_TAG)"
LDFLAGS +=  -X "$(VERSIONMOD).gitStatus=$(GIT_STATUS)"

.PHONY: all
all: build


.PHONY: build
build: $(BINDIR)/$(BINNAME)

$(BINDIR)/$(BINNAME): $(SRC)
	@echo
	@echo  "==> Building ./cmd/rq-pod $(BINDIR)/$(BINNAME) <=="
	GO111MODULE=on go build $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(BINDIR)/$(BINNAME) ./cmd/rq-pod

.PHONY: test
test: build
test: TESTFLAGS += -race -v
test: test-lint
test: test-unit
test: test-coverage
test: test-bench

.PHONY: test-lint
test-lint:$(GOLINT)
	@echo
	@echo  "==> Running lint test <=="
	GO111MODULE=on $(GOLINT) run


$(GOLINT):
	@echo
	@echo  "==> Installing golangci-lint <=="
	(cd /; curl -sfL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh| sh -s -- -b $(GOPATH)/bin v1.21.0)

.PHONY: format
format: $(GOIMPORTS)
	@echo
	@echo  "==> Formatting <=="
	GO111MODULE=on go list -f '{{.Dir}}' ./... | xargs $(GOIMPORTS) -w 

$(GOIMPORTS):
	@echo
	@echo  "==> Installing goimports <=="
	(cd /; GO111MODULE=on go get -u golang.org/x/tools/cmd/goimports)

.PHONY: test-unit
test-unit:
	@echo
	@echo  "==> Running unit tests <=="
	GO111MODULE=on go test -run $(TESTS) $(PKG) $(TESTFLAGS)

.PHONY: test-bench
test-bench:
	@echo
	@echo  "==> Running benchmark tests <=="
	GO111MODULE=on go test -bench $(TESTS) $(PKG)

.PHONY: test-coverage
test-coverage:
	@echo
	@echo  "==> Running unit tests with coverage <=="
	@ ./scripts/coverage.sh

.PHONY: install
install:
	GO111MODULE=on go build -i $(GOFLAGS) -ldflags '$(LDFLAGS)' -o $(INSTALLPATH)/$(BINNAME) ./cmd/rq-pod

.PHONY: clean
clean:
	@rm -rf $(BINDIR)

.PHONY: info
info:
	 @echo "Version:    ${VERSION}"
	 @echo "Go Version: ${GOVERSION}"
	 @echo "Git Tag:    ${GIT_TAG}"
	 @echo "Git Commit: ${GIT_COMMIT}"
	 @echo "Git Status: ${GIT_STATUS}"
