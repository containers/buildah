AUTOTAGS := $(shell ./btrfs_tag.sh) $(shell ./libdm_tag.sh)
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
BASHINSTALLDIR=${PREFIX}/share/bash-completion/completions
BUILDFLAGS := -tags "$(AUTOTAGS) $(TAGS)"

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_INFO := $(shell date +%s)

LDFLAGS := -ldflags '-X main.gitCommit=${GIT_COMMIT} -X main.buildInfo=${BUILD_INFO}'

all: buildah imgtype docs

buildah: *.go imagebuildah/*.go cmd/buildah/*.go docker/*.go util/*.go
	go build $(LDFLAGS) -o buildah $(BUILDFLAGS) ./cmd/buildah

imgtype: *.go docker/*.go util/*.go tests/imgtype.go
	go build $(LDFLAGS) -o imgtype $(BUILDFLAGS) ./tests/imgtype.go

.PHONY: clean
clean:
	$(RM) buildah
	$(MAKE) -C docs clean 

.PHONY: docs
docs: ## build the docs on the host
	$(MAKE) -C docs

# For vendoring to work right, the checkout directory must be such that our top
# level is at $GOPATH/src/github.com/projectatomic/buildah.
.PHONY: gopath
gopath:
	test $(shell pwd) = $(shell cd ../../../../src/github.com/projectatomic/buildah ; pwd)

# We use https://github.com/lk4d4/vndr to manage dependencies.
.PHONY: deps
deps: gopath
	env GOPATH=$(shell cd ../../../.. ; pwd) vndr

.PHONY: validate
validate:
	@./tests/validate/gofmt.sh
	@./tests/validate/govet.sh
	@./tests/validate/git-validation.sh
	@./tests/validate/gometalinter.sh . cmd/buildah

.PHONY: install.tools
install.tools:
	go get -u $(BUILDFLAGS) github.com/cpuguy83/go-md2man
	go get -u $(BUILDFLAGS) github.com/vbatts/git-validation
	go get -u $(BUILDFLAGS) gopkg.in/alecthomas/gometalinter.v1
	gometalinter.v1 -i

.PHONY: install
install:
	install -D -m0755 buildah $(DESTDIR)/$(BINDIR)/buildah
	$(MAKE) -C docs install

.PHONY: install.completions
install.completions:
	install -m 644 -D contrib/completions/bash/buildah $(DESTDIR)/${BASHINSTALLDIR}/buildah
