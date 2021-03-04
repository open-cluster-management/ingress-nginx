# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

# Release Tag
if [ "$TRAVIS_BRANCH" = "main" ]; then
    RELEASE_TAG=2.5.0
else
    RELEASE_TAG="${TRAVIS_BRANCH#release-}-latest"
fi
if [ "$TRAVIS_TAG" != "" ]; then
    RELEASE_TAG="${TRAVIS_TAG#v}"
fi
export RELEASE_TAG="$RELEASE_TAG"

# Release Tag
echo TRAVIS_EVENT_TYPE=$TRAVIS_EVENT_TYPE
echo TRAVIS_BRANCH=$TRAVIS_BRANCH
echo TRAVIS_TAG=$TRAVIS_TAG
echo RELEASE_TAG="$RELEASE_TAG"
