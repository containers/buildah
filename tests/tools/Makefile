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
