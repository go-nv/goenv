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

## Unreleased (master)

## 1.23.3

### Added

* Add installation definition for unix and ARM Golang 1.12beta1
; Ref: https://github.com/syndbg/goenv/pull/61

## 1.23.2

### Added

* Add installation definition for unix and ARM Golang 1.11.4
; Ref: https://github.com/syndbg/goenv/pull/60
* Add installation definition for unix and ARM Golang 1.10.7
; Ref: https://github.com/syndbg/goenv/pull/60

## 1.23.1

### Added

* Add installation definition for unix and ARM Golang 1.11.3
; Ref: https://github.com/syndbg/goenv/pull/59
* Add installation definition for unix and ARM Golang 1.10.6
; Ref: https://github.com/syndbg/goenv/pull/59

## 1.23.0

### Added

* Add installation definition for unix and ARM Golang 1.11.2
; Ref: https://github.com/syndbg/goenv/pull/58
* Add installation definition for unix and ARM Golang 1.10.5
; Ref: https://github.com/syndbg/goenv/pull/58

## 1.22.0

### Added

* Add installation definition for unix and ARM Golang 1.11.1
; Ref: https://github.com/syndbg/goenv/pull/55

## 1.21.0

### Added

* Add installation definition for unix and ARM Golang 1.11.0
; Ref: https://github.com/syndbg/goenv/pull/53
* Add installation definition for unix and ARM Golang 1.10.4
; Ref: https://github.com/syndbg/goenv/pull/53


## 1.20.0

### Added

* Add installation definition for unix and ARM Golang 1.11rc2
; Ref: https://github.com/syndbg/goenv/pull/52

## 1.19.0

### Added

* Add installation definition for unix and ARM Golang 1.11rc1
; Ref: https://github.com/syndbg/goenv/pull/49

## 1.18.0

### Added

* Add installation definition for unix and ARM Golang 1.11beta3
; Ref: https://github.com/syndbg/goenv/pull/48

## 1.17.0

### Added

* Add ARM installation definition for Golang 1.10.3
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10.2
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10.1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10.0
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10rc2
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10rc1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.10beta2
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.7
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.6
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.5
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.4
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.3
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.2
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.9.0
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.7
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.5
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.4
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.3
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.8.0
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.7.5
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.7.4
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.7.3
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.7.1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.7.0
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.6.4
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.6.3
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.6.2
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.6.1
; Ref: https://github.com/syndbg/goenv/pull/47
* Add ARM installation definition for Golang 1.6.0
; Ref: https://github.com/syndbg/goenv/pull/47
* Add installation support for ARM builds of Golang
; Ref: https://github.com/syndbg/goenv/pull/47

## 1.16.0

### Added

* Add installation support for Golang 1.11beta2
; Ref: https://github.com/syndbg/goenv/pull/46

## 1.15.0

### Added

* Add installation support for Golang 1.10.3
; Ref: https://github.com/syndbg/goenv/pull/45
* Add installation support for Golang 1.9.7
; Ref: https://github.com/syndbg/goenv/pull/45

## 1.14.0

### Added

* Installation support for BSD `amd64` architectures as 64bit
; Ref: https://github.com/syndbg/goenv/pull/44

### Fixed

* Installation on BSD `i686` architectures failing due to invalid bash code
; Ref: https://github.com/syndbg/goenv/pull/44

## 1.13.0

### Added

* Add support for Golang 1.10.2
; Ref: https://github.com/syndbg/goenv/pull/43
* Add support for Golang 1.9.6
; Ref: https://github.com/syndbg/goenv/pull/43


## 1.12.0

### Added

* Add support for Golang 1.10.1
; Ref: https://github.com/syndbg/goenv/pull/42
* Add support for Golang 1.9.5
; Ref: https://github.com/syndbg/goenv/pull/42

## 1.11.0

### Added

* Add support for Golang 1.10
; Ref: https://github.com/syndbg/goenv/pull/41

## 1.10.0

### Added

* Add support for Golang 1.10rc2
; Ref: https://github.com/syndbg/goenv/pull/40
* Add support for Golang 1.9.4
; Ref: https://github.com/syndbg/goenv/pull/40
* Add support for Golang 1.8.7
; Ref: https://github.com/syndbg/goenv/pull/40

## 1.9.0

### Added

* Add support for Golang 1.11rc1
; Ref: https://github.com/syndbg/goenv/pull/39

## 1.8.0

### Added

* Add support for Golang 1.10beta2
; Ref: https://github.com/syndbg/goenv/pull/36
* Add support for Golang 1.9.3
; Ref: https://github.com/syndbg/goenv/pull/36

## 1.7.0

### Added

* Add support for Golang 1.9.2
; Ref: https://github.com/syndbg/goenv/pull/35
* Add support for Golang 1.8.5
; Ref: https://github.com/syndbg/goenv/pull/35

## 1.6.0

### Removed

* Remove `GOROOT` environment variable setup for `goenv-init`
; Ref: https://github.com/syndbg/goenv/pull/34
* Remove `GOROOT` environment variable setup when `GOENV_VERSION` is `system` for `goenv-exec`
; Ref: https://github.com/syndbg/goenv/pull/34

## 1.5.0

### Added

* Add support for Golang 1.9.1
; Ref: https://github.com/syndbg/goenv/pull/33

## 1.4.0

### Added

* Add support for Golang 1.6.4
; Ref: https://github.com/syndbg/goenv/pull/32

## 1.3.0

### Added

* Add support for Golang 1.9.0
; Ref: https://github.com/syndbg/goenv/pull/31

## 1.2.1

### Fixed

* Replace usage of `setenv` with `set -gx` for fish shells
; Ref: https://github.com/syndbg/goenv/pull/28

## 1.2.0

### Added

* Add support for Golang 1.8.1
; Ref: https://github.com/syndbg/goenv/pull/25
* Add support for Golang 1.8.3
; Ref: https://github.com/syndbg/goenv/pull/27

## 1.1.0

### Changed

* Update goenv homebrew installation instructions in README.md, since it's available as a core formula
; Ref: https://github.com/syndbg/goenv/pull/19
* Update COMMANDS.md and remove duplicate command examples
; Ref: https://github.com/syndbg/goenv/pull/20

### Fixed

* Fix `goenv init` for fish shells
; Ref: https://github.com/syndbg/goenv/pull/22
https://github.com/syndbg/goenv/commit/80fb488d01baef3b4d262e5b4828175c7ed44289

## 1.0.0

### Fixed

* Switch to semver release versioning

## 1.8 (I have no idea why this release exists)

### Added

* Add support for Golang 1.8.0
; Ref: https://github.com/syndbg/goenv/pull/16
* Add support for Golang 1.7.5
; Ref: https://github.com/syndbg/goenv/pull/14
https://github.com/syndbg/goenv/pull/15

## v20161215

### Added

* Add support for Golang 1.7.4
; Ref: https://github.com/syndbg/goenv/pull/11
* Travis CI support
; Ref: https://github.com/syndbg/goenv/pull/10

### Fixed

* Test command `pyenv echo` => `goenv echo`
; Ref: https://github.com/syndbg/goenv/pull/9

## v20161028

### Added

* Add support for Golang 1.7.3
; Ref: https://github.com/syndbg/goenv/pull/5
* Add support for Golang 1.7.1
; Ref: https://github.com/syndbg/goenv/pull/5
* Add support for Golang 1.7.0
; Ref: https://github.com/syndbg/goenv/pull/5
* Add support for Golang 1.6.3
; Ref: https://github.com/syndbg/goenv/pull/4

### Removed

* Remove some more `pyenv` and `python-build` references in README.md
; Ref: https://github.com/syndbg/goenv/pull/7

### Fixed

* Fix bash auto-completion trying to use `pyenv` instead of `goenv`
; Ref: https://github.com/syndbg/goenv/pull/6

## v20160814

### Changed

* Updated comparison with other golang environment management projects 
; Ref: https://github.com/syndbg/goenv/commit/68a5a18d493dc9f6d9ab45f7c4bc4b52a10557e2 
* Update homebrew installation instructions
; Ref: https://github.com/syndbg/goenv/pull/1

### Fixed

* Installation on Linux `i686` architectures failing due to invalid bash code
; Ref: https://github.com/syndbg/goenv/pull/3

* Wrong checksum for 64bit Linux release of Golang 1.2.2
; Ref: https://github.com/syndbg/goenv/pull/2

## v20160424

### Added

* Add support for unix Go 1.6.2
; Ref: https://github.com/syndbg/goenv/commit/fc211b1b78370f7e679872c6cebbffa92dd0017f
* Add support for unix Go versions 1.6.1
; Ref: https://github.com/syndbg/goenv/commit/8f2171d4014bff3ba15faf7d784965a8a7590205
https://github.com/syndbg/goenv/commit/61689d52e9d23a46ab12c5195ac32452dea3ef75
* Add support for unix Go versions 1.6.0
; Ref: https://github.com/syndbg/goenv/commit/8f2171d4014bff3ba15faf7d784965a8a7590205
https://github.com/syndbg/goenv/commit/61689d52e9d23a46ab12c5195ac32452dea3ef75
* Add support for unix Go 1.5.4
; Ref: https://github.com/syndbg/goenv/commit/15e9863b33aa23f6794261fbdf247f9760cc43e1
* Add support for unix Go 1.5.3
; Ref: https://github.com/syndbg/goenv/commit/a099796aafa70da932e9cd8aa0a6a99f64f49904
* Add support for unix Go 1.5.2
; Ref: https://github.com/syndbg/goenv/commit/01ec11c3fd4058eb55d57f61d494233e37c233a6
* Add support for unix Go 1.5.1
; Ref: https://github.com/syndbg/goenv/commit/0745eb7243afc9a41997acc0080ca4e292ecdbe4
* Add support for unix Go 1.5.0
; Ref: https://github.com/syndbg/goenv/commit/f527101285c32b1abd3726d535ccf2f87d4a2447
* Add support for unix Go 1.4.3
; Ref: https://github.com/syndbg/goenv/commit/fa5856a476a0d32674e26cd9e7283806ef9f78b8
* Add support for unix Go 1.4.2
; Ref: https://github.com/syndbg/goenv/commit/87a123c682e00bb024a0a5ddf2c0a2d6e9fe18a3
* Add support for unix Go 1.4.1
; Ref: https://github.com/syndbg/goenv/commit/9bb9168555d38376f5251c01f51f8c84ea3cf4e4
* Add support for unix Go 1.4.0
; Ref: https://github.com/syndbg/goenv/commit/02e3aea3f1222e21b67d41696cc0eb65485e106f
* Add support for unix Go 1.3.3
; Ref: https://github.com/syndbg/goenv/commit/2543028c389311a66db26d39a4e674eae326549d
* Add support for unix Go 1.3.2
; Ref: https://github.com/syndbg/goenv/commit/bc48eda1a77d89b844fa4d920e68a2207d85c72c
* Add support for unix Go 1.3.1
; Ref: https://github.com/syndbg/goenv/commit/63537895c87d4db2d76bb1c99bb0a4c3b0e44442
* Add support for unix Go 1.3.0
; Ref: https://github.com/syndbg/goenv/commit/b8fc6e7013028daf78080fda86a1afa2fa3ef590
* Add support support for unix Go 1.2.2 
; Ref: https://github.com/syndbg/goenv/commit/fe1db9bfc8f2dc0fb3f3e0eb0d02192fd596b046
* Installation definition functions for Darwin 10.6 32/64bit, Darwin 10.8 32/64bit
; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
* Add SHA1 checksum verification support 
; Ref: https://github.com/syndbg/goenv/commit/d38c5875f7aaa559b1dfc9976f5f52309953a023

### Changed

* Update documentation to fix usage where `pyenv` is specified instead of `goenv`. 
; Ref: https://github.com/syndbg/goenv/commit/807520548ae6e2727a87ab04d2993522ae0e76d0

### Removed

* Obsolete compilation functionality for gcc and similar
; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
* Obsolete MacOS build functionality 
; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418
* Obsolete compilation tests 
; Ref: https://github.com/syndbg/goenv/commit/4e810f8afd5086ef4fa618a0800d50dec54e6418

## v20160417

### Added

* `goenv` basic functionality cloned from https://github.com/yyuu/pyenv
