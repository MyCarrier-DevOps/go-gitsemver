SHELL := /bin/bash

BINARY := gitsemver
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "dev")
LDFLAGS := -X main.version=$(VERSION)

.PHONY: build
build:
	go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

.PHONY: test
test:
	go test -race -cover -coverprofile=coverage.out -covermode=atomic ./...
	go tool cover -func coverage.out

.PHONY: lint
lint: install-tools
	golangci-lint run --timeout 5m ./...

.PHONY: fmt
fmt: install-tools
	golangci-lint fmt ./...

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
