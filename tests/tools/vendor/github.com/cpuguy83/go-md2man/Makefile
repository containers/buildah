GO111MODULE ?= on
LINTER_BIN ?= golangci-lint

export GO111MODULE

.PHONY:
build: bin/go-md2man

.PHONY: clean
clean:
	@rm -rf bin/*

.PHONY: check
check: lint

.PHONY: test
test:
	@go test $(TEST_FLAGS) ./...

.PHONY: lint
lint:
	@$(LINTER_BIN) run --new-from-rev "HEAD~$(git rev-list master.. --count)" ./...

bin/go-md2man: actual_build_flags := $(BUILD_FLAGS) -o bin/go-md2man
bin/go-md2man: bin
	@CGO_ENABLED=0 go build $(actual_build_flags)

bin:
	@mkdir ./bin

$(LINTER_BIN): linter_bin_path := $(shell which $(LINTER_BIN))
$(LINTER_BIN):
	@if [ -z $(linter_bin_path) ] || [ ! -x $(linter_bin_path) ]; then \
		curl -sfL https://install.goreleaser.com/github.com/golangci/golangci-lint.sh | sh -s -- -b $(shell go env GOPATH)/bin v1.15.0; \
	fi

.PHONY: mod
mod:
	@go mod tidy

.PHONY: check-mod
check-mod: # verifies that module changes for go.mod and go.sum are checked in
	@hack/ci/check_mods.sh

.PHONY: vendor
vendor: mod
	@go mod vendor -v

