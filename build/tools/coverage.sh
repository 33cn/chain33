#!/bin/bash
#
# Code coverage generation
set -e -o pipefail

COVERAGE_DIR="${COVERAGE_DIR:-build/coverage}"
#PKG_LIST=$(go list ./... | grep -v "vendor" | grep -v "chain33/test" | grep -v "mock" | grep -v "mocks" \
#    | grep -v "cmd" | grep -v "nat" | grep -v "pbft" |grep -v "common" |grep -v "system" |grep -v "wallet" |grep -v "blockchain" \
#    | grep -v "types" |grep -v "rpc" |grep -v "util" |grep -v "consensus" |grep -v "mempool"|grep -v "queue" |grep -v "store")

PKG_LIST=$(go list ./... | grep -E "account|client"|grep -v "cmd" |grep -v "rpc" | grep -v "mocks")

# Create the coverage files directory
mkdir -p "$COVERAGE_DIR"

# Create a coverage file for each package
for package in ${PKG_LIST}; do
    go test -covermode=count -coverprofile "${COVERAGE_DIR}/${package##*/}.cov" "$package"
done

# Merge the coverage profile files
echo 'mode: count' >./coverage.cov
tail -q -n +2 "${COVERAGE_DIR}"/*.cov >>./coverage.cov

# Display the global code coverage
go tool cover -func=./coverage.cov

# If needed, generate HTML report
if [ "$1" == "html" ]; then
    go tool cover -html=./coverage.cov -o coverage.html
fi

# Remove the coverage files directory
rm -rf "$COVERAGE_DIR"
