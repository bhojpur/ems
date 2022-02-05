PREFIX=/usr/local
BINDIR=${PREFIX}/bin
DESTDIR=
BLDDIR = build
BLDFLAGS=
EXT=
ifeq (${GOOS},windows)
    EXT=.exe
endif

APPS = emsd emslookupd emsadmin ems_to_ems ems_to_file ems_to_http ems_tail ems_stat to_ems
all: $(APPS)

$(BLDDIR)/emsd:        $(wildcard apps/emsd/*.go       emsd/*.go       ems/*.go internal/*/*.go)
$(BLDDIR)/emslookupd:  $(wildcard apps/emslookupd/*.go emslookupd/*.go ems/*.go internal/*/*.go)
$(BLDDIR)/emsadmin:    $(wildcard apps/emsadmin/*.go   emsadmin/*.go emsadmin/templates/*.go internal/*/*.go)
$(BLDDIR)/ems_to_ems:  $(wildcard apps/ems_to_ems/*.go  ems/*.go internal/*/*.go)
$(BLDDIR)/ems_to_file: $(wildcard apps/ems_to_file/*.go ems/*.go internal/*/*.go)
$(BLDDIR)/ems_to_http: $(wildcard apps/ems_to_http/*.go ems/*.go internal/*/*.go)
$(BLDDIR)/ems_tail:    $(wildcard apps/ems_tail/*.go    ems/*.go internal/*/*.go)
$(BLDDIR)/ems_stat:    $(wildcard apps/ems_stat/*.go             internal/*/*.go)
$(BLDDIR)/to_ems:      $(wildcard apps/to_ems/*.go               internal/*/*.go)

$(BLDDIR)/%:
	@mkdir -p $(dir $@)
	go build ${BLDFLAGS} -o $@ ./apps/$*

$(APPS): %: $(BLDDIR)/%

clean:
	rm -fr $(BLDDIR)

.PHONY: install clean all
.PHONY: $(APPS)

install: $(APPS)
	install -m 755 -d ${DESTDIR}${BINDIR}
	for APP in $^ ; do install -m 755 ${BLDDIR}/$$APP ${DESTDIR}${BINDIR}/$$APP${EXT} ; done