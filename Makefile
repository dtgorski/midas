.PHONY: help clean test bench sniff .travis

SUBLINE := Fast and cheap HTTP access logger middleware for Go
PACKAGE := $(shell basename $$(grep ^module go.mod | head -n1 | cut -d" " -f2))
VERSION := $(shell 2>/dev/null git describe --always --long --dirty --tags)
SOURCES := $(shell 2>/dev/null git config --get remote.origin.url | sed "s/^.*:\/\///")

help:                   # Displays this list
	@echo
	@echo " \e[1m$(PACKAGE) $(VERSION)\e[0m - $(SOURCES)"
	@echo " $(SUBLINE)\n"
	@cat Makefile \
	    | grep "^[a-z][a-zA-Z0-9_<> -]\+:" \
	    | sed -r "s/:[^#]*?#?(.*)?/\r\t\t\1/" \
	    | sed "s/^/ make /" && echo

clean:                  # Removes build/test artifacts
	@find . -type f | grep "\.out$$" | xargs -I{} rm {};
	@find . -type f | grep "\.html$$" | xargs -I{} rm {};
	@find . -type f | grep "\.test$$" | xargs -I{} rm {};

test: clean             # Runs integrity test with -race
	CGO_ENABLED=1 go test -v -count=1 -race -covermode=atomic -coverprofile=./coverage.out .
	@go tool cover -html=./coverage.out -o ./coverage.html && echo "coverage: <file://$(PWD)/coverage.html>"

bench: clean            # Executes artificial benchmarks
	CGO_ENABLED=0 go test -run=^$$ -bench=. -benchmem

sniff:                  # Format check & linter (void on success)
	@gofmt -d .
	@golint ./...

.travis:                # Travis CI (see .travis.yml), runs tests
    ifndef TRAVIS
	    @echo "Fail: requires Travis runtime"
    else
	    @$(MAKE) test --no-print-directory && \
	    goveralls -coverprofile=./coverage.out -service=travis-ci
    endif
