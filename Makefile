AUTOTAGS := $(shell ./btrfs_tag.sh) $(shell ./libdm_tag.sh)
PREFIX := $(DESTDIR)/usr/local
BINDIR := $(PREFIX)/bin
BASHINSTALLDIR=${PREFIX}/share/bash-completion/completions

all: buildah install.tools docs

buildah: *.go cmd/buildah/*.go
	go build -o buildah -tags "$(AUTOTAGS) $(TAGS)" ./cmd/buildah

.PHONY: clean
clean:
	$(RM) buildah
	$(MAKE) -C docs clean 

.PHONY: docs
docs: ## build the docs on the host
	$(MAKE) -C docs docs

# For vendoring to work right, the checkout directory must be such that our top
# level is at $GOPATH/src/github.com/projectatomic/buildah.
.PHONY: gopath
gopath:
	test $(shell pwd) = $(shell cd ../../../../src/github.com/projectatomic/buildah ; pwd)

# We use https://github.com/lk4d4/vndr to manage dependencies.
.PHONY: deps
deps: gopath
	env GOPATH=$(shell cd ../../../.. ; pwd) vndr

install:
	install -D -m0755 buildah $(BINDIR)/buildah
	$(MAKE) -C docs install


.PHONY: install.tools

install.tools: .install.md2man

.install.md2man:
	go get github.com/cpuguy83/go-md2man


.PHONY: install.completions
install.completions:
	install -d -m 755 ${BASHINSTALLDIR}
	install -m 644 -D contrib/completions/bash/buildah ${BASHINSTALLDIR}
