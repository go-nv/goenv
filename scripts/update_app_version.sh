#!/bin/bash

# This script is used to update the version of the app.

LATEST_DRAFT_VERSION=$(gh release list -L 1 | awk -F '\t' '{if (match($3, "^[0-9]+\.[0-9]+\.[0-9]+")) print $3}')
LATEST_VERSION=$(gh release view --json tagName -q .tagName)

if [[ $LATEST_DRAFT_VERSION == $LATEST_VERSION ]]; then
    echo "latest draft version ($LATEST_DRAFT_VERSION) matches latest version ($LATEST_VERSION)"
    exit 0
fi

git config --global user.name "${GITHUB_ACTOR}"
git config --global user.email "${GITHUB_ACTOR}@users.noreply.github.com"

BRANCH_NAME_PREFIX="update-app-version"

BRANCH_NAME="$BRANCH_NAME_PREFIX-$LATEST_DRAFT_VERSION"

EXISTS=$(git branch -r -l 'origin*' | sed -E -e 's/^[^\/]+\///g' -e 's/HEAD.+//' | grep "$BRANCH_NAME")

if [[ -n $EXISTS ]]; then
    echo "A PR already exists on branch $BRANCH_NAME for App Version update: $LATET_DRAFT_VERSION"
    exit 0
fi

echo "checking out new Git branch $BRANCH_NAME..."

git switch -c $BRANCH_NAME master

printf " done\n"

echo "Updating APP_VERSION..."

echo $LATEST_DRAFT_VERSION >APP_VERSION

echo "Committing changes..."

COMMIT_MSG="Update APP_VERSION to $LATEST_DRAFT_VERSION"

git add APP_VERSION

git commit -m "$COMMIT_MSG"

echo "Pushing to origin..."

git push -u origin $BRANCH_NAME

echo "Creating Pull Request..."

printf "This updates the App Version to $LATEST_DRAFT_VERSION" | gh pr create -R go-nv/goenv -B master \
    -t "$COMMIT_MSG" \
    -F -

echo 'All done!'
