#!/usr/bin/env bats

load test_helper

@test "has usage instructions" {
  run goenv-help --usage installed
  assert_success_out <<OUT
Usage: goenv installed [<version>]
OUT
}

@test "installed has completion support" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  run goenv-installed --complete
  assert_success_out <<OUT
latest
system
1.10.9
1.9.10
OUT
}

@test "installed fails when no versions are installed" {
  run goenv-installed
  assert_failure "goenv: no versions installed"
}

@test "prints installed version when no arguments are given and there's a '.go-version' file in current dir" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-installed
  assert_success "1.2.3"
}

@test "prints installed version '.go-version' file in parent directory when no arguments are given" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  mkdir -p "subdir"
  cd "subdir"

  run goenv-installed
  assert_success "1.2.3"
}

@test "sets installed version when version argument is given and matches at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-installed 1.2.3
  assert_success "1.2.3"
}

@test "goenv installed sets properly sorted latest version when 'latest' version is given and any version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.10.10"
  mkdir -p "${GOENV_ROOT}/versions/1.10.9"
  mkdir -p "${GOENV_ROOT}/versions/1.9.10"
  mkdir -p "${GOENV_ROOT}/versions/1.9.9"
  run goenv-installed latest
  assert_success "1.10.10"
}

@test "goenv installed sets latest version when major version is given and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.6"
  run goenv-installed 1
  assert_success "1.2.10"
}

@test "goenv installed fails setting latest version when major or minor single number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/4.5.10"
  run goenv-installed 9
  assert_failure "goenv: version '9' not installed"
}

@test "goenv installed sets latest version when minor version is given as single number and any matching major.minor version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/4.5.2"
  run goenv-installed 2
  assert_success "1.2.10"
}

@test "goenv installed sets latest version when minor version is given as major.minor number and any matching version is installed" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.10"
  mkdir -p "${GOENV_ROOT}/versions/1.2.9"
  mkdir -p "${GOENV_ROOT}/versions/1.2.2"
  mkdir -p "${GOENV_ROOT}/versions/1.3.11"
  mkdir -p "${GOENV_ROOT}/versions/2.1.2"
  run goenv-installed 1.2
  assert_success "1.2.10"
}

@test "goenv installed fails setting latest version when major.minor number is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.1.9"
  run goenv-installed 1.9
  assert_failure "goenv: version '1.9' not installed"
}

@test "fails setting installed version when version argument is given and does not match at 'GOENV_ROOT/versions/<version>'" {
  mkdir -p "${GOENV_ROOT}/versions/1.2.3"
  run goenv-installed 1.2.4
  assert_failure "goenv: version '1.2.4' not installed"
}
