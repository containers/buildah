MANDIR=/usr/share/man/man1
GOMD2MAN = go-md2man

docs: $(patsubst %.md,%.1,$(wildcard *.md))

%.1: %.md
	$(GOMD2MAN) -in $^ -out $@

.PHONY: install
install:
	install -m 0644 buildah*.1 ${MANDIR}

.PHONY: clean
clean:
	$(RM) $(MANDIR)/buildah*.1
	$(RM) buildah*.1
