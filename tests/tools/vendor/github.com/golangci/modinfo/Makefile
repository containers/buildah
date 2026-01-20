.PHONY: clean check test

default: clean check test

clean:
	rm -rf dist/ cover.out

test: clean
	go test -v -cover ./...

check:
	golangci-lint run
