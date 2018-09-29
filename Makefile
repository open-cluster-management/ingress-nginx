include Configfile

.PHONY: build doc fmt lint run test vendor_clean vendor_get vendor_update vet

default: build

deps:
	go get github.com/golang/lint/golint
	go get -u github.com/apg/patter
	go get -u github.com/wadey/gocovmerge
	go get -u github.com/alecthomas/gometalinter
	gometalinter --install

lint:
	golint -set_exit_status=true pkg/
	golint -set_exit_status=true cmd/

build:
	go build -v -i -o bin/icp-management-ingress github.ibm.com/IBMPrivateCloud/icp-management-ingress/cmd/nginx

docker-binary:
	cp pkg/ingress/controller/config/config.go.no-fips pkg/ingress/controller/config/config.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -v -i -o rootfs/icp-management-ingress github.ibm.com/IBMPrivateCloud/icp-management-ingress/cmd/nginx
	strip rootfs/icp-management-ingress

image:: docker-binary

docker-binary-fips:
	cp pkg/ingress/controller/config/config.go.fips pkg/ingress/controller/config/config.go
	CGO_ENABLED=0 go build -a -installsuffix cgo -v -i -o rootfs/icp-management-ingress github.ibm.com/IBMPrivateCloud/icp-management-ingress/cmd/nginx
	strip rootfs/icp-management-ingress

image-fips:: docker-binary-fips

test:
	@./build/test.sh

coverage:
	go tool cover -html=cover.out -o=cover.html
	@./build/calculate-coverage.sh

fmt:
	gofmt -l ${GOFILES}

vet:
	gometalinter  --deadline=1000s --disable-all --enable=vet --enable=vetshadow --enable=ineffassign --enable=goconst --tests  --vendor ./...

include Makefile.docker
