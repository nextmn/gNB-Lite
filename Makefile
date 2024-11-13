prefix = /usr/local
exec_prefix = $(prefix)
bindir = $(exec_prefix)/bin
BASHCOMPLETIONSDIR = $(exec_prefix)/share/bash-completion/completions


RM = rm -f
INSTALL = install -D

.PHONY: install uninstall build clean default
default: build
build:
	go build
clean:
	go clean
reinstall: uninstall install
install:
	$(INSTALL) gnb-lite $(DESTDIR)$(bindir)/gnb-lite
	$(INSTALL) bash-completion/completions/gnb-lite $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite
	@echo "================================="
	@echo ">> Now run the following command:"
	@echo -e "\tsource $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite"
	@echo "================================="
uninstall:
	$(RM) $(DESTDIR)$(bindir)/gnb-lite
	$(RM) $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite
