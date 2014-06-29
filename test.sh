#!/bin/bash

# Runs the Agency tests. Intended for local and CI-server use.

# -----
# Setup
# -----

DEFAULT_JOB_NAME="tcpez-test"

if [ -z $JOB_NAME ]; then
  JOB_NAME=$DEFAULT_JOB_NAME
  echo "Warning: Setting JOB_NAME to default: $JOB_NAME"
fi

# To "sandbox" the packages from other Golang projects, install them within the
# workspace, not a globally shared folder.
export GOPATH="$HOME/${JOB_NAME}-pkg"

# Clean the installed packages so the latest versions available are always used.
echo "Cleaning packages"
rm -rf $GOPATH

# Ignore exit status to work around this bug:
#   https://code.google.com/p/go/issues/detail?id=5396
echo "Installing packages"
go get || true

echo "Installing test packages"
go get "github.com/bmizerany/assert"
go get "bitbucket.org/tebeka/go2xunit"

# ----
# Test
# ----

# Run the tests, capture the output, then parse the output with go2xunit:
#   https://bitbucket.org/tebeka/go2xunit
echo "Running tests"
OUTFILE="gotest.out"
go test -v | tee "${OUTFILE}"
"${GOPATH}"/bin/go2xunit -fail -input "${OUTFILE}" -output "gotest.xml"
