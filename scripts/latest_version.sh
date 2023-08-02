#!/bin/bash

GIT_ROOT=$(git rev-parse --show-toplevel)
GO_BUILD_DIR=$GIT_ROOT/plugins/go-build
GO_DEFINITIONS_DIR=$GO_BUILD_DIR/share/go-build

OLD_PATH=$PATH

PATH="$PATH:$GIT_ROOT/plugins/go-build/bin"

# Load shared library functions
eval "$(go-build --lib)"

definitions() {
    local query="$1"
    go-build --definitions | $(type -p ggrep grep | head -1) -F "$query" || true
}

latest_includes_unstable_version() {
    definitions | grep -ioE "^$1\\.?([0-9]+|beta[0-9]+|rc[0-9]+)?$" | tail -1
}

capitalize() {
    printf '%s' "$1" | head -c 1 | tr [:lower:] [:upper:]
    printf '%s' "$1" | tail -c '+2'
}

BRANCH_NAME_PREFIX="goenv_bot_add"

while read LATEST_GO_DEFINITION; do
    printf "fetching latest go version..."

    LATEST_GO_VERSION=$(echo $LATEST_GO_DEFINITION | jq '.version' | sed -e 's/go//' -e 's/\"//g')

    printf " %s\n" $LATEST_GO_VERSION

    printf "fetching latest goenv definition..."

    LATEST_GOENV_DEFINITION=$(latest_includes_unstable_version $(echo $LATEST_GO_VERSION | sed -r 's/^([0-9]+\.[0-9]+)(\.[0-9]+)?$/\1/' | sed s/\\./\\\\./))

    printf " %s\n" $LATEST_GOENV_DEFINITION

    if [[ $LATEST_GO_VERSION == $LATEST_GOENV_DEFINITION ]]; then
        echo "latest goenv definition ($LATEST_GOENV_DEFINITION) matches latest Go version ($LATEST_GO_VERSION)"
        continue
    fi

    printf "Creating new definitions for $LATEST_GO_VERSION..."

    BRANCH_NAME="${BRANCH_NAME_PREFIX}_${LATEST_GO_VERSION}"

    GO_BUILD_DEFINITION_FILE=$GO_DEFINITIONS_DIR/$LATEST_GO_VERSION

    LATEST_FILE_LIST=$(echo $LATEST_GO_DEFINITION | jq -c '.files[] | select(.os == "darwin" or .os == "linux" or .os == "freebsd") | select(.arch == "386" or .arch == "amd64" or .arch == "armv6l" or .arch == "arm64") | select(.kind == "archive")')

    echo $LATEST_FILE_LIST | jq -c '.' | while read FILE_DATA; do
        FILENAME=$(echo $FILE_DATA | jq -r '.filename')
        SHA256=$(echo $FILE_DATA | jq -r '.sha256')
        OS=$(echo $FILE_DATA | jq -r '.os')
        OS_TITLE_CASE=$(capitalize $OS)
        ARCH=$(echo $FILE_DATA | jq -r '.arch')

        case $OS in
        darwin) ;;
        linux) ;;
        freebsd)
            OS="bsd"
            ;;
        *)
            echo "$OS is not valid"
            continue
            ;;
        esac

        case $ARCH in
        386)
            ARCH="32bit"
            ;;
        amd64)
            ARCH="64bit"
            ;;
        arm64)
            if [[ $OS == "linux" ]]; then
                ARCH="arm_64bit"
            else
                ARCH="arm"
            fi
            ;;
        armv6l)
            ARCH="arm"
            ;;
        *)
            echo "$ARCH is not valid"
            continue
            ;;
        esac

        # Version file pattern
        # install_{OS}_{ARCH} "Go {OS_TITLE_CASE} {ARCH} {VERSION}" "{FILENAME}#{FILE_SHA256}"
        printf 'install_%s_%s "Go %s %s %s" "%s#%s"\n\n' $OS $ARCH $OS_TITLE_CASE "$(echo $ARCH | sed -e 's/_/ /')" $LATEST_GO_VERSION $FILENAME $SHA256 >>$GO_BUILD_DEFINITION_FILE
    done

    LATEST_GO_VERSIONS+=($LATEST_GO_VERSION)

done <<<"$(curl -s 'https://go.dev/dl/?mode=json' | jq -c '.[]')"

if [[ ${#LATEST_GO_VERSIONS[@]} == 0 ]]; then
    echo "There are no versions to update. Exiting..."
    exit 0
fi

git config --global user.name "${GITHUB_ACTOR}"
git config --global user.email "${GITHUB_ACTOR}@users.noreply.github.com"

EXISTS=$(git branch -r -l 'origin*' | sed -E -e 's/^[^\/]+\///g' -e 's/HEAD.+//' | grep "$BRANCH_NAME")

if [[ -n $EXISTS ]]; then
    echo "A PR already exists on branch $BRANCH_NAME for the latest Go version(s) (${LATEST_GO_VERSIONS[@]})"
    exit 0
fi

echo "checking out new Git branch $BRANCH_NAME..."

git switch -c $BRANCH_NAME master

printf " done\n"

echo "Committing changes..."

COMMIT_MSG="[goenv-bot]: Add ${LATEST_GO_VERSIONS[@]} definition to goenv"

git add $GO_DEFINITIONS_DIR

git commit -m "$COMMIT_MSG"

echo "Pushing to origin..."

git push -u origin $BRANCH_NAME

echo "Creating Pull Request..."

printf "This adds the Go Definitions for version ${LATEST_GO_VERSIONS[@]}.\n\nCreated by Github action automation\n" | gh pr create -R go-nv/goenv -B master \
    -t "$COMMIT_MSG" \
    -F -

echo 'All done!'

# All done, reset PATH
PATH=$OLD_PATH
