AUTOTAGS := $(shell ./btrfs_tag.sh) $(shell ./libdm_tag.sh)

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
