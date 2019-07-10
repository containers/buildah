export GO111MODULE=off

GO := go

BUILDDIR := build

all: $(BUILDDIR)

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify

define go-build
	$(shell cd `pwd` && $(GO) build -o $(BUILDDIR)/$(shell basename $(1)) $(1))
	@echo > /dev/null
endef

.PHONY: clean
clean:
	rm -rf $(BUILDDIR)

$(BUILDDIR): \
	$(BUILDDIR)/ginkgo \
	$(BUILDDIR)/git-validation \
	$(BUILDDIR)/go-md2man \
	$(BUILDDIR)/golangci-lint

$(BUILDDIR)/ginkgo:
	$(call go-build,./vendor/github.com/onsi/ginkgo/ginkgo)

$(BUILDDIR)/git-validation:
	$(call go-build,./vendor/github.com/vbatts/git-validation)

$(BUILDDIR)/go-md2man:
	$(call go-build,./vendor/github.com/cpuguy83/go-md2man)

$(BUILDDIR)/golangci-lint:
	$(call go-build,./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint)
