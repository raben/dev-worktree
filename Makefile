.PHONY: test lint

# bash 5+ is required (associative arrays in lib/dev/utils.sh).
# macOS ships bash 3.2; install bash 5 via: brew install bash
BREW_PREFIX := $(shell brew --prefix 2>/dev/null || echo /opt/homebrew)
export PATH := $(BREW_PREFIX)/bin:$(PATH)

# Run all bats tests
test:
	bats test/

# Run a specific test file: make test-utils
test-%:
	bats test/$*.bats

# Syntax check bin/dev
lint:
	bash -n bin/dev
