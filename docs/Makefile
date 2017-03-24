GOMD2MAN = go-md2man

docs: $(patsubst %.md,%.1,$(wildcard *.md))

%.1: %.md
	$(GOMD2MAN) -in $^ -out $@
