#!/bin/bash

# Licensed Materials - Property of IBM
# Copyright IBM Corporation 2018. All Rights Reserved.
# U.S. Government Users Restricted Rights -
# Use, duplication or disclosure restricted by GSA ADP
# IBM Corporation - initial API and implementation

COVERAGE=$(go tool cover -func=cover.out | grep "total:" | awk '{ print $3 }' | sed 's/[][()><%]/ /g')

echo "-------------------------------------------------------------------------"
echo "TOTAL COVERAGE IS ${COVERAGE}%"
echo "-------------------------------------------------------------------------"
