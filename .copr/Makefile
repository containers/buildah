#!/usr/bin/make -f

spec := contrib/rpm/buildah_copr.spec
outdir := $(CURDIR)
tmpdir := build
gitdir := $(PWD)/.git

rev := $(shell sed 's/\(.......\).*/\1/' $(gitdir)/$$(sed -n '/^ref:/{s/.* //;p}' $(gitdir)/HEAD))
date := $(shell date +%Y%m%d.%H%M)

version := $(shell sed -n '/Version:/{s/.* //;p}' $(spec))
release := $(date).git.$(rev)

srpm: $(outdir)/buildah-$(version)-$(release).src.rpm

$(tmpdir)/buildah.spec: $(spec)
	@mkdir -p $(tmpdir)
	sed '/^Release:/s/\(: *\).*/\1$(release)%{?dist}/' $< >$@

$(tmpdir)/$(version).tar.gz: $(gitdir)/..
	@mkdir -p $(tmpdir)
	tar c --exclude-vcs --exclude-vcs-ignores -C $< --transform 's|^\.|buildah-$(version)|' . | gzip -9 >$@

$(outdir)/buildah-$(version)-$(release).src.rpm: $(tmpdir)/buildah.spec $(tmpdir)/$(version).tar.gz
	@mkdir -p $(outdir)
	rpmbuild -D'_srcrpmdir $(outdir)' -D'_sourcedir $(tmpdir)' -bs $(tmpdir)/buildah.spec

.PHONY: srpm
