PREFIX := /usr/local
DATADIR := ${PREFIX}/share
MANDIR := $(DATADIR)/man
# Following go-md2man is guaranteed on host
GOMD2MAN ?= ../tests/tools/build/go-md2man
ifeq ($(shell uname -s),FreeBSD)
SED=gsed
else
SED=sed
endif

docs: $(patsubst %.md,%,$(wildcard *.md))

%.1: %.1.md
### sed is used to filter http/s links as well as relative links
### replaces "\" at the end of a line with two spaces
### this ensures that manpages are rendered correctly
	@$(SED) -e 's/\((buildah[^)]*\.md\(#.*\)\?)\)//g' \
	 -e 's/\[\(buildah[^]]*\)\]/\1/g' \
	 -e 's/\[\([^]]*\)](http[^)]\+)/\1/g' \
	 -e 's;<\(/\)\?\(a\|a\s\+[^>]*\|sup\)>;;g' \
	 -e 's/\\$$/  /g' $<  | \
	$(GOMD2MAN) -in /dev/stdin -out  $@

.PHONY: install
install:
	install -d ${DESTDIR}/${MANDIR}/man1
	install -m 0644 buildah*.1 ${DESTDIR}/${MANDIR}/man1
	install -m 0644 links/buildah*.1 ${DESTDIR}/${MANDIR}/man1

.PHONY: clean
clean:
	$(RM) buildah*.1
