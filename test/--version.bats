#!/usr/bin/env bats

load test_helper

export GIT_DIR="${GOENV_TEST_DIR}/.git"

setup() {
  mkdir -p "$HOME"
  git config --global user.name  "Tester"
  git config --global user.email "tester@test.local"
  cd "$GOENV_TEST_DIR"
}

git_commit() {
  git commit --quiet --allow-empty -m "empty"
}

@test "default version" {
  assert [ ! -e "$GOENV_ROOT" ]
  run goenv---version
  assert_success
  [[ $output == "goenv 20"* ]]
}

@test "doesn't read version from non-goenv repo" {
  git init
  git remote add origin https://github.com/homebrew/homebrew.git
  git_commit
  git tag v1.0

  run goenv---version
  assert_success
  [[ $output == "goenv 20"* ]]
}

@test "reads version from git repo" {
  git init
  git remote add origin https://github.com/syndbg/goenv.git
  git_commit
  git tag v20380119
  git_commit
  git_commit

  run goenv---version
  assert_success "goenv 20160417"
}

@test "prints default version if no tags in git repo" {
  git init
  git remote add origin https://github.com/syndbg/goenv.git
  git_commit

  run goenv---version
  [[ $output == "goenv 20"* ]]
}
