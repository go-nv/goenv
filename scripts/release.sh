#!/bin/bash

set -o errexit

cd $(brew --repository homebrew/core)

git remote add
git remote set-url origin https://github.com/homebrew/homebrew-core.git
git remote add $GITHUB_ACTOR https://github.com/$GITHUB_ACTOR/homebrew-core.git
brew update-reset

env | grep HOMEBREW || true

brew tap || true

# if [ ! "published" = "$(jq -r '.action' "$GITHUB_EVENT_PATH")" ]; then
#     echo "Release was not published, skipping."
#     exit 78
# fi

if [ "true" = "$(jq -r '.release.draft' "$GITHUB_EVENT_PATH")" ]; then
    echo "This is a draft release... skipping"
    exit 78
fi

if [ "true" = "$(jq -r '.release.prerelease' "$GITHUB_EVENT_PATH")" ]; then
    echo "Homebrew does not accept pre-releases or other unstable releases in Homebrew-core"
    exit 78
fi

tag="$(jq -r '.release.tag_name' "$GITHUB_EVENT_PATH" || true)"

if [ -z "$tag" ]; then
    echo "Release tag was empty $tag"
    exit 78
fi

if [ -z "${BREW_PR_AUTHOR}" ]; then
    BREW_PR_AUTHOR="$(jq -r '.release.author.login' "$GITHUB_EVENT_PATH")"
fi
if [ -z "${BREW_PR_AUTHOR_EMAIL}" ]; then
    BREW_PR_AUTHOR_EMAIL="$(jq -r '.release.author.html_url' "$GITHUB_EVENT_PATH")"
fi

git config --global user.name "${BREW_PR_AUTHOR}"
git config --global user.email "${BREW_PR_AUTHOR_EMAIL}"

if [ -z "${BREW_PR_FORMULA}" ]; then
    if [ -z "$1" ]; then
        BREW_PR_FORMULA="$(jq -r '.repository.name' "$GITHUB_EVENT_PATH")"
    else
        BREW_PR_FORMULA="$1"
    fi
fi

echo "Auditing formula to ensure there are no formatting surprises..."
brew audit --strict "${BREW_PR_FORMULA}"

BREW_PR_MSG="Release goenv $GITHUB_REF_NAME\n\nThis releases goenv $GITHUB_REF_NAME"

# Check for tag and SHA versioning and do PR bump
if brew cat "${BREW_PR_FORMULA}" | awk '/url ".*",/,/:revision => ".*"/' | grep ':tag' >/dev/null 2>&1; then
    echo "Formula ${BREW_PR_FORMULA} appears to use tag and SHA versioning."
    brew bump-formula-pr --strict --tag="${GITHUB_REF#/refs/tags/}" --revision="${GITHUB_SHA}" --dry-run --message="${BREW_PR_MSG}" "${BREW_PR_FORMULA}"
    brew bump-formula-pr --strict --tag="${GITHUB_REF#/refs/tags/}" --revision="${GITHUB_SHA}" --verbose --message="${BREW_PR_MSG}" --no-browse "${BREW_PR_FORMULA}"
    exit $?
fi

# Not tag and SHA versioning, proceed with URL and SHA256 versioning
old_url="$(brew cat "${BREW_PR_FORMULA}" | awk '/url ".*"/,/sha256 ".*"/' | sed -n '1 s/^.*"\(.*\)"$/\1/p')"
if [ -z "$old_url" ]; then
    echo "Error: couldn't parse old url out of url & sha256 formula: ${BREW_PR_FORMULA}"
    exit 99
fi

if [ "https://github.com/${GITHUB_REPOSITORY}/archive" = "${old_url%/*}" ]; then
    echo "Formula ${BREW_PR_FORMULA} appears to use default archives created by github"
    if echo "$old_url" | grep '.tar.gz$' >/dev/null 2>&1; then
        suffix=.tar.gz
    elif echo "$old_url" | grep '.zip$' >/dev/null 2>&1; then
        suffix=.zip
    else
        echo "Error: Could not determine suffix from old download url:\
	$old_url"
        exit 99
    fi
    new_url="${old_url%/*}/${tag}${suffix}"
    wget "$new_url"
    if [ ! -f "${tag}${suffix}" ]; then
        echo "Failed to get ${tag}${suffix} from ${new_url}" >&2
        exit 99
    fi
    sha256="$(sha256sum "${tag}${suffix}" | sed "s/ *${tag}${suffix} *$//")"
    echo "WARNING: Computing sha256 checksum on the fly from ${new_url}" >&2
    echo "CHECKSUM: ${sha256} ${tag}${suffix}" >&2
    brew bump-formula-pr --strict --url="${new_url}" --sha256="${sha256}" --message="${BREW_PR_MSG}" --dry-run "${BREW_PR_FORMULA}"
    brew bump-formula-pr --strict --url="${new_url}" --sha256="${sha256}" --message="${BREW_PR_MSG}" --verbose --no-browse "${BREW_PR_FORMULA}"
    exit $?
fi

num_release_assets="$(jq -r '.release.assets | length' "$GITHUB_EVENT_PATH" || true)"
if [ ! "$num_release_assets" -gt 0 ]; then
    echo "No release assets, cannot proceed" >&2
    exit 99
else
    old_version="$(brew info --json "${BREW_PR_FORMULA}" | jq -r '.[0].versions.stable')"
    new_url="$(echo "${old_url}" | sed "s/${old_version}/${tag}/g")"
    old_file=${old_url##*/}
    new_file="$(echo "${old_file}" | sed "s/${old_version}/${tag}/g")"
    echo "$num_release_assets release assets found, getting names and urls..."
    release_assets_urls="$(jq -r '.assets.[].browser_download_url' "$GITHUB_EVENT_PATH")"
    release_assets_names="$(jq -r '.assets.[].name' "$GITHUB_EVENT_PATH")"
    for u in ${release_assets_urls}; do
        wget "$u"
        if [ "$u" = "$new_url" ]; then
            echo "New release found in release assets"
            new_release_present=true
        fi
    done
    if [ ! "$new_release_present" = "true" ]; then
        echo "There does not appear to be a file at ${new_url}" >&2
        exit 99
    fi
    for f in ${release_assets_names}; do
        case "$f" in
        *sha256* | *SHA256*)
            if grep "${new_file}" "$f"; then
                if sha256sum -c "$f" | grep "${new_file}.* OK"; then
                    echo "Checksum for ${new_file} verified"
                    checksum_verified=true
                else
                    echo "Checksums did not appear to match" >&2
                    checksum_fault=true
                fi
                grep "${new_file}" "$f"
                sha256sum "${new_file}"
                if [ "$checksum_fault" = "true" ]; then
                    exit 99
                fi
            fi
            shasum_files="${shasum_files:-} $f"
            ;;
        *md5* | *MD5*)
            md5sum -c "$f"
            ;;
        *.asc)
            gpg --verify "$f"
            ;;
        esac
    done
    sha256="$(sha256sum "${new_file}" | sed "s/ *${new_file} *//")"
    if [ ! "$checksum_verified" = "true" ]; then
        echo "WARNING: Computing sha256 checksum on the fly from ${new_url}" >&2
        echo "CHECKSUM: ${sha256} ${new_file}" >&2
    fi
    brew bump-formula-pr --strict --url="${new_url}" --sha256="${sha256}" --message="${BREW_PR_MSG}" --dry-run "${BREW_PR_FORMULA}"
    brew bump-formula-pr --strict --url="${new_url}" --sha256="${sha256}" --message="${BREW_PR_MSG}" --verbose --no-browse "${BREW_PR_FORMULA}"
    exit $?
fi
