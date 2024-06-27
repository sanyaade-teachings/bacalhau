#!/usr/bin/env bash

# Exit on error. Append || true if you expect an error.
set -o errexit
# Exit on error inside any functions or subshells.
set -o errtrace
# Do not allow use of undefined vars. Use ${VAR:-} to use an undefined VAR
set -o nounset
# Catch the error in case mysqldump fails (but gzip succeeds) in `mysqldump |gzip`
set -o pipefail
# Turn on traces, useful while debugging but commented out by default
#set -o xtrace

# Function to find test files
find_test_files() {
  grep --exclude-dir='*vendor*' --include '*_test.go' -lR 'func Test[A-Z].*(t \*testing.T' ./* || {
    echo "Error: Failed to find test files."
    exit 1
  }
}

# Function to check for missing build headers
check_missing_headers() {
  local test_files="$1"
  grep --files-without-match -e '//go:build integration || !unit' -e '//go:build unit || !integration' ${test_files} || {
    echo "Error: Failed to check for missing build headers."
    exit 1
  }
}

# Main script execution
main() {
  local test_files
  test_files=$(find_test_files)

  if [[ -n "${test_files}" ]]; then
    local files_without_header
    files_without_header=$(check_missing_headers "${test_files}")

    if [[ -n "${files_without_header}" ]]; then
      printf "Test files missing '//go:build integration || !unit' or '//go:build unit || !integration':\n%s\n" "${files_without_header}"
      exit 1
    fi
  fi
}

main
