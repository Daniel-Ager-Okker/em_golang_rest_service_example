#!/bin/bash
#
# Code coverage generation

COVERAGE_DIR="${COVERAGE_DIR:-coverage}"
PKG_LIST=$(go list ./... | grep -v -E '(^|/)(vendor|mocks|tests)(/|$)')

# Create the coverage files directory
mkdir -p "$COVERAGE_DIR";

# Create a coverage file for each package
for package in ${PKG_LIST}; do
    go test -covermode=count -coverprofile "${COVERAGE_DIR}/${package##*/}.cov" "$package" ;
done ;

COV_RESULT=coverage.res

# Merge the coverage profile files
echo 'mode: count' > "${COVERAGE_DIR}"/${COV_RESULT} ;
tail -q -n +2 "${COVERAGE_DIR}"/*.cov >> "${COVERAGE_DIR}"/${COV_RESULT} ;

# Display the global code coverage
go tool cover -func="${COVERAGE_DIR}"/${COV_RESULT} ;

go tool cover -html="${COVERAGE_DIR}"/${COV_RESULT} -o coverage.html ;

# Remove the coverage files directory
rm -rf "$COVERAGE_DIR";