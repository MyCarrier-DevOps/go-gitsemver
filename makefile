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
	go test -race -cover -coverprofile=coverage.out -covermode=atomic $(COVER_PKGS)
	go tool cover -func coverage.out

.PHONY: e2e
e2e:
	go test -race -count=1 -v ./e2e/...

.PHONY: test-all
test-all: test e2e

.PHONY: lint
lint: install-tools
	golangci-lint run --timeout 5m --config ./.golangci.yml ./...

.PHONY: fmt
fmt: install-tools
	golangci-lint fmt --config ./.golangci.yml ./...

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
	curl -sSfL https://raw.githubusercontent.com/golangci/golangci-lint/HEAD/install.sh | sh -s -- -b $$(go env GOPATH)/bin v2.9.0

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
