lint:
	golangci-lint run ./...

.PHONY: test-deps
test-deps:
	cd testdata/src/a && go mod vendor

.PHONY: test
test: test-deps
	go test ./...

.PHONY: build
build:
	go build -o bin/logrlint ./cmd/logrlint

.PHONY: build-plugin
build-plugin:
	CGO_ENABLED=1 go build -o bin/logrlint.so -buildmode=plugin ./plugin
