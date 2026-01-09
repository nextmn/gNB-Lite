prefix = /usr/local
exec_prefix = $(prefix)
bindir = $(exec_prefix)/bin
BASHCOMPLETIONSDIR = $(exec_prefix)/share/bash-completion/completions


RM = rm -f
INSTALL = install -D
MKDIRP = mkdir -p

.PHONY: install uninstall build clean default
default: build
build:
	go build
clean:
	go clean
reinstall: uninstall install
install:
	$(INSTALL) gnb-lite $(DESTDIR)$(bindir)/gnb-lite
	$(MKDIRP) $(DESTDIR)$(BASHCOMPLETIONSDIR)
	$(DESTDIR)$(bindir)/gnb-lite completion bash > $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite
	@echo "================================="
	@echo ">> Now run the following command:"
	@echo "\tsource $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite"
	@echo "================================="
uninstall:
	$(RM) $(DESTDIR)$(bindir)/gnb-lite
	$(RM) $(DESTDIR)$(BASHCOMPLETIONSDIR)/gnb-lite
