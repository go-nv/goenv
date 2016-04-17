.PHONY: test

# Do not pass in user flags to build tests.
unexport GOGCCFLAGS

test: bats
	PATH="./bats/bin:$$PATH" test/run
	cd plugins/go-build && $(PWD)/bats/bin/bats $${CI:+--tap} test

bats:
	git clone https://github.com/sstephenson/bats.git
