AUTOTAGS := $(shell ./btrfs_tag.sh) $(shell ./libdm_tag.sh)
PREFIX := $(DESTDIR)/usr/local
BINDIR := $(PREFIX)/sbin
BASHINSTALLDIR=${PREFIX}/share/bash-completion/completions

all: buildah

buildah: *.go cmd/buildah/*.go
	go build -o buildah -tags "$(AUTOTAGS) $(TAGS)" ./cmd/buildah

.PHONY: clean
clean:
	$(RM) buildah

# For vendoring to work right, the checkout directory must be such that out top
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

.PHONY: install.completions
install.completions:
	install -d -m 755 ${BASHINSTALLDIR}
	install -m 644 -D contrib/completions/bash/buildah ${BASHINSTALLDIR}
