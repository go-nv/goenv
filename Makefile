.PHONY: test

test: bats
	PATH="./bats-core/bin:$$PATH" test/run
	cd plugins/go-build && $(PWD)/bats-core/bin/bats $${CI:+--tap} test

bats:
	if [ -d "$(PWD)/bats-core" ]; then \
		echo "bats-core already exists. Nothing to do"; \
	else \
		git clone --depth 1 https://github.com/bats-core/bats-core.git; \
	fi
