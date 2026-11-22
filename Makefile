# this is the what ends up in the RPM "Version" field and embedded in the --version CLI flag
VERSION ?= $(shell .ci/get_version_from_git.sh)
PREFIX ?= /usr
SBINDIR ?= $(PREFIX)/sbin
BINDIR ?= $(PREFIX)/bin
LIBDIR ?= $(PREFIX)/lib
SHAREDIR ?= $(PREFIX)/share
TEMPLATEDIR ?= $(SHAREDIR)/hawk/templates
STATICDIR ?= $(SHAREDIR)/hawk/static

default: build test
build: tools
	go vet ./...
	go build -ldflags "-s -w -X main.version=$(VERSION)"
	go mod tidy
test:
	go test ./... -v
tools/common.h: tools/common.h.in Makefile
	sed -e 's|@LIBDIR@|$(LIBDIR)|g' \
	    -e 's|@BINDIR@|$(BINDIR)|g' \
	    -e 's|@SBINDIR@|$(SBINDIR)|g' \
	    -e 's|@WITHIN_VAGRANT@|0|g' $< > $@

tools/hawk_chkpwd: tools/hawk_chkpwd.c tools/common.h
	$(CC) $(CPPFLAGS) $(CFLAGS) -fpie -pie $(LDFLAGS) -o $@ $< $(LDLIBS) -lpam

tools: tools/hawk_chkpwd

install: tools
	install -D -m 0755 hawk-apiserver "$(DESTDIR)$(SBINDIR)/hawk-apiserver"
	install -D -o root -g haclient -m 4750 tools/hawk_chkpwd "$(DESTDIR)$(SBINDIR)/hawk_chkpwd"
	install -d "$(DESTDIR)$(TEMPLATEDIR)"
	install -m 0644 templates/* "$(DESTDIR)$(TEMPLATEDIR)/"
	install -d "$(DESTDIR)$(STATICDIR)"
	cp -R static/* "$(DESTDIR)$(STATICDIR)/"

clean:
	rm -f hawk-apiserver tools/hawk_chkpwd tools/common.h

.PHONY: default build test tools install clean
