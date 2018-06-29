#!/bin/bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2018. All Rights Reserved.
# U.S. Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP
# IBM Corporation - initial API and implementation

# NOTE: This script should not be called directly. Please run `make test`.

set -e

_package=$1
echo "Testing package $_package"

# Make sure temporary files do not exist
rm -f cover.tmp

# Run tests
# -coverpkg=./... produces warnings to stderr that we filter out
go test -v -cover -coverpkg=./... -covermode=atomic -coverprofile=cover.tmp $_package

# Merge coverage files
if [ -a cover.tmp ]; then
    $GOPATH/bin/gocovmerge cover.tmp cover.out > cover.all
    mv cover.all cover.out
fi

# Clean up temporary files
rm -f cover.tmp
