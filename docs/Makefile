PREFIX := /usr/local
DATADIR := ${PREFIX}/share
MANDIR := $(DATADIR)/man
# Following go-md2man is guaranteed on host
GOMD2MAN ?= ../tests/tools/build/go-md2man

docs: $(patsubst %.md,%,$(wildcard *.md))

%.5:  %.5.md
	$(GOMD2MAN) -in $^ -out $@

%.1: %.1.md
### sed is used to filter http/s links as well as relative links
### replaces "\" at the end of a line with two spaces
### this ensures that manpages are renderd correctly

	@sed -e 's/\((buildah[^)]*\.md\(#.*\)\?)\)//g' \
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
	install -d ${DESTDIR}/${MANDIR}/man5
	install -m 0644 *.5 ${DESTDIR}/${MANDIR}/man5

.PHONY: clean
clean:
	$(RM) buildah*.1
	$(RM) *.5
