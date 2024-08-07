GO := go
GO_BUILD=$(GO) build

BUILDDIR := build

all: $(BUILDDIR)

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify
	if test -n "$(strip $(shell go env GOTOOLCHAIN))"; then go mod edit -toolchain none ; fi

vendor-in-container:
	if test -d `go env GOCACHE` && test -w `go env GOCACHE` ; then \
		podman run --privileged --rm --env HOME=/root -v `go env GOCACHE`:/root/.cache/go-build --env GOCACHE=/root/.cache/go-build -v `pwd`:/src -w /src docker.io/library/golang:1.21 make vendor ; \
	else \
		podman run --privileged --rm --env HOME=/root -v `pwd`:/src -w /src docker.io/library/golang:1.21 make vendor ; \
	fi

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)

.PHONY: $(BUILDDIR)
$(BUILDDIR): \
	$(BUILDDIR)/go-md2man \
	$(BUILDDIR)/golangci-lint

$(BUILDDIR)/go-md2man:
	$(GO_BUILD) -o $@ ./vendor/github.com/cpuguy83/go-md2man/v2

$(BUILDDIR)/golangci-lint:
	$(GO_BUILD) -o $@ ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint
