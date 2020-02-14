#!/bin/bash
set -e

make docker-binary

echo "Building openresty image"
export OPENRESTY_VERSION=1.13.6.2
export DOCKER_IMAGE_AND_TAG=openresty:${OPENRESTY_VERSION}
export DOCKER_FILE=docker/openresty/${OPENRESTY_VERSION}/alpine/Dockerfile
make docker/build

echo "Building management-ingress image"
sed -i "s|BASE_IMAGE|$DOCKER_IMAGE_AND_TAG|g" Dockerfile
export DOCKER_IMAGE_AND_TAG=${1}
export DOCKER_FILE=Dockerfile
make docker/build