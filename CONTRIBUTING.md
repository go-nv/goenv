#  Contributing

The goenv source code is [hosted on GitHub](https://github.com/syndbg/goenv). 
It's clean, modular, and easy to understand, even if you're not a shell hacker. (I hope)

Tests are executed using [Bats](https://github.com/bats-core/bats-core).

Please feel free to submit pull requests and file bugs on the [issue tracker](https://github.com/syndbg/goenv/issues).

## Prerequisites

* Linux with any (or more than 1) of `zsh`, `bash`, `zsh`.

## Common commands

### Running the tests for both `goenv` and `goenv-go-build`

```shell
> make test
```

### Running the tests only for `goenv`

```shell
> make test-goenv
```

### Running the tests only for `goenv-go-build`

```shell
> make test-goenv-go-build
```

### Others

Check the [Makefile](./Makefile)

## Workflows

### Submitting an issue

1. Check existing issues and verify that your issue is not already submitted.
 If it is, it's highly recommended to add  to that issue with your reports.
2. Open issue
3. Be as detailed as possible - Linux distribution, shell, what did you do, 
what did you expect to happen, what actually happened.

### Submitting a PR

1. Find an existing issue to work on or follow `Submitting an issue` to create one
 that you're also going to fix. 
 Make sure to notify that you're working on a fix for the issue you picked.
1. Branch out from latest `master`.
1. Code, add, commit and push your changes in your branch.
1. Make sure that tests (or let the CI do the heavy work for you).
1. Submit a PR.
1. Make sure to clarify if the PR is ready for review or work-in-progress.
 A simple `[WIP]` (in any form) is a good indicator whether the PR is still being actively developed.
1. Collaborate with the codeowners/reviewers to merge this in `master`.

### Release process

Described in details at [RELEASE_PROCESS](./RELEASE_PROCESS.md).
