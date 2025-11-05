# Release process

Releases are done **automatically** via GitHub actions and Release Drafter.
 
## Rules

1. Releases are only created from `master`.
1. `master` is meant to be stable, so before tagging and create a new release, make sure that the CI checks pass.
1. Releases are GitHub releases.
1. Releases are following *semantic versioning*.
1. Releases are to be named in pattern of `X.Y.Z`. The produced binary artifacts contain the `X.Y.Z` in their names.
1. Changelog must up-to-date with what's going to be released. Check [CHANGELOG](./CHANGELOG.md).
1. APP_VERSION file **must be updated** to match the draft version to be released (Please run the Pre Release action to create a PR update).

## Flow

1. Create a new GitHub release using https://github.com/go-nv/goenv
1. `Tag Version` and `Release Title` are going to be in pattern of `vX.Y.Z`.
1. `Describe this release` (content) is going to link the appropriate [CHANGELOG](./CHANGELOG.md) entry.
