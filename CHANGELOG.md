# Version History

## Intro

The version history is motivated by https://semver.org/ and https://keepachangelog.com/en/1.0.0/ .

NOTE: This project went from non-standard versioning to semver at some point.

## Structure

Types of changes that can be seen in the changelog

```
Added: for new features/functionality.
Changed: for changes in existing features/functionality.
Deprecated: for soon-to-be removed features. Removed in the
Removed: for now removed features.
Fixed: for any bug fixes.
Security: in case of vulnerabilities.
```

## How deprecation of functionality is handled?

tl;dr 1 minor release stating that the functionality is going to be deprecated. Then in the next major - removed.

```
Deprecating existing functionality is a normal part of software development and
is often required to make forward progress.

When you deprecate part of your public API, you should do two things:

(1) update your documentation to let users know about the change,
(2) issue a new minor release with the deprecation in place.
Before you completely remove the functionality in a new major
release there should be at least one minor release
that contains the deprecation so that users can smoothly transition to the new API
```

As per https://semver.org/ .

As per rule-of-thumb, moving the project forward is very important,
but providing stability is the most important thing to anyone using `goenv`.

Introducing breaking changes under a feature flag can be ok in some cases where new functionality needs user feedback before being introduced in next major release.

## Changelog

Change line format:

```
* <Change title/PR title/content> ; Ref: <pr link>
```

## Unreleased (master)

## 2.0.5

### Added

- Support Golang 1.18.9 and 1.19.4 https://github.com/syndbg/goenv/pull/277
- Support 1.20rc1 https://github.com/syndbg/goenv/pull/278
- Support 1.18.10, 1.19.5, 1.20rc2: https://github.com/syndbg/goenv/pull/282

## 2.0.4

### Added

- Resolve init rehash issue Ref: https://github.com/syndbg/goenv/pull/275

## 2.0.3

### Added

- Make it so tests dont look at real definitions; fix flaky test Ref: https://github.com/syndbg/goenv/pull/269
- [goenv-bot]: Add 1.19.3 1.18.8 definition to goenv Ref: https://github.com/syndbg/goenv/pull/268
- move latest patch code below once definition path is set by goenv local Ref: https://github.com/syndbg/goenv/pull/272
- Ensure install doesn't exit silently when no installable definition found Ref: https://github.com/syndbg/goenv/pull/273

## 2.0.2

### Added

- fix version printout for `goenv --version`; update changelog Ref: https://github.com/syndbg/goenv/pull/260

## 2.0.1

### Added

- install latest patch in go-build script Ref: https://github.com/syndbg/goenv/pull/258
- Download all packages from go.dev Ref: https://github.com/syndbg/goenv/pull/218
- move shims to end of $PATH Ref: https://github.com/syndbg/goenv/pull/248
- fix tests Ref: https://github.com/syndbg/goenv/pull/259

## 2.0.0

### Added

- Prepare goenv 2.0.0beta1 Ref: https://github.com/syndbg/goenv/pull/62
- Follow up of PR #56 Ref: https://github.com/syndbg/goenv/pull/63
- :tada: add 1.12beta2 Ref: https://github.com/syndbg/goenv/pull/64
- add 1.11.5 and 1.10.8 Ref: https://github.com/syndbg/goenv/pull/65
- add 1.12rc1 Ref: https://github.com/syndbg/goenv/pull/66
- add 1.12.0 Ref: https://github.com/syndbg/goenv/pull/68
- [GH-30][gh-50] Improve GOPATH and GOROOT env var management Ref: https://github.com/syndbg/goenv/pull/70
- add 1.12.1 and 1.11.6 Ref: https://github.com/syndbg/goenv/pull/71
- add 1.12.2, 1.12.3, 1.11.7 and 1.11.8 Ref: https://github.com/syndbg/goenv/pull/73
- Prepare 2.0.0beta8 Ref: https://github.com/syndbg/goenv/pull/74
- [GH-76] Fix docs values Ref: https://github.com/syndbg/goenv/pull/77
- add 1.12.4 and 1.11.9 Ref: https://github.com/syndbg/goenv/pull/78
- [GH-54] Fix golang releases without patch version not being installed Ref: https://github.com/syndbg/goenv/pull/75
- Prepare 2.0.0beta9 Ref: https://github.com/syndbg/goenv/pull/79
- add 1.12.5 and 1.11.10 Ref: https://github.com/syndbg/goenv/pull/83
- add 1.12.6 and 1.11.11 Ref: https://github.com/syndbg/goenv/pull/84
- add 1.13beta1 Ref: https://github.com/syndbg/goenv/pull/86
- add 1.12.7 and 1.11.12 Ref: https://github.com/syndbg/goenv/pull/88
- add 1.12.8 and 1.11.13 Ref: https://github.com/syndbg/goenv/pull/90
- add 1.12.9 Ref: https://github.com/syndbg/goenv/pull/91
- :tada: add 1.13rc1 Ref: https://github.com/syndbg/goenv/pull/92
- add 1.13 Ref: https://github.com/syndbg/goenv/pull/95
- add 1.13rc2 Ref: https://github.com/syndbg/goenv/pull/94
- `go-build` fails if curl or wget does not exist, but no error message is displayed. Ref: https://github.com/syndbg/goenv/pull/93
- Fixed typo Ref: https://github.com/syndbg/goenv/pull/96
- add 1.13.1 and 1.12.10 Ref: https://github.com/syndbg/goenv/pull/97
- move $GOPATH/bin to end of $PATH Ref: https://github.com/syndbg/goenv/pull/100
- add 1.13.3 and 1.12.12 Ref: https://github.com/syndbg/goenv/pull/102
- add 1.13.2 and 1.12.11 Ref: https://github.com/syndbg/goenv/pull/101
- add 1.13.4 and 1.12.13 Ref: https://github.com/syndbg/goenv/pull/103
- add 1.13.5 and 1.12.14 Ref: https://github.com/syndbg/goenv/pull/104
- add 1.14beta1 Ref: https://github.com/syndbg/goenv/pull/105
- 1.13.6 and 1.12.15 Ref: https://github.com/syndbg/goenv/pull/107
- add 1.13.7 and 1.12.16 Ref: https://github.com/syndbg/goenv/pull/108
- add 1.14rc1 Ref: https://github.com/syndbg/goenv/pull/109
- add 1.13.8 and 1.12.17 Ref: https://github.com/syndbg/goenv/pull/110
- add macos testing Ref: https://github.com/syndbg/goenv/pull/111
- add 1.14.0 Ref: https://github.com/syndbg/goenv/pull/113
- add 1.14.1 and 1.13.9 Ref: https://github.com/syndbg/goenv/pull/116
- add 1.14.2 and 1.13.10 Ref: https://github.com/syndbg/goenv/pull/120
- Add ZPlug installation instructions Ref: https://github.com/syndbg/goenv/pull/122
- add 1.14.3 and 1.13.11 Ref: https://github.com/syndbg/goenv/pull/124
- automatic shims Ref: https://github.com/syndbg/goenv/pull/119
- add 1.14.4 and 1.13.12 Ref: https://github.com/syndbg/goenv/pull/125
- add 1.15beta1 Ref: https://github.com/syndbg/goenv/pull/126
- Remove duplicate test Ref: https://github.com/syndbg/goenv/pull/128
- add 1.14.5 and 1.13.13 Ref: https://github.com/syndbg/goenv/pull/127
- add 1.14.6 and 1.13.14 Ref: https://github.com/syndbg/goenv/pull/129
- add 1.15rc2 Ref: https://github.com/syndbg/goenv/pull/134
- add 1.13.15 Ref: https://github.com/syndbg/goenv/pull/137
- add 1.15.0 Ref: https://github.com/syndbg/goenv/pull/138
- add 1.14.7 Ref: https://github.com/syndbg/goenv/pull/135
- Add 1.15.1 and 1.14.8 Ref: https://github.com/syndbg/goenv/pull/139
- support go1.15.2 Ref: https://github.com/syndbg/goenv/pull/142
- add 1.14.9 Ref: https://github.com/syndbg/goenv/pull/141
- Add 1.15.3 and 1.14.10 Ref: https://github.com/syndbg/goenv/pull/149
- search relative to bin_path for plugins Ref: https://github.com/syndbg/goenv/pull/146
- Add 1.15.4 and 1.14.11 Ref: https://github.com/syndbg/goenv/pull/152
- Add 1.15.5 and 1.14.12 Ref: https://github.com/syndbg/goenv/pull/153
- Add 1.15.6 and 1.14.13 Ref: https://github.com/syndbg/goenv/pull/154
- Add 1.16beta1 Ref: https://github.com/syndbg/goenv/pull/155
- Support darwin-arm64 arch on 1.16beta1 Ref: https://github.com/syndbg/goenv/pull/158
- Add Linux arm 64bit Ref: https://github.com/syndbg/goenv/pull/159
- Add 1.15.7 and 1.14.14 Ref: https://github.com/syndbg/goenv/pull/160
- add GOENV_APPEND_GOPATH and GOENV_PREPEND_GOPATH options Ref: https://github.com/syndbg/goenv/pull/148
- clean up init function Ref: https://github.com/syndbg/goenv/pull/161
- Add 1.15.8 and 1.14.15 Ref: https://github.com/syndbg/goenv/pull/162
- Add Go 1.16 Ref: https://github.com/syndbg/goenv/pull/164
- Fix linux arm 64bit version in link Ref: https://github.com/syndbg/goenv/pull/166
- Add 1.15.9 and 1.16.1 Ref: https://github.com/syndbg/goenv/pull/165
- Add go 1.15.10 and 1.16.2 Ref: https://github.com/syndbg/goenv/pull/167
- ISSUE-169: GOENV_GOPATH_PREFIX does not work as expected Ref: https://github.com/syndbg/goenv/pull/170
- Add go1.16.3 (#173) Ref: https://github.com/syndbg/goenv/pull/174
- Add 1.15.11 Ref: https://github.com/syndbg/goenv/pull/175
- Use a POSIX-compatible regex in goenv-version-file-read Ref: https://github.com/syndbg/goenv/pull/176
- Add 1.16.4 and 1.15.12 Ref: https://github.com/syndbg/goenv/pull/178
- Add 1.16.5 and 1.15.13 Ref: https://github.com/syndbg/goenv/pull/181
- Add 1.16.6, 1.15.14 and 1.17beta1 Ref: https://github.com/syndbg/goenv/pull/183
- Add 1.17rc1 Ref: https://github.com/syndbg/goenv/pull/185
- Show progress bar during download tarball Ref: https://github.com/syndbg/goenv/pull/187
- Add 1.15.15, 1.16.7 and 1.17rc2 Ref: https://github.com/syndbg/goenv/pull/189
- Add 1.17 Ref: https://github.com/syndbg/goenv/pull/193
- Add 1.17.1 and 1.16.8 Ref: https://github.com/syndbg/goenv/pull/195
- Add 1.17.2 and 1.16.9 Ref: https://github.com/syndbg/goenv/pull/196
- Use correct checksum for Go Darwin arm 1.17.2 Ref: https://github.com/syndbg/goenv/pull/197
- Add 1.17.3 and 1.16.10 Ref: https://github.com/syndbg/goenv/pull/199
- test_assert_helper: use type -aP instead of which -a Ref: https://github.com/syndbg/goenv/pull/201
- Add 1.17.4, 1.17.5, 1.16.11 and 1.16.12 Ref: https://github.com/syndbg/goenv/pull/204
- Add 1.18beta1 Ref: https://github.com/syndbg/goenv/pull/208
- Add 1.17.6 and 1.16.13 Ref: https://github.com/syndbg/goenv/pull/211
- Update realpath extension source with upstream changes Ref: https://github.com/syndbg/goenv/pull/206
- Add 1.18beta2 Ref: https://github.com/syndbg/goenv/pull/212
- Add 1.16.14 & 1.17.7 Ref: https://github.com/syndbg/goenv/pull/213
- Adds support for 1.18rc1 release Ref: https://github.com/syndbg/goenv/pull/214
- add 1.17.8 and 1.16.15 Ref: https://github.com/syndbg/goenv/pull/216
- Support 1.18.0 Ref: https://github.com/syndbg/goenv/pull/217
- support 1.18.1 Ref: https://github.com/syndbg/goenv/pull/219
- Support go 1.17.9 Ref: https://github.com/syndbg/goenv/pull/221
- feat: suport force darwin arch Ref: https://github.com/syndbg/goenv/pull/220
- Go 1.18.2 Ref: https://github.com/syndbg/goenv/pull/224
- Add Go 1.17.10 Ref: https://github.com/syndbg/goenv/pull/225
- Support Go 1.18.3 Ref: https://github.com/syndbg/goenv/pull/228
- Go 1.18.3 Fix tar.gz MacOS checksum Ref: https://github.com/syndbg/goenv/pull/229
- Support go 1.17.11 Ref: https://github.com/syndbg/goenv/pull/231
- Add go1.19beta1 support Ref: https://github.com/syndbg/goenv/pull/232
- Support Go 1.18.4 and 1.17.12 Ref: https://github.com/syndbg/goenv/pull/236
- Remove redundant command prompts Ref: https://github.com/syndbg/goenv/pull/235
- Support Go 1.18.5 and 1.17.13 Ref: https://github.com/syndbg/goenv/pull/239
- Support Go 1.19 Ref: https://github.com/syndbg/goenv/pull/240
- Doc: simplify git cmd for upgrade & checkout version Ref: https://github.com/syndbg/goenv/pull/242
- Support Go 1.19.1 and 1.18.6 Ref: https://github.com/syndbg/goenv/pull/245
- If `.go-version` is missing, fallback on `go.mod` Ref: https://github.com/syndbg/goenv/pull/227
- Support 1.19.2 and 1.18.7 Ref: https://github.com/syndbg/goenv/pull/250
- Install latest patch version if major.minor semantic version provided; Add latest command to install Ref: https://github.com/syndbg/goenv/pull/252

## 2.0.0beta11

### Added

- Add golang installations of 1.12.6 and 1.11.11 ; Ref: https://github.com/syndbg/goenv/pull/84

## 2.0.0beta10

### Added

- Add golang installations of 1.12.5 and 1.11.10 ; Ref: https://github.com/syndbg/goenv/pull/83

## 2.0.0beta9

### Added

- Add golang installations of 1.12.4 and 1.11.9 ; Ref: https://github.com/syndbg/goenv/pull/78

### Fixed

- Golang releases without patch version not being installed ; Ref: https://github.com/syndbg/goenv/pull/75

## 2.0.0beta8

### Added

- Add golang installations of 1.12.2, 1.12.3, 1.11.7 and 1.11.8 ; Ref: https://github.com/syndbg/goenv/pull/73

### Fixed

- Lack of environment variables configuration documentation after https://github.com/syndbg/goenv/pull/70.
  Also fixed lack of Contributing guidelines ; Ref https://github.com/syndbg/goenv/pull/74

## 2.0.0beta7

### Added

- Add golang installations of 1.12.1. and 1.11.6 ; Ref: https://github.com/syndbg/goenv/pull/71

## 2.0.0beta6

### Added

- Add management of env variable `GOROOT` that can be disabled with env var `GOENV_DISABLE_GOROOT=1`,
  when calling `goenv-sh-rehash` (`goenv rehash` when `eval $(goenv init -)` was previously executed).
  It does not attempt to manage when version is `system`.
  ; Ref: https://github.com/syndbg/goenv/pull/70
- Add management of env variable `GOPATH` that can be disabled with env var `GOENV_DISABLE_GOPATH=1`,
  when calling `goenv-sh-rehash` (`goenv rehash` when `eval $(goenv init -)` was previously executed).
  It does not attempt to manage when version is `system`.
  ; Ref: https://github.com/syndbg/goenv/pull/70
- Add configurable managed `GOPATH` prefix for `goenv-sh-rehash`
  (`goenv rehash` when `eval $(goenv init -)` was previously executed).
  Configured via `GOENV_GOPATH_PREFIX=<your prefix>`.
  E.g `GOENV_GOPATH_PREFIX=/tmp`.
  Default managed `GOPATH` is `$HOME/go`.
  ; Ref: https://github.com/syndbg/goenv/pull/70
- Add `--only-manage-paths` option to `goenv-sh-rehash` (`goenv rehash` when `eval $(goenv init -)` was previously executed) to skip calling `goenv-rehash` and update shims.
  Instead it only updates managed `GOPATH` and `GOROOT` env variables.
  It does not attempt to manage when version is `system`.
  ; Ref: https://github.com/syndbg/goenv/pull/70

### Changed

- Changed `goenv`'s bootstrap (`eval $(goenv init -)`) now to call `goenv-sh-rehash --only-manage-paths`.
  This means that it'll export and manage `GOROOT` and `GOPATH` env vars.
  It does not attempt to manage when version is `system`.
  ; Ref: https://github.com/syndbg/goenv/pull/70
- Changed `goenv-exec` now to set `GOPATH` and `GOROOT` environment variables before
  executing specified cmd and args. Can be disable via `GOENV_DISABLE_GOPATH=1` and `GOENV_DISABLE_GOROOT=1`.
  `GOPATH` can be configured with `GOENV_GOPATH_PREFIX`. E.g `GOENV_GOPATH_PREFIX=/tmp/goenv`.
  Default managed `GOPATH` is `$HOME/go`.
  ; Ref: https://github.com/syndbg/goenv/pull/70

## 2.0.0beta5

### Added

- Add installation definitions for Golang 1.12.0.
  ; Ref: https://github.com/syndbg/goenv/pull/68

## 2.0.0beta4

### Added

- Add installation definitions for Golang 1.12rc1.
  ; Ref: https://github.com/syndbg/goenv/pull/66

## 2.0.0beta3

### Added

- Add installation definitions for Golang 1.11.5 and 1.10.8.
  ; Ref: https://github.com/syndbg/goenv/pull/65

## 2.0.0beta2

### Added

- Add installation definitions for Golang 1.12beta2.
  ; Ref: https://github.com/syndbg/goenv/pull/64

## 2.0.0beta1

### Added

- `make test-goenv-go-build` to test the `go-build` plugin.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- For tests, fake Python-based HTTP file server to download definitions.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `make test=<target_test_suite_path>.bats test-goenv{-go-build,}` functionality to execute a single test suite
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv rehash` via `goenv help --usage rehash`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv root` via `goenv help --usage root`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv sh-rehash` via `goenv help --usage sh-rehash`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv version` via `goenv help --usage version`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Summary for `goenv version-file-read` via `goenv help version-file-read`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Summary for `goenv completions` via `goenv help completions`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv version-name` via `goenv help --usage version-name`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Usage instructions for `goenv version-origin` via `goenv help --usage version-origin`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Debugging support via `GOENV_DEBUG=1` for `goenv uninstall`
  ; Ref: https://github.com/syndbg/goenv/pull/62

### Changed

- `goenv shell` now fails and prints more helpful instructions when the former command is run without proper shell setup via `eval $(goenv init -)`.
  ; Ref: https://github.com/syndbg/goenv/pull/56
  https://github.com/syndbg/goenv/pull/63
- Re-enabled, greatly refactored and made the test suite pass again.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Travis CI test suite to run against `xenial` Ubuntu.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Use https://github.com/bats-core/bats-core instead of https://github.com/sstephenson/bats for test suite runner and replace links.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Backfilled the CHANGELOG.md
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv` error message when `GOENV_NATIVE_EXT=1`, but native extension is not present, to quote `realpath` with single quotes. It's now `failed to load 'realpath' builtin`
- `goenv` error message when `GOENV_DIR` (e.g `/home/syndbg/.goenv`), but it's not writable, to quote `$GOENV_DIR` with single quotes. It's now `cannot change working directory to '$GOENV_DIR'`.
- `goenv` error message when unknown command is given (e.g `goenv potato`), to quote `$command` with single quotes. It's now `no such command '$command'`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv` and `goenv help` is called to quote `goenv help <command>` with single quotes. It's now `'goenv help <command>'`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv init` and `goenv init <shell>` are more explicit now that the given shell is unknown. E.g `profile="<unknown shell: <shell>, replace with your profile path>"`
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv init` and `goenv init <shell>` now return exit status 0.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv prefix <version>` error message when not installed version is given, to quote `$version` with single quotes. It's now `goenv: version '${version}' not installed`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv version-name <version>` error message when not installed version is given, to quote `$version` with single quotes. It's now `goenv: version '${version}' is not installed (set by $(goenv-version-origin))`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv versions` error message when `GOENV_NATIVE_EXT=1`, but native extension is not present, to quote `realpath` with single quotes. It's now `goenv: failed to load 'realpath' builtin`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv which <command>` error message when current version (specified by `GOENV_VERSION` env var or `.go-version` file) is not installed, to quote now with single quotes. It's now `goenv: version '$version' is not installed (set by $(goenv-version-origin))`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv which <command>` error message when `$command` is not a binary executable present in current version, but it's found in other versions, to quote `$command` in single quotes. It's now `The '$command' command exists in these Go versions:`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv which <command>` error message when `$command` is not a binary executable present in $PATH, to quote `$command`in single quotes. It's now`goenv: '$GOENV_COMMAND' command not found`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed `go-build` and `goenv install`'s error message when no `curl` or `wget` are present to now quote using single quotes. It's now `error: please install 'curl' or 'wget' and try again`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed mentions of `pyenv` to `goenv` when no `go` executable is found after installation of definition.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `goenv --version` now returns only goenv version. Previous format of `goenv <version>-<num_commits>-<git_sha>`, now just `<version>`. E.g `goenv 1.23.3`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `go-build --version` now returns only go-build version. Previous format of `go-build <version>-<num_commits>-<git_sha>`, now just `<version>`. E.g `go-build 1.23.3`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed `goenv install <version>`'s error message when `version` is not a known installable definition, but other similar ones are found to be quoted with single quotes. It's now `The following versions contain '$DEFINITION' in the name:` and `See all available versions with 'goenv install --list'.`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed `goenv uninstall <version>`'s error message when `version` is not installed to be quoted using single quotes. It's now `goenv: version '$VERSION_NAME' not installed`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed `goenv uninstall <version>`'s to fail regardless whether `--force` or `-f` is used when version is not installed. This also means that `before_uninstall` hooks won't be triggered.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Changed the `README.md` to be easier to navigate and read by extracting "how it works" to HOW_IT_WORKS.md, "advanced config" to ADVANCED_CONFIGURATION.md, "installation" to INSTALL.md, move Homebrew installation instructions from "advanced config" to INSTALL.md.
  ; Ref: https://github.com/syndbg/goenv/pull/62

### Removed

- `goenv versions` does not look for versions in `{GOENV_ROOT}/versions/*/envs/*` anymore. This was legacy from pyenv.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed `--enable-shared` support in `go-build`. This was legacy from pyenv.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed mentions of default golang download mirrors in README.md. This was legacy from pyenv.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed default golang download mirrors in `go-build`. This was legacy from pyenv.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed `make_args` from `go-build`. This was legacy from pyenv.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed installation definition functions `configured_with_package_dir`, `needs_yaml`, `try_go_module`, `verify_go_module` and `use_homebrew_yaml`. This was legacy from pyenv and it's not useful since we're not compiling.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed logic to determine `go` suffix after installation of definition. It's legacy from pyenv. It's always `go`.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed `unset`-ing of `GOHOME` environment variable after installation. It's not used nowadays in Go.
  ; Ref: https://github.com/syndbg/goenv/pull/62
- Removed `GOENV_BOOTSTRAP_VERSION` support in `goenv install`. Legacy from pyenv and not useful in Go.
  ; Ref: https://github.com/syndbg/goenv/pull/62

### Fixed

- Bad table formatting in the README.md
  ; Ref: https://github.com/syndbg/goenv/pull/62
- `make bats` failing when `bats` already exists locally
  ; Ref: https://github.com/syndbg/goenv/pull/62

## 1.23.3

### Added

- Add installation definition for unix and ARM Golang 1.12beta1
  ; Ref: https://github.com/syndbg/goenv/pull/61

## 1.23.2

### Added

- Add installation definition for unix and ARM Golang 1.11.4
  ; Ref: https://github.com/syndbg/goenv/pull/60
- Add installation definition for unix and ARM Golang 1.10.7
  ; Ref: https://github.com/syndbg/goenv/pull/60

## 1.23.1

### Added

- Add installation definition for unix and ARM Golang 1.11.3
  ; Ref: https://github.com/syndbg/goenv/pull/59
- Add installation definition for unix and ARM Golang 1.10.6
  ; Ref: https://github.com/syndbg/goenv/pull/59

## 1.23.0

### Added

- Add installation definition for unix and ARM Golang 1.11.2
  ; Ref: https://github.com/syndbg/goenv/pull/58
- Add installation definition for unix and ARM Golang 1.10.5
  ; Ref: https://github.com/syndbg/goenv/pull/58

## 1.22.0

### Added

- Add installation definition for unix and ARM Golang 1.11.1
  ; Ref: https://github.com/syndbg/goenv/pull/55

## 1.21.0

### Added

- Add installation definition for unix and ARM Golang 1.11.0
  ; Ref: https://github.com/syndbg/goenv/pull/53
- Add installation definition for unix and ARM Golang 1.10.4
  ; Ref: https://github.com/syndbg/goenv/pull/53

## 1.20.0

### Added

- Add installation definition for unix and ARM Golang 1.11rc2
  ; Ref: https://github.com/syndbg/goenv/pull/52

## 1.19.0

### Added

- Add installation definition for unix and ARM Golang 1.11rc1
  ; Ref: https://github.com/syndbg/goenv/pull/49

## 1.18.0

### Added

- Add installation definition for unix and ARM Golang 1.11beta3
  ; Ref: https://github.com/syndbg/goenv/pull/48

## 1.17.0

### Added

- Add ARM installation definition for Golang 1.10.3
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10.2
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10.1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10.0
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10rc2
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10rc1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.10beta2
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.7
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.6
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.5
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.4
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.3
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.2
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.9.0
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.7
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.5
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.4
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.3
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.8.0
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.7.5
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.7.4
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.7.3
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.7.1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.7.0
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.6.4
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.6.3
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.6.2
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.6.1
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add ARM installation definition for Golang 1.6.0
  ; Ref: https://github.com/syndbg/goenv/pull/47
- Add installation support for ARM builds of Golang
  ; Ref: https://github.com/syndbg/goenv/pull/47

## 1.16.0

### Added

- Add installation support for Golang 1.11beta2
  ; Ref: https://github.com/syndbg/goenv/pull/46

## 1.15.0

### Added

- Add installation support for Golang 1.10.3
  ; Ref: https://github.com/syndbg/goenv/pull/45
- Add installation support for Golang 1.9.7
  ; Ref: https://github.com/syndbg/goenv/pull/45

## 1.14.0

### Added

- Installation support for BSD `amd64` architectures as 64bit
  ; Ref: https://github.com/syndbg/goenv/pull/44

### Fixed

- Installation on BSD `i686` architectures failing due to invalid bash code
  ; Ref: https://github.com/syndbg/goenv/pull/44

## 1.13.0

### Added

- Add support for Golang 1.10.2
  ; Ref: https://github.com/syndbg/goenv/pull/43
- Add support for Golang 1.9.6
  ; Ref: https://github.com/syndbg/goenv/pull/43

## 1.12.0

### Added

- Add support for Golang 1.10.1
  ; Ref: https://github.com/syndbg/goenv/pull/42
- Add support for Golang 1.9.5
  ; Ref: https://github.com/syndbg/goenv/pull/42

## 1.11.0

### Added

- Add support for Golang 1.10
  ; Ref: https://github.com/syndbg/goenv/pull/41

## 1.10.0

### Added

- Add support for Golang 1.10rc2
  ; Ref: https://github.com/syndbg/goenv/pull/40
- Add support for Golang 1.9.4
  ; Ref: https://github.com/syndbg/goenv/pull/40
- Add support for Golang 1.8.7
  ; Ref: https://github.com/syndbg/goenv/pull/40

## 1.9.0

### Added

- Add support for Golang 1.11rc1
  ; Ref: https://github.com/syndbg/goenv/pull/39

## 1.8.0

### Added

- Add support for Golang 1.10beta2
  ; Ref: https://github.com/syndbg/goenv/pull/36
- Add support for Golang 1.9.3
  ; Ref: https://github.com/syndbg/goenv/pull/36

## 1.7.0

### Added

- Add support for Golang 1.9.2
  ; Ref: https://github.com/syndbg/goenv/pull/35
- Add support for Golang 1.8.5
  ; Ref: https://github.com/syndbg/goenv/pull/35

## 1.6.0

### Removed

- Remove `GOROOT` environment variable setup for `goenv-init`
  ; Ref: https://github.com/syndbg/goenv/pull/34
- Remove `GOROOT` environment variable setup when `GOENV_VERSION` is `system` for `goenv-exec`
  ; Ref: https://github.com/syndbg/goenv/pull/34

## 1.5.0

### Added

- Add support for Golang 1.9.1
  ; Ref: https://github.com/syndbg/goenv/pull/33

## 1.4.0

### Added

- Add support for Golang 1.6.4
  ; Ref: https://github.com/syndbg/goenv/pull/32

## 1.3.0

### Added

- Add support for Golang 1.9.0
  ; Ref: https://github.com/syndbg/goenv/pull/31

## 1.2.1

### Fixed

- Replace usage of `setenv` with `set -gx` for fish shells
  ; Ref: https://github.com/syndbg/goenv/pull/28

## 1.2.0

### Added

- Add support for Golang 1.8.1
  ; Ref: https://github.com/syndbg/goenv/pull/25
- Add support for Golang 1.8.3
  ; Ref: https://github.com/syndbg/goenv/pull/27

## 1.1.0

### Changed

- Update goenv homebrew installation instructions in README.md, since it's available as a core formula
  ; Ref: https://github.com/syndbg/goenv/pull/19
- Update COMMANDS.md and remove duplicate command examples
  ; Ref: https://github.com/syndbg/goenv/pull/20

### Fixed

- Fix `goenv init` for fish shells
  ; Ref: https://github.com/syndbg/goenv/pull/22
  https://github.com/syndbg/goenv/commit/80fb488d01baef3b4d262e5b4828175c7ed44289

## 1.0.0

### Fixed

- Switch to semver release versioning

## 1.8 (I have no idea why this release exists)

### Added

- Add support for Golang 1.8.0
  ; Ref: https://github.com/syndbg/goenv/pull/16
- Add support for Golang 1.7.5
  ; Ref: https://github.com/syndbg/goenv/pull/14
  https://github.com/syndbg/goenv/pull/15

## v20161215

### Added

- Add support for Golang 1.7.4
  ; Ref: https://github.com/syndbg/goenv/pull/11
- Travis CI support
  ; Ref: https://github.com/syndbg/goenv/pull/10

### Fixed

- Test command `pyenv echo` => `goenv echo`
  ; Ref: https://github.com/syndbg/goenv/pull/9

## v20161028

### Added

- Add support for Golang 1.7.3
  ; Ref: https://github.com/syndbg/goenv/pull/5
- Add support for Golang 1.7.1
  ; Ref: https://github.com/syndbg/goenv/pull/5
- Add support for Golang 1.7.0
  ; Ref: https://github.com/syndbg/goenv/pull/5
- Add support for Golang 1.6.3
  ; Ref: https://github.com/syndbg/goenv/pull/4

### Removed

- Remove some more `pyenv` and `python-build` references in README.md
  ; Ref: https://github.com/syndbg/goenv/pull/7

### Fixed

- Fix bash auto-completion trying to use `pyenv` instead of `goenv`
  ; Ref: https://github.com/syndbg/goenv/pull/6

## v20160814

### Changed

- Updated comparison with other golang environment management projects
  ; Ref: https://github.com/syndbg/goenv/commit/68a5a18d493dc9f6d9ab45f7c4bc4b52a10557e2
- Update homebrew installation instructions
  ; Ref: https://github.com/syndbg/goenv/pull/1

### Fixed

- Installation on Linux `i686` architectures failing due to invalid bash code
  ; Ref: https://github.com/syndbg/goenv/pull/3

- Wrong checksum for 64bit Linux release of Golang 1.2.2
  ; Ref: https://github.com/syndbg/goenv/pull/2

## v20160424

### Added

- Add support for unix Go 1.6.2
  ; Ref: https://github.com/syndbg/goenv/commit/fc211b1b78370f7e679872c6cebbffa92dd0017f
- Add support for unix Go versions 1.6.1
  ; Ref: https://github.com/syndbg/goenv/commit/8f2171d4014bff3ba15faf7d784965a8a7590205
  https://github.com/syndbg/goenv/commit/61689d52e9d23a46ab12c5195ac32452dea3ef75
- Add support for unix Go versions 1.6.0
  ; Ref: https://github.com/syndbg/goenv/commit/8f2171d4014bff3ba15faf7d784965a8a7590205
  https://github.com/syndbg/goenv/commit/61689d52e9d23a46ab12c5195ac32452dea3ef75
- Add support for unix Go 1.5.4
  ; Ref: https://github.com/syndbg/goenv/commit/15e9863b33aa23f6794261fbdf247f9760cc43e1
- Add support for unix Go 1.5.3
  ; Ref: https://github.com/syndbg/goenv/commit/a099796aafa70da932e9cd8aa0a6a99f64f49904
- Add support for unix Go 1.5.2
  ; Ref: https://github.com/syndbg/goenv/commit/01ec11c3fd4058eb55d57f61d494233e37c233a6
- Add support for unix Go 1.5.1
  ; Ref: https://github.com/syndbg/goenv/commit/0745eb7243afc9a41997acc0080ca4e292ecdbe4
- Add support for unix Go 1.5.0
  ; Ref: https://github.com/syndbg/goenv/commit/f527101285c32b1abd3726d535ccf2f87d4a2447
- Add support for unix Go 1.4.3
  ; Ref: https://github.com/syndbg/goenv/commit/fa5856a476a0d32674e26cd9e7283806ef9f78b8
- Add support for unix Go 1.4.2
  ; Ref: https://github.com/syndbg/goenv/commit/87a123c682e00bb024a0a5ddf2c0a2d6e9fe18a3
- Add support for unix Go 1.4.1
  ; Ref: https://github.com/syndbg/goenv/commit/9bb9168555d38376f5251c01f51f8c84ea3cf4e4
- Add support for unix Go 1.4.0
  ; Ref: https://github.com/syndbg/goenv/commit/02e3aea3f1222e21b67d41696cc0eb65485e106f
- Add support for unix Go 1.3.3
  ; Ref: https://github.com/syndbg/goenv/commit/2543028c389311a66db26d39a4e674eae326549d
- Add support for unix Go 1.3.2
  ; Ref: https://github.com/syndbg/goenv/commit/bc48eda1a77d89b844fa4d920e68a2207d85c72c
- Add support for unix Go 1.3.1
  ; Ref: https://github.com/syndbg/goenv/commit/63537895c87d4db2d76bb1c99bb0a4c3b0e44442
- Add support for unix Go 1.3.0
  ; Ref: https://github.com/syndbg/goenv/commit/b8fc6e7013028daf78080fda86a1afa2fa3ef590
- Add support support for unix Go 1.2.2
  ; Ref: https://github.com/syndbg/goenv/commit/fe1db9bfc8f2dc0fb3f3e0eb0d02192fd596b046
- Installation definition functions for Darwin 10.6 32/64bit, Darwin 10.8 32/64bit
  ; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
- Add SHA1 checksum verification support
  ; Ref: https://github.com/syndbg/goenv/commit/d38c5875f7aaa559b1dfc9976f5f52309953a023

### Changed

- Update documentation to fix usage where `pyenv` is specified instead of `goenv`.
  ; Ref: https://github.com/syndbg/goenv/commit/807520548ae6e2727a87ab04d2993522ae0e76d0

### Removed

- Obsolete compilation functionality for gcc and similar
  ; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
- Obsolete MacOS build functionality
  ; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
- Obsolete compilation tests
  ; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418

## v20160417

### Added

- `goenv` basic functionality cloned from https://github.com/yyuu/pyenv
