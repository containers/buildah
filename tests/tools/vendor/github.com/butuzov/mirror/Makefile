# --- Required ----------------------------------------------------------------
export PATH   := $(PWD)/bin:$(PATH)                    # ./bin to $PATH
export SHELL  := bash                                  # Default Shell

GOPKGS := $(shell go list ./... | grep -vE "(cmd|sandbox|testdata)" | tr -s '\n' ',' | sed 's/.\{1\}$$//' )


build:
	@ go build -trimpath -ldflags="-w -s" \
		-o bin/mirror ./cmd/mirror/

build-race:
	@ go build -race -trimpath -ldflags="-w -s" \
		-o bin/mirror ./cmd/mirror/

tests:
	go test -v -count=1 -race \
		-failfast \
		-parallel=2 \
		-timeout=1m \
		-covermode=atomic \
		-coverpkg=$(GOPKGS) -coverprofile=coverage.cov ./...

tests-summary:
	go test -v -count=1 -race \
		-failfast \
		-parallel=2 \
		-timeout=1m \
		-covermode=atomic \
		-coverpkg=$(GOPKGS) -coverprofile=coverage.cov --json ./... | tparse -all

test-generate:
	go run ./cmd/internal/generate-tests/ "$(PWD)/testdata"

lints:
	golangci-lint run --no-config ./... -D deadcode --skip-dirs "^(cmd|sandbox|testdata)"


cover:
	go tool cover -html=coverage.cov

install:
	go install -trimpath -v -ldflags="-w -s" \
		./cmd/mirror

funcs:
	echo "" > "out/results.txt"
	go list std | grep -v "vendor" | grep -v "internal" | \
		xargs -I {} sh -c 'go doc -all {} > out/$(basename {}).txt'

bin/goreleaser:
	@curl -Ls https://github.com/goreleaser/goreleaser/releases/download/v1.17.2/goreleaser_Darwin_all.tar.gz | tar -zOxf - goreleaser > ./bin/goreleaser
	chmod 0755 ./bin/goreleaser

test-release: bin/goreleaser
	goreleaser release --help
	goreleaser release -f .goreleaser.yaml \
		--skip-validate --skip-publish --clean
