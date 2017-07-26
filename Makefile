AUTOTAGS := $(shell ./btrfs_tag.sh) $(shell ./libdm_tag.sh)
TAGS := seccomp
PREFIX := /usr/local
BINDIR := $(PREFIX)/bin
BASHINSTALLDIR=${PREFIX}/share/bash-completion/completions
BUILDFLAGS := -tags "$(AUTOTAGS) $(TAGS)"

GIT_COMMIT := $(shell git rev-parse --short HEAD)
BUILD_INFO := $(shell date +%s)

RUNC_COMMIT := c5ec25487693612aed95673800863e134785f946
LIBSECCOMP_COMMIT := release-2.3

LDFLAGS := -ldflags '-X main.gitCommit=${GIT_COMMIT} -X main.buildInfo=${BUILD_INFO}'

all: buildah imgtype docs

buildah: *.go imagebuildah/*.go cmd/buildah/*.go docker/*.go util/*.go
	go build $(LDFLAGS) -o buildah $(BUILDFLAGS) ./cmd/buildah

imgtype: *.go docker/*.go util/*.go tests/imgtype.go
	go build $(LDFLAGS) -o imgtype $(BUILDFLAGS) ./tests/imgtype.go

test-unit:
	cd tests && ./unit-tests.sh

test-integration:
	cd tests && ./test_runner.sh

.PHONY: clean
clean:
	$(RM) buildah imgtype
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

.PHONY: runc
runc: gopath
	rm -rf ../../opencontainers/runc
	git clone https://github.com/opencontainers/runc ../../opencontainers/runc
	cd ../../opencontainers/runc && git checkout $(RUNC_COMMIT) && go build -tags "$(AUTOTAGS) $(TAGS)"
	ln -sf ../../opencontainers/runc/runc

.PHONY: install.libseccomp.sudo
install.libseccomp.sudo: gopath
	rm -rf ../../seccomp/libseccomp
	git clone https://github.com/seccomp/libseccomp ../../seccomp/libseccomp
	cd ../../seccomp/libseccomp && git checkout $(LIBSECCOMP_COMMIT) && ./autogen.sh && ./configure --prefix=/usr && make all && sudo make install

.PHONY: install
install:
	install -D -m0755 buildah $(DESTDIR)/$(BINDIR)/buildah
	$(MAKE) -C docs install

.PHONY: install.completions
install.completions:
	install -m 644 -D contrib/completions/bash/buildah $(DESTDIR)/${BASHINSTALLDIR}/buildah
