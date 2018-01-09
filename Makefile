.PHONY: build doc fmt lint run test vendor_clean vendor_get vendor_update vet

default: build

lint:
	@git diff-tree --check $(shell git hash-object -t tree /dev/null) HEAD $(shell ls -d * | grep -v vendor)

build:
	go build -v -i -o bin/icp-management-ingress github.ibm.com/IBMPrivateCloud/icp-management-ingress/cmd/nginx

docker-binary:
	CGO_ENABLED=0 go build -a -installsuffix cgo -v -i -o bin/icp-management-ingress github.ibm.com/IBMPrivateCloud/icp-management-ingress/cmd/nginx
	strip bin/icp-management-ingress

image:: docker-binary

test:
	go test -v -race $(shell go list github.ibm.com/IBMPrivateCloud/icp-management-ingress/... | grep -v vendor | grep -v '/test/e2e')
