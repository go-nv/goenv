.ONESHELL:
.PHONY: test test-goenv test-goenv-go-build

test: test-goenv
#test: test-goenv test-goenv-go-build

test-goenv: bats
	set -e
	PATH="./bats-core/bin:$$PATH"

	if [ -n "$$GOENV_NATIVE_EXT" ]; then
	  src/configure
	  make -C src
	fi

	test_target=$${test_target:-test}
	exec bats $${CI:+--tap} $$test_target

test-goenv-go-build: bats
	set -e

	PATH="$$(pwd)/bats-core/bin:$$PATH"
	test_target=$${test_target:-test}

	cd plugins/go-build
	exec bats $${CI:+--tap} $$test_target

bats:
	set -e

	if [ -d "$(PWD)/bats-core" ]; then
		echo "bats-core already exists. Nothing to do" ;
	else
		git clone --depth 1 https://github.com/bats-core/bats-core.git ;
	fi
