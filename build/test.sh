#!/bin/bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2018. All Rights Reserved.
# U.S. Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP
# IBM Corporation - initial API and implementation

set -e

_script_dir=$(dirname "$0")
echo 'mode: atomic' > cover.out
echo '' > cover.tmp
go list ./... | grep -v vendor | xargs -n1 -I{} $_script_dir/test-package.sh {}
