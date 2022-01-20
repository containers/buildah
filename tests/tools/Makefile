GO := go
GO_BUILD=$(GO) build

BUILDDIR := build

all: $(BUILDDIR)

.PHONY: vendor
vendor:
	$(GO) mod tidy
	$(GO) mod vendor
	$(GO) mod verify

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)

.PHONY: $(BUILDDIR)
$(BUILDDIR): \
	$(BUILDDIR)/ginkgo \
	$(BUILDDIR)/go-md2man \
	$(BUILDDIR)/golangci-lint

$(BUILDDIR)/ginkgo:
	$(GO_BUILD) -o $@ ./vendor/github.com/onsi/ginkgo/ginkgo

$(BUILDDIR)/go-md2man:
	$(GO_BUILD) -o $@ ./vendor/github.com/cpuguy83/go-md2man/v2

$(BUILDDIR)/golangci-lint:
	$(GO_BUILD) -o $@ ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint
