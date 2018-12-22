.PHONY: test

# Do not pass in user flags to build tests.
unexport GOGCCFLAGS

test: bats
	PATH="./bats/bin:$$PATH" test/run
	cd plugins/go-build && $(PWD)/bats-core/bin/bats $${CI:+--tap} test

bats:
	if [ -d "$(PWD)/bats-core" ]; then \
		echo "bats-core already exists. Nothing to do"; \
	else \
		git clone https://github.com/bats-core/bats-core.git; \
	fi
