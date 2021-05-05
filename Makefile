# Copyright (c) 2021 Red Hat, Inc.
# Copyright Contributors to the Open Cluster Management project

include build/Configfile

REGISTRY ?= $(DOCKER_REGISTRY)/$(DOCKER_NAMESPACE)
COMMIT_SHA ?= git-$(shell git rev-parse --short HEAD)
IMAGE_TAG ?= $(COMMIT_SHA)

.PHONY: all
all: deps fmt lint coverage copyright-check vet image

.PHONY: deps
deps:
	go get golang.org/x/lint/golint
	go get -u github.com/apg/patter
	go get -u github.com/wadey/gocovmerge

.PHONY: image
image: docker-binary docker-image

.PHONY: docker-binary
docker-binary:
	CGO_ENABLED=0 go build -mod=mod -a -installsuffix cgo -v -i -o rootfs/management-ingress github.com/open-cluster-management/management-ingress/cmd/nginx
	strip rootfs/management-ingress

.PHONY: docker-image
docker-image:
	docker build -t $(REGISTRY)/$(IMAGE_NAME):$(IMAGE_TAG) --build-arg ARCH=$(ARCH_TYPE) .

.PHONY: fmt
fmt:
	gofmt -l ${GOFILES}

.PHONY: lint
lint:
	golint -set_exit_status=true pkg/
	golint -set_exit_status=true cmd/

.PHONY: test
test:
	@./build/test.sh

.PHONY: coverage
coverage:
	go tool cover -html=cover.out -o=cover.html
	@./build/calculate-coverage.sh

.PHONY: copyright-check
copyright-check:
	./build/copyright-check.sh $(TRAVIS_BRANCH)

.PHONY: vet
vet:
	gometalinter  --deadline=1000s --disable-all --enable=vet --enable=vetshadow --enable=ineffassign --enable=goconst --tests  --vendor ./...
