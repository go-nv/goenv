SHELL:=/bin/bash
.ONESHELL:
.PHONY: test test-goenv test-goenv-go-build bats start-fake-go-build-http-server stop-fake-go-build-http-server run-goenv-go-build-tests
MAKEFLAGS += -s

ifeq (test-target,$(firstword $(MAKECMDGOALS)))
  # use the rest as arguments for "test-target"
  TEST_TARGET_ARGS := $(wordlist 2,$(words $(MAKECMDGOALS)),$(MAKECMDGOALS))
  # ...and turn them into do-nothing targets
  $(shell echo $(TEST_TARGET_ARGS):;@:)
  $(eval $(TEST_TARGET_ARGS):;@:)
endif

default: test

test: test-goenv test-goenv-go-build

# USAGE: make -- test-target [args..]
test-target: bats
	set -e; \
	PATH="./bats-core/bin:$$PATH"; \
	if [ -n "$$GOENV_NATIVE_EXT" ]; then \
		src/configure; \
		make -C src; \
	fi; \
	unset $${!GOENV_*}; \
	test_target=$${test_target:-test}; \
	exec bats $(TEST_TARGET_ARGS);

test-goenv: bats
	set -e; \
	PATH="./bats-core/bin:$$PATH"; \
	if [ -n "$$GOENV_NATIVE_EXT" ]; then \
		src/configure; \
		make -C src; \
	fi; \
	unset $${!GOENV_*}; \
	test_target=$${test_target:-test}; \
	exec bats $${CI:+--tap} $$test_target;

test-goenv-go-build: bats stop-fake-go-build-http-server start-fake-go-build-http-server run-goenv-go-build-tests stop-fake-go-build-http-server

stop-fake-go-build-http-server:
	pkill fake_file_server || true

run-goenv-go-build-tests:
	set -e; \
	PATH="$$(pwd)/bats-core/bin:$$PATH"; \
	if [ -n "$$GOENV_NATIVE_EXT" ]; then \
		src/configure; \
		make -C src; \
	fi; \
	unset $${!GOENV_*}; \
	test_target=$${test_target:-test}; \
	cd plugins/go-build; \
	exec bats $${CI:+--tap} $$test_target;

start-fake-go-build-http-server:
	set -e; \
	port=$${port:-8090}; \
	cd plugins/go-build/test; \
	(bash -c "exec -a fake_file_server python3 fake_file_server.py $$port") & \
	until lsof -Pi :$${port} -sTCP:LISTEN -t >/dev/null; do \
		echo "wait"; \
		sleep 2; \
	done;

bats:
	set -e; \
	if [ -d "$(PWD)/bats-core" ]; then \
		echo "bats-core already exists. Nothing to do"; \
	else \
		git clone --depth 1 --single-branch --branch=v1.10.0 https://github.com/bats-core/bats-core.git; \
	fi;
