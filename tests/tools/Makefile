GO := go

BUILDDIR := build

.PHONY: vendor
vendor:
	export GO111MODULE=on \
		$(GO) mod tidy && \
		$(GO) mod vendor && \
		$(GO) mod verify
.PHONY: build
build:
	$(GO) build -o $(BUILDDIR)/go-md2man ./vendor/github.com/cpuguy83/go-md2man
	$(GO) build -o $(BUILDDIR)/git-validation ./vendor/github.com/vbatts/git-validation
	$(GO) build -o $(BUILDDIR)/ginkgo ./vendor/github.com/onsi/ginkgo/ginkgo
	$(GO) build -o $(BUILDDIR)/golangci-lint ./vendor/github.com/golangci/golangci-lint/cmd/golangci-lint
