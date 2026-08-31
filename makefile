SHELL := /bin/bash

BINARY := go-gitsemver
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

# Unit tests with coverage (excludes e2e and testutil)
COVER_PKGS := $(shell go list ./... | grep -v -E '/(e2e|testutil)')

.PHONY: test
test:
	go test -race -count=1 -cover -coverprofile=coverage.out -covermode=atomic $(COVER_PKGS)
	go tool cover -func coverage.out

.PHONY: e2e
e2e:
	go test -race -count=1 -v ./e2e/...

.PHONY: test-all
test-all: test e2e

.PHONY: lint
lint: install-tools
	golangci-lint run --timeout 5m --config ./.github/.golangci.yml ./...

.PHONY: fmt
fmt: install-tools
	golangci-lint fmt --config ./.github/.golangci.yml ./...

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: bump
bump:
	go get -u ./...
	go mod tidy

.PHONY: check-sec
check-sec:
	go install golang.org/x/vuln/cmd/govulncheck@v1.1.4
	govulncheck -show verbose -test=false ./...

.PHONY: clean
clean:
	go clean ./...
	go clean -testcache
	rm -rf bin/ coverage.out

.PHONY: install-tools
install-tools:
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin $(GOLANGCI_LINT_VERSION)

.PHONY: coverage-check
coverage-check: test
	@TOTAL=$$(go tool cover -func=coverage.out | grep total | awk '{print $$3}' | sed 's/%//'); \
	echo "Coverage: $$TOTAL%"; \
	if [ $$(echo "$$TOTAL < 85" | bc) -eq 1 ]; then \
		echo "FAIL: Coverage $$TOTAL% is below 85% threshold"; \
		exit 1; \
	fi

.PHONY: release-build
release-build:
	@mkdir -p bin/
	@echo "Building release binaries..."
	CGO_ENABLED=0 GOOS=linux   GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/go-gitsemver-linux-amd64   .
	CGO_ENABLED=0 GOOS=linux   GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/go-gitsemver-linux-arm64   .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/go-gitsemver-darwin-amd64  .
	CGO_ENABLED=0 GOOS=darwin  GOARCH=arm64 go build -ldflags "$(LDFLAGS)" -o bin/go-gitsemver-darwin-arm64  .
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 go build -ldflags "$(LDFLAGS)" -o bin/go-gitsemver-windows-amd64.exe .
	@echo "Generating checksums..."
	cd bin && shasum -a 256 go-gitsemver-* > checksums.txt
	@echo "Release artifacts:"
	@ls -lh bin/go-gitsemver-*
	@echo ""
	@cat bin/checksums.txt

.PHONY: ci
ci: fmt lint test-all coverage-check build

GOOS   ?= $(shell go env GOOS)

GOARCH ?= $(shell go env GOARCH)

# This module lives at the repository root (go.mod is at ./go.mod).
APPLICATION := .

GOLANGCI_LINT_VERSION := v2.12.2

GOVULNCHECK_VERSION   := v1.1.4

MUTEST_VERSION        := v0.6.0

MUTATION_BASE      ?= origin/main

MUTATION_THRESHOLD ?= 100

.PHONY: mutation
mutation:
	@echo "Mutation testing code changed vs $(MUTATION_BASE)..."
	@for dir in $(APPLICATION); do \
		if [ -d "$$dir" ]; then \
			echo "Mutation testing $$dir module..."; \
			(cd $$dir && { command -v mutest >/dev/null 2>&1 || go install github.com/fchimpan/mutest@$(MUTEST_VERSION); } && mutest -diff $(MUTATION_BASE) -threshold $(MUTATION_THRESHOLD) ./...) || exit 1; \
		fi; \
	done

.PHONY: mutation-all
mutation-all:
	@echo "Mutation testing all modules (threshold $(MUTATION_THRESHOLD)%)..."
	@for dir in $(APPLICATION); do \
		if [ -d "$$dir" ]; then \
			echo "Mutation testing $$dir module..."; \
			(cd $$dir && go mod download && { command -v mutest >/dev/null 2>&1 || go install github.com/fchimpan/mutest@$(MUTEST_VERSION); } && mutest -threshold $(MUTATION_THRESHOLD) ./...) || exit 1; \
		fi; \
	done

.PHONY: run
run:
	go run . $(ARGS)

.PHONY: help
help:
	@echo "Available targets:"
	@echo "  make build          - build ./bin/$(BINARY)"
	@echo "  make release-build  - cross-compile release binaries + checksums"
	@echo "  make run ARGS=...   - run the CLI from source"
	@echo "  make test           - unit tests with coverage (excludes e2e, testutil)"
	@echo "  make e2e            - end-to-end tests"
	@echo "  make test-all       - test + e2e"
	@echo "  make coverage-check - fail if coverage is below 85%"
	@echo "  make mutation       - mutation-test code changed vs $(MUTATION_BASE) (mutest, $(MUTATION_THRESHOLD)% kill)"
	@echo "  make mutation-all   - mutation-test the whole module (weekly CI audit)"
	@echo "  make lint           - run golangci-lint"
	@echo "  make fmt            - format code via golangci-lint"
	@echo "  make check-sec      - run govulncheck"
	@echo "  make tidy           - go mod tidy"
	@echo "  make bump           - upgrade dependencies"
	@echo "  make clean          - clean build & test caches"
	@echo "  make install-tools  - install golangci-lint $(GOLANGCI_LINT_VERSION) locally"
	@echo "  make ci             - fmt + lint + test-all + coverage-check + build"
