VERSION ?= "unknown"
VERSION_FLAG := -X github.com/nunnatsa/ginkgolinter/version.version=$(VERSION)
COMMIT_HASH := $(shell git rev-parse HEAD)
HASH_FLAG := -X github.com/nunnatsa/ginkgolinter/version.gitHash=$(COMMIT_HASH)

BUILD_ARGS := -ldflags "$(VERSION_FLAG) $(HASH_FLAG)"

build: unit-test
	go build $(BUILD_ARGS) -o ginkgolinter ./cmd/ginkgolinter

unit-test:
	go test ./...

build-for-windows:
	GOOS=windows GOARCH=amd64 go build $(BUILD_ARGS) -o bin/ginkgolinter-amd64.exe ./cmd/ginkgolinter

build-for-mac:
	GOOS=darwin GOARCH=amd64 go build $(BUILD_ARGS) -o bin/ginkgolinter-amd64-darwin ./cmd/ginkgolinter

build-for-linux:
	GOOS=linux GOARCH=amd64 go build $(BUILD_ARGS) -o bin/ginkgolinter-amd64-linux ./cmd/ginkgolinter
	GOOS=linux GOARCH=386 go build $(BUILD_ARGS) -o bin/ginkgolinter-386-linux ./cmd/ginkgolinter

build-all: build build-for-linux build-for-mac build-for-windows

test: build
	./tests/e2e.sh
