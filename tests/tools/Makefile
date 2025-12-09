GO := go
# test for go module support
ifeq ($(shell go help mod >/dev/null 2>&1 && echo true), true)
export GO_BUILD=GO111MODULE=on $(GO) build -mod=vendor
else
export GO_BUILD=$(GO) build
endif

BUILDDIR := build

all: $(BUILDDIR)

.PHONY: vendor
vendor:
	GO111MODULE=on $(GO) mod tidy
	GO111MODULE=on $(GO) mod vendor
	GO111MODULE=on $(GO) mod verify

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)

$(BUILDDIR): \
	$(BUILDDIR)/git-validation \
	$(BUILDDIR)/go-md2man \
	$(BUILDDIR)/golangci-lint

$(BUILDDIR)/git-validation:
	$(GO_BUILD) -o $@ ./vendor/github.com/vbatts/git-validation

$(BUILDDIR)/go-md2man:
	$(GO_BUILD) -o $@ ./vendor/github.com/cpuguy83/go-md2man

$(BUILDDIR)/golangci-lint:
	$(GO_BUILD) -o $@ ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint
