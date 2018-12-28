.ONESHELL:
.PHONY: test test-goenv test-goenv-go-build bats start-fake-go-build-http-server stop-fake-go-build-http-server run-goenv-go-build-tests

default: test

test: test-goenv test-goenv-go-build

test-goenv: bats
	set -e
	PATH="./bats-core/bin:$$PATH"

	if [ -n "$$GOENV_NATIVE_EXT" ]; then
		src/configure
		make -C src
	fi

	test_target=$${test_target:-test}
	exec bats $${CI:+--tap} $$test_target

test-goenv-go-build: bats stop-fake-go-build-http-server start-fake-go-build-http-server run-goenv-go-build-tests stop-fake-go-build-http-server

stop-fake-go-build-http-server:
	pkill fake_file_server || true

run-goenv-go-build-tests:
	set -e
	PATH="$$(pwd)/bats-core/bin:$$PATH"

	if [ -n "$$GOENV_NATIVE_EXT" ]; then
		src/configure
		make -C src
	fi

	test_target=$${test_target:-test}

	cd plugins/go-build
	exec bats $${CI:+--tap} $$test_target

start-fake-go-build-http-server:
	set -e

	port=$${port:-8090}
	cd plugins/go-build/test
	(bash -c "exec -a fake_file_server python3 fake_file_server.py $$port") &

	until lsof -Pi :$${port} -sTCP:LISTEN -t >/dev/null; do
		echo "wait"
		sleep 2
	done

bats:
	set -e

	if [ -d "$(PWD)/bats-core" ]; then
		echo "bats-core already exists. Nothing to do" ;
	else
		git clone --depth 1 https://github.com/bats-core/bats-core.git ;
	fi
