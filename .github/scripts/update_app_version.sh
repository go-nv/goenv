#!/bin/bash -e

if [[ $DEBUG == "true" ]]; then
    echo "DEBUG is enabled"
    set -x # Print commands and their arguments as they are executed
fi

APP_VERSION_FILE="APP_VERSION"

# This script is used to update the version of the app.

# Get the latest draft version from the GitHub Releases
# and take into accoutn that the most recent version is the latest version,
# while the second most recent version is the second entry.
# ❯ gh release list -L 5
# TITLE   TYPE    TAG NAME  PUBLISHED
# 2.2.27  Latest  2.2.27    about 5 days ago
# 2.2.28  Draft   2.2.28    about 1 month ago
# 2.2.26          2.2.26    about 1 month ago
# 2.2.25          2.2.25    about 2 months ago
# 2.2.24          2.2.24    about 2 months ago
LATEST_DRAFT_VERSION=$(gh release list -L 20 | awk -F '\t' '{if ($2 == "Draft" && match($3, "^[0-9]+\\.[0-9]+\\.[0-9]+")) {print $3; exit}}')
LATEST_VERSION=$(gh release view --json tagName -q .tagName)

if [[ $LATEST_DRAFT_VERSION == $LATEST_VERSION ]]; then
    echo "latest draft version ($LATEST_DRAFT_VERSION) matches latest version ($LATEST_VERSION)"
    exit 0
elif [[ $LATEST_DRAFT_VERSION == $(cat $APP_VERSION_FILE) ]]; then
    echo "latest draft version ($LATEST_DRAFT_VERSION) matches current version in $APP_VERSION_FILE"
    exit 0
fi

if [[ -n $GITHUB_ACTOR ]]; then
    git config --global user.name "${GITHUB_ACTOR}"
    git config --global user.email "${GITHUB_ACTOR}@users.noreply.github.com"
fi

BRANCH_NAME_PREFIX="update-app-version"

BRANCH_NAME="$BRANCH_NAME_PREFIX-$LATEST_DRAFT_VERSION"

EXISTS=$(git branch -r -l 'origin*' | sed -E -e 's/^[^\/]+\///g' -e 's/HEAD.+//' | grep "$BRANCH_NAME" || echo "false")

if [[ -n $EXISTS ]] && [[ $EXISTS != "false" ]]; then
    echo "A PR already exists on branch $BRANCH_NAME for App Version update: $LATET_DRAFT_VERSION"
    exit 0
fi

echo "checking out new Git branch $BRANCH_NAME..."

git switch -c $BRANCH_NAME master

printf " done\n"

echo "Updating $APP_VERSION_FILE..."

echo $LATEST_DRAFT_VERSION >$APP_VERSION_FILE

echo "Committing changes..."

COMMIT_MSG="Update $APP_VERSION_FILE to $LATEST_DRAFT_VERSION"

git add $APP_VERSION_FILE

git commit -m "$COMMIT_MSG"

echo "Pushing to origin..."

git push -u origin $BRANCH_NAME

echo "Creating Pull Request..."

printf "This updates the App Version to $LATEST_DRAFT_VERSION" | gh pr create -R go-nv/goenv -B master \
    -t "$COMMIT_MSG" \
    -F -

echo 'All done!'
